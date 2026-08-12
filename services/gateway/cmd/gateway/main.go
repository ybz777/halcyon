package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	platformratelimit "github.com/halcyon-io/halcyon/pkg/platform/ratelimit"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
	httpapi "github.com/halcyon-io/halcyon/services/gateway/internal/api/http"
	"github.com/halcyon-io/halcyon/services/gateway/internal/config"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/apikey"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/health"
	jwksclient "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/jwks"
	jwtinfra "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/jwt"
	gwmetrics "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/proxy"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/securityevents"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		slog.Error("gateway failed", "error", err)
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
	metrics := gwmetrics.NewGatewayMetrics()
	httpMetrics := observability.NewHTTPMetrics(cfg.ServiceName)
	rlMetrics := platformratelimit.NewMetrics(cfg.ServiceName)

	jwks := jwksclient.NewClient(cfg.JWKSURL, cfg.JWSCacheTTL, &http.Client{Timeout: 5 * time.Second}, metrics.RecordJWKSRefresh)
	if err := jwks.Refresh(ctx); err != nil {
		return fmt.Errorf("initial jwks refresh: %w", err)
	}

	validator := jwtinfra.NewValidator(cfg.JWTIssuer, cfg.JWTAudience, jwks)
	signer, err := servicetoken.NewSigner(cfg.ServicePrivateKey, cfg.ServiceKeyID, cfg.ServiceIdentity, cfg.ServiceTokenAudience, cfg.ServiceTokenTTL)
	if err != nil {
		return fmt.Errorf("create service token signer: %w", err)
	}

	rlCfg := gatewayRateLimitConfig(cfg)
	var redisClient *redis.Client
	var limiter platformratelimit.Limiter = platformratelimit.DisabledLimiter{}
	var concurrency *platformratelimit.ConcurrencyLimiter
	var readiness *health.RedisChecker
	if cfg.RateLimitEnabled {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return fmt.Errorf("parse redis url: %w", err)
		}
		redisClient = redis.NewClient(opt)
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("ping redis: %w", err)
		}
		limiter = platformratelimit.NewRedisLimiter(redisClient, rlCfg, logger, rlMetrics)
		concurrency = platformratelimit.NewConcurrencyLimiter(redisClient, rlCfg.RedisPrefix, cfg.WorkflowConcurrency, 5*time.Minute, logger, rlMetrics)
		readiness = health.NewRedisChecker(redisClient)
	}

	client := proxy.NewHTTPClient(cfg.UpstreamTimeout, cfg.UpstreamTLSInsecure)
	upstreamURLs := map[string]string{
		"auth":         cfg.AuthServiceURL,
		"organization": cfg.OrgServiceURL,
	}
	if cfg.AuditServiceURL != "" {
		upstreamURLs["audit"] = cfg.AuditServiceURL
	}
	if cfg.IntegrationsServiceURL != "" {
		upstreamURLs["integrations"] = cfg.IntegrationsServiceURL
	}
	if cfg.NotificationsServiceURL != "" {
		upstreamURLs["notifications"] = cfg.NotificationsServiceURL
	}
	if cfg.JobsServiceURL != "" {
		upstreamURLs["jobs"] = cfg.JobsServiceURL
	}
	upstreams := make(map[string]*proxy.Upstream)
	for name, rawURL := range upstreamURLs {
		breaker := proxy.NewCircuitBreaker(cfg.CircuitFailureThreshold, cfg.CircuitOpenDuration, func() {
			metrics.RecordCircuitOpen(name)
		})
		if !cfg.CircuitBreakerEnabled {
			breaker = nil
		}
		upstream, err := proxy.NewUpstream(name, rawURL, client, breaker, signer, cfg.ServiceIdentity, metrics)
		if err != nil {
			return fmt.Errorf("create upstream %s: %w", name, err)
		}
		upstreams[name] = upstream
	}

	var kafkaProducer kafka.Producer
	if cfg.KafkaBrokers != "" {
		brokers := splitCSV(cfg.KafkaBrokers)
		kafkaProducer = kafka.NewWriterProducer(kafka.Config{Brokers: brokers, ClientID: "gateway"})
		defer kafkaProducer.Close()
	}
	securityPublisher := securityevents.NewPublisher(kafkaProducer)
	apiKeyAuth := apikey.NewPortAdapter(apikey.NewAuthenticator(cfg.AuthServiceURL))

	router := httpapi.NewRouter(httpapi.RouterDeps{
		Logger:             logger,
		ServiceName:        cfg.ServiceName,
		Config:             cfg,
		Readiness:          readiness,
		Metrics:            httpMetrics,
		ProxyMetrics:       metrics,
		Validator:          validator,
		APIKeys:            apiKeyAuth,
		RateLimiter:        limiter,
		RateLimitMetrics:   rlMetrics,
		RateLimitPolicy:    platformratelimit.GatewayPolicySet(rlCfg),
		ConcurrencyLimiter: concurrency,
		Security:           securityPublisher,
		Upstreams:          upstreams,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  90 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting gateway", "port", cfg.Port, "rate_limit_enabled", cfg.RateLimitEnabled)
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
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if redisClient != nil {
		_ = redisClient.Close()
	}
	return nil
}

func gatewayRateLimitConfig(cfg config.Config) platformratelimit.Config {
	failureMode := platformratelimit.FailureModeOpen
	if cfg.RateLimitFailClosed || strings.EqualFold(cfg.RateLimitFailureMode, "closed") {
		failureMode = platformratelimit.FailureModeClosed
	}
	return platformratelimit.Config{
		Enabled:             cfg.RateLimitEnabled,
		RedisURL:            cfg.RedisURL,
		RedisPrefix:         cfg.RateLimitRedisPrefix,
		DefaultRequests:     cfg.RateLimitDefault,
		DefaultWindow:       cfg.RateLimitWindow,
		AuthRequests:        cfg.RateLimitAuth,
		AuthWindow:          cfg.RateLimitAuthWindow,
		WorkflowRequests:    cfg.RateLimitWorkflow,
		WorkflowWindow:      cfg.RateLimitWorkflowWindow,
		MaxBodyBytes:        cfg.MaxRequestBodyBytes,
		WorkflowConcurrency: cfg.WorkflowConcurrency,
		FailureMode:         failureMode,
		TrustProxy:          cfg.RateLimitTrustProxy,
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
