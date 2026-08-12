package observability

import (
	"context"
	"log/slog"

	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
	"go.opentelemetry.io/otel/trace"
)

// TraceLogHandler enriches slog records with trace and request correlation fields.
type TraceLogHandler struct {
	next slog.Handler
}

// NewTraceLogHandler wraps a slog handler with trace correlation.
func NewTraceLogHandler(next slog.Handler) *TraceLogHandler {
	return &TraceLogHandler{next: next}
}

func (h *TraceLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *TraceLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	if id := contextx.RequestIDFromContext(ctx); id != "" {
		record.AddAttrs(slog.String("request_id", id))
	}
	if id := contextx.CorrelationIDFromContext(ctx); id != "" {
		record.AddAttrs(slog.String("correlation_id", id))
	}
	if id := contextx.CausationIDFromContext(ctx); id != "" {
		record.AddAttrs(slog.String("causation_id", id))
	}
	return h.next.Handle(ctx, record)
}

func (h *TraceLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceLogHandler{next: h.next.WithAttrs(attrs)}
}

func (h *TraceLogHandler) WithGroup(name string) slog.Handler {
	return &TraceLogHandler{next: h.next.WithGroup(name)}
}

// NewLoggerWithTracing creates a logger with trace correlation support.
func NewLoggerWithTracing(level, format string) *slog.Logger {
	base := NewLogger(level, format)
	return slog.New(NewTraceLogHandler(base.Handler()))
}
