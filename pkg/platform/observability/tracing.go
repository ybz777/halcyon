package observability

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider wraps the OTel SDK provider with shutdown semantics.
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	cfg      TracingConfig
}

// InitTracing configures global tracing from environment variables.
// When disabled, installs a no-op provider and returns a no-op shutdown function.
func InitTracing(ctx context.Context, serviceName string) (*TracerProvider, error) {
	cfg, err := LoadTracingConfig(serviceName)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return &TracerProvider{cfg: cfg}, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(trimScheme(cfg.OTLPEndpoint)),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"https://opentelemetry.io/schemas/1.26.0",
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", cfg.ServiceVersion),
			attribute.String("deployment.environment", cfg.Environment),
			attribute.String("service.instance.id", cfg.InstanceID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(cfg.BatchTimeout),
			sdktrace.WithMaxExportBatchSize(cfg.BatchExportMax),
			sdktrace.WithMaxQueueSize(cfg.MaxQueueSize),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(newSampler(cfg)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return &TracerProvider{provider: tp, cfg: cfg}, nil
}

// Shutdown flushes and closes the tracer provider.
func (p *TracerProvider) Shutdown(ctx context.Context) error {
	if p == nil || p.provider == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, p.cfg.ShutdownTimeout)
	defer cancel()
	return p.provider.Shutdown(ctx)
}

// Tracer returns a named tracer for the service.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Enabled reports whether tracing export is active.
func (p *TracerProvider) Enabled() bool {
	return p != nil && p.cfg.Enabled
}

func newSampler(cfg TracingConfig) sdktrace.Sampler {
	switch cfg.Sampler {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(cfg.SamplerArg)
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerArg))
	}
}

func trimScheme(endpoint string) string {
	return strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
}
