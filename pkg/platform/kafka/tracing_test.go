package kafka_test

import (
	"context"
	"testing"

	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInjectAndExtractTraceContext(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	ctx, span := tp.Tracer("test").Start(context.Background(), "producer")
	defer span.End()

	headers := map[string]string{}
	kafka.InjectTraceContext(ctx, headers)
	require.NotEmpty(t, headers["traceparent"])

	extracted := kafka.ExtractTraceContext(context.Background(), headers)
	require.NotEqual(t, context.Background(), extracted)
}
