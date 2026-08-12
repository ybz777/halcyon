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
	httpapi "github.com/halcyon-io/halcyon/services/organization/internal/api/http"
	httpmiddleware "github.com/halcyon-io/halcyon/services/organization/internal/api/http/middleware"
	apporg "github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	"github.com/halcyon-io/halcyon/services/organization/internal/config"
	infraevents "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/events"
	infrahealth "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/health"
	"github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/invitation"
	orgmetrics "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/postgres"
	jwtinfra "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/security/jwt"
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

	logger := observability.NewLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	ctx := context.Background()

	var pool *pgxpool.Pool
	apiEnabled := false
	var handler *httpapi.OrganizationHandler
	var validator *jwtinfra.Validator
	var resolver *httpmiddleware.MembershipResolver
	var authz *apporg.Authorizer

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
		}

		if err := postgres.Ping(ctx, pool); err != nil {
			return fmt.Errorf("validate database: %w", err)
		}

		validator, err = jwtinfra.NewValidator([]byte(cfg.JWTSecret), cfg.JWTIssuer, cfg.JWTAudience)
		if err != nil {
			return fmt.Errorf("create jwt validator: %w", err)
		}

		metrics := orgmetrics.New(cfg.ServiceName)
		orgRepo := postgres.NewOrganizationRepository(pool)
		memberRepo := postgres.NewMembershipRepository(pool)
		inviteRepo := postgres.NewInvitationRepository(pool)
		txManager := postgres.NewTransactionManager(pool)
		eventPublisher := infraevents.NewOutboxPublisher(pool)
		authz = apporg.NewAuthorizer(metrics)
		tokens := invitation.NewTokenService()
		resolver = httpmiddleware.NewMembershipResolver(memberRepo, orgRepo)

		handler = httpapi.NewOrganizationHandler(
			apporg.NewCreateOrganizationUseCase(orgRepo, memberRepo, txManager, eventPublisher, metrics),
			apporg.NewGetOrganizationUseCase(orgRepo, authz, metrics),
			apporg.NewUpdateOrganizationUseCase(orgRepo, txManager, authz, eventPublisher, metrics),
			apporg.NewDeleteOrganizationUseCase(orgRepo, txManager, authz, eventPublisher, metrics),
			apporg.NewListMembersUseCase(memberRepo, authz),
			apporg.NewUpdateMemberUseCase(memberRepo, authz, txManager, eventPublisher, metrics),
			apporg.NewRemoveMemberUseCase(memberRepo, authz, txManager, eventPublisher, metrics),
			apporg.NewInviteMemberUseCase(inviteRepo, authz, metrics, tokens, eventPublisher),
			apporg.NewAcceptInvitationUseCase(memberRepo, inviteRepo, orgRepo, txManager, tokens, eventPublisher, metrics),
			apporg.NewListInvitationsUseCase(inviteRepo, authz),
			apporg.NewRevokeInvitationUseCase(inviteRepo, authz, eventPublisher, metrics),
		)
		apiEnabled = true
	}

	var idemMiddleware func(http.Handler) http.Handler
	var redisClient *redis.Client
	if apiEnabled && cfg.IdempotencyEnabled {
		idemCfg, err := platformidempotency.LoadConfig(cfg.ServiceName)
		if err != nil {
			return fmt.Errorf("load idempotency config: %w", err)
		}
		idemCfg.Enabled = cfg.IdempotencyEnabled
		mw, client, err := platformidempotency.RedisMiddleware(ctx, cfg.RedisURL, idemCfg, platformidempotency.HTTPOptions{
			Metrics:      platformidempotency.NewMetrics(cfg.ServiceName),
			Policies:     httpapi.OrganizationPolicies,
			ResolveScope: httpapi.ResolveOrganizationScope,
		})
		if err != nil {
			logger.Warn("idempotency disabled; redis unavailable", "error", err)
		} else if mw != nil {
			idemMiddleware = mw
			redisClient = client
			defer redisClient.Close()
		}
	}

	router := httpapi.NewRouter(httpapi.RouterDeps{
		Logger:      logger,
		ServiceName: cfg.ServiceName,
		Readiness:   infrahealth.NewPostgresChecker(cfg.DatabaseURL, pool),
		Metrics:     observability.NewHTTPMetrics(cfg.ServiceName),
		Handler:     handler,
		Validator:   validator,
		Resolver:    resolver,
		Authorizer:  authz,
		APIEnabled:  apiEnabled,
		Idempotency: idemMiddleware,
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
		logger.Info("starting server", "service", cfg.ServiceName, "port", cfg.Port, "api_enabled", apiEnabled)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("listen and serve: %w", err)
	case <-stop:
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
