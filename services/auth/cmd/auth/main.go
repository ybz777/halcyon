package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	platformidempotency "github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	platformratelimit "github.com/halcyon-io/halcyon/pkg/platform/ratelimit"
	httpapi "github.com/halcyon-io/halcyon/services/auth/internal/api/http"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/config"
	authinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/auth"
	infraevents "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/events"
	infrahealth "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/health"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres"
	authratelimit "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/ratelimit"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/argon2"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
	jwksinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwks"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := context.Background()

	tracer, logger, err := observability.Bootstrap(
		ctx,
		cfg.ServiceName,
		cfg.LogLevel,
		cfg.LogFormat,
	)
	if err != nil {
		return fmt.Errorf("bootstrap observability: %w", err)
	}

	var pool *pgxpool.Pool

	if cfg.DatabaseURL != "" {
		pool, err = postgres.NewPool(ctx, postgres.PoolConfig{
			URL:             cfg.DatabaseURL,
			MaxConns:        cfg.DatabaseMaxConns,
			MinConns:        cfg.DatabaseMinConns,
			MaxConnLifetime: cfg.DatabaseConnMaxLife,
			MaxConnIdleTime: cfg.DatabaseConnMaxIdle,
		})
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		defer pool.Close()

		if cfg.DatabaseMigrate {
			if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
				return fmt.Errorf("run migrations: %w", err)
			}

			if err := migrations.Run(cfg.DatabaseURL); err != nil {
				return fmt.Errorf("run platform migrations: %w", err)
			}

			logger.Info("database migrations applied")
		}

		if err := postgres.Ping(ctx, pool); err != nil {
			return fmt.Errorf("validate database: %w", err)
		}

		logger.Info("database connection validated")
	}

	readiness := infrahealth.NewPostgresChecker(cfg.DatabaseURL, pool)
	metrics := observability.NewHTTPMetrics(cfg.ServiceName)

	rlCfg, err := platformratelimit.LoadConfig(cfg.ServiceName)
	if err != nil {
		return fmt.Errorf("load rate limit config: %w", err)
	}

	rlCfg.Enabled = cfg.RateLimitEnabled
	rlCfg.RedisURL = cfg.RedisURL

	rlMetrics := platformratelimit.NewMetrics(cfg.ServiceName)

	var loginLimiter ports.LoginRateLimiter = ports.NoopLoginRateLimiter{}
	var apiKeyLimiter ports.APIKeyRateLimiter = ports.NoopAPIKeyRateLimiter{}

	var redisClient *redis.Client

	if rlCfg.Enabled || cfg.IdempotencyEnabled {
		if cfg.RedisURL == "" {
			return fmt.Errorf("REDIS_URL is required when rate limiting or idempotency is enabled")
		}

		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return fmt.Errorf("parse redis url: %w", err)
		}

		redisClient = redis.NewClient(opt)

		if err := redisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("ping redis: %w", err)
		}

		defer redisClient.Close()
	}

	if rlCfg.Enabled && redisClient != nil {
		limiter := platformratelimit.NewRedisLimiter(
			redisClient,
			rlCfg,
			logger,
			rlMetrics,
		)

		policies := platformratelimit.AuthPolicySet(rlCfg)

		loginLimiter = authratelimit.NewLoginAdapter(
			limiter,
			policies.Auth,
		)

		apiKeyLimiter = authratelimit.NewAPIKeyAdapter(
			limiter,
			policies.API,
		)
	}

	var authHandler *httpapi.AuthHandler
	var loginHandler *httpapi.LoginHandler
	var registerHandler *httpapi.RegisterHandler
	var refreshHandler *httpapi.RefreshHandler
	var apiKeyHandler *httpapi.APIKeyHandler
	var introspectHandler *httpapi.IntrospectAPIKeyHandler
	var validateToken *appauth.ValidateTokenUseCase
	var jwksHandler http.Handler

	authEnabled := false

	if pool != nil {
		dbMetrics := postgres.NewMetrics(
			cfg.ServiceName,
			func() int32 {
				return pool.Stat().AcquiredConns()
			},
		)

		userRepo := postgres.NewUserRepository(pool, dbMetrics)
		roleRepo := postgres.NewUserRoleRepository(pool)
		refreshRepo := postgres.NewRefreshTokenRepository(pool)

		hasher := argon2.NewHasher(argon2.Params{
			Memory:      cfg.Argon2Memory,
			Iterations:  cfg.Argon2Iterations,
			Parallelism: cfg.Argon2Parallelism,
			SaltLength:  cfg.Argon2SaltLength,
			KeyLength:   cfg.Argon2KeyLength,
		})

		jwtService, err := jwtinfra.NewService(
			[]byte(cfg.JWTSecret),
			cfg.ServiceName,
			cfg.JWTAccessTTL,
			cfg.JWTRefreshTTL,
		)
		if err != nil {
			return fmt.Errorf("create jwt service: %w", err)
		}

		authMetrics := authinfra.NewMetrics(cfg.ServiceName)
		txManager := postgres.NewTransactionManager(pool)
		eventPublisher := infraevents.NewOutboxPublisher(pool)

		registerUC := appauth.NewRegisterUserHandler(
			userRepo,
			roleRepo,
			hasher,
			txManager,
			eventPublisher,
		)

		loginUC := appauth.NewLoginUseCase(
			userRepo,
			roleRepo,
			refreshRepo,
			hasher,
			jwtService,
			jwtService,
			loginLimiter,
			authMetrics,
			txManager,
			eventPublisher,
		)

		refreshUC := appauth.NewRefreshTokenUseCase(
			userRepo,
			roleRepo,
			refreshRepo,
			jwtService,
			jwtService,
			authMetrics,
		)

		logoutUC := appauth.NewLogoutUseCase(
			refreshRepo,
			jwtService,
			authMetrics,
		)

		validateToken = appauth.NewValidateTokenUseCase(
			jwtService,
			authMetrics,
		)

		jwksProvider := jwksinfra.NewProvider(
			cfg.JWTKeyID,
			[]byte(cfg.JWTSecret),
		)

		jwksHandler = httpapi.NewJWKSHandler(jwksProvider)

		apiKeyHasher := argon2.NewAPIKeyHasher(hasher)

		serviceAccountRepo := postgres.NewServiceAccountRepository(pool)
		apiKeyRepo := postgres.NewAPIKeyRepository(pool)

		createAPIKey := appauth.NewCreateAPIKeyHandler(
			apiKeyRepo,
			serviceAccountRepo,
			apiKeyHasher,
			authMetrics,
			txManager,
			eventPublisher,
		)

		listAPIKeys := appauth.NewListAPIKeysHandler(
			apiKeyRepo,
		)

		getAPIKey := appauth.NewGetAPIKeyHandler(
			apiKeyRepo,
		)

		revokeAPIKey := appauth.NewRevokeAPIKeyHandler(
			apiKeyRepo,
			serviceAccountRepo,
			txManager,
			eventPublisher,
			authMetrics,
		)

		rotateAPIKey := appauth.NewRotateAPIKeyHandler(
			apiKeyRepo,
			serviceAccountRepo,
			apiKeyHasher,
			eventPublisher,
			authMetrics,
		)

		authenticateAPIKey := appauth.NewAuthenticateAPIKeyHandler(
			apiKeyRepo,
			serviceAccountRepo,
			apiKeyHasher,
			apiKeyLimiter,
			authMetrics,
			eventPublisher,
		)

		introspectHandler = httpapi.NewIntrospectAPIKeyHandler(
			authenticateAPIKey,
		)

		authHandler = httpapi.NewAuthHandler(logoutUC)
		loginHandler = httpapi.NewLoginHandler(loginUC)
		registerHandler = httpapi.NewRegisterHandler(registerUC)
		refreshHandler = httpapi.NewRefreshHandler(refreshUC)

		apiKeyHandler = httpapi.NewAPIKeyHandler(
			createAPIKey,
			listAPIKeys,
			getAPIKey,
			revokeAPIKey,
			rotateAPIKey,
		)

		authEnabled = true
	}

	var idemMiddleware func(http.Handler) http.Handler

	if authEnabled && cfg.IdempotencyEnabled && redisClient != nil {
		idemCfg, err := platformidempotency.LoadConfig(cfg.ServiceName)
		if err != nil {
			return fmt.Errorf("load idempotency config: %w", err)
		}

		idemCfg.Enabled = true

		idemMiddleware = platformidempotency.Middleware(
			platformidempotency.HTTPOptions{
				Store:        platformidempotency.NewRedisStore(redisClient),
				Config:       idemCfg,
				Metrics:      platformidempotency.NewMetrics(cfg.ServiceName),
				Policies:     httpapi.AuthPolicies,
				ResolveScope: httpapi.ResolveAuthScope,
			},
		)
	}

	router := httpapi.NewRouter(httpapi.RouterDeps{
		Logger:        logger,
		ServiceName:   cfg.ServiceName,
		Readiness:     readiness,
		Metrics:       metrics,
		Login:         loginHandler,
		Register:      registerHandler,
		Refresh:       refreshHandler,
		Auth:          authHandler,
		APIKeys:       apiKeyHandler,
		Introspect:    introspectHandler,
		ValidateToken: validateToken,
		JWKS:          jwksHandler,
		AuthEnabled:   authEnabled,
		Idempotency:   idemMiddleware,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info(
			"starting server",
			"service", cfg.ServiceName,
			"port", cfg.Port,
			"auth_enabled", authEnabled,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("listen and serve: %w", err)

	case sig := <-stop:
		logger.Info(
			"shutdown signal received",
			"signal", sig.String(),
		)
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if tracer != nil {
		_ = tracer.Shutdown(shutdownCtx)
	}

	logger.Info("server stopped")

	return nil
}