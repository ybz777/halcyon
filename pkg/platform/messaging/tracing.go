package messaging

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ContextWithTrace attaches trace context from envelope fields.
func ContextWithTrace(ctx context.Context, traceID, spanID string) context.Context {
	if traceID == "" {
		return ctx
	}

	carrier := propagation.MapCarrier{
		"traceparent": formatTraceParent(traceID, spanID),
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// TraceFieldsFromContext extracts trace identifiers for envelope propagation.
func TraceFieldsFromContext(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return "", ""
	}
	sc := span.SpanContext()
	return sc.TraceID().String(), sc.SpanID().String()
}

func formatTraceParent(traceID, spanID string) string {
	if len(traceID) != 32 {
		return ""
	}
	if spanID == "" {
		spanID = "0000000000000000"
	}
	if len(spanID) != 16 {
		spanID = padLeft(spanID, 16, '0')
	}
	return "00-" + traceID + "-" + spanID + "-01"
}

func padLeft(s string, n int, ch rune) string {
	for len(s) < n {
		s = string(ch) + s
	}
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}
