package kafka

import (
	"context"

	"go.opentelemetry.io/otel"
)

// HeaderCarrier adapts Kafka headers for W3C trace propagation.
type HeaderCarrier map[string]string

func (c HeaderCarrier) Get(key string) string { return c[key] }
func (c HeaderCarrier) Set(key, value string) { c[key] = value }
func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectTraceContext writes W3C trace context into a header map.
func InjectTraceContext(ctx context.Context, headers map[string]string) {
	if headers == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, HeaderCarrier(headers))
}

// ExtractTraceContext returns a context with propagated trace information.
func ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, HeaderCarrier(headers))
}

// HeadersFromMessage builds a string map from Kafka message headers.
func HeadersFromMessage(msg Message) map[string]string {
	out := map[string]string{}
	if msg.TraceID != "" {
		out["trace_id"] = msg.TraceID
	}
	if msg.SpanID != "" {
		out["span_id"] = msg.SpanID
	}
	if msg.RequestID != "" {
		out["request_id"] = msg.RequestID
	}
	if msg.CorrelationID != "" {
		out["correlation_id"] = msg.CorrelationID
	}
	return out
}
