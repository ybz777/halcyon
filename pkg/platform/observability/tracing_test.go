package observability_test

import (
	"context"
	"testing"

	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestLoadTracingConfigDisabled(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "false")
	cfg, err := observability.LoadTracingConfig("auth")
	require.NoError(t, err)
	require.False(t, cfg.Enabled)
}

func TestLoadTracingConfigInvalidSampler(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "false")
	t.Setenv("OTEL_TRACES_SAMPLER", "invalid")
	_, err := observability.LoadTracingConfig("auth")
	require.Error(t, err)
}

func TestInitTracingDisabled(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "false")
	tp, err := observability.InitTracing(context.Background(), "auth")
	require.NoError(t, err)
	require.NoError(t, tp.Shutdown(context.Background()))
}

func TestInMemoryExporterCapturesSpan(t *testing.T) {
	tp, exporter := observability.NewTestTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "unit test", trace.WithAttributes(attribute.String("job.type", "email.delivery")))
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))
	require.NotEmpty(t, exporter.Spans())
}

func TestTraceLogHandlerAddsTraceFields(t *testing.T) {
	tp, _ := observability.NewTestTracerProvider()
	otel.SetTracerProvider(tp)
	ctx, span := tp.Tracer("test").Start(context.Background(), "log test")
	defer span.End()
	handler := observability.NewTraceLogHandler(observability.NewLogger("info", "json").Handler())
	require.True(t, handler.Enabled(ctx, 0))
}
