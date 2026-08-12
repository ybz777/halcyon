package observability

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// InMemoryExporter stores spans for tests.
type InMemoryExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func NewInMemoryExporter() *InMemoryExporter {
	return &InMemoryExporter{}
}

func (e *InMemoryExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *InMemoryExporter) Shutdown(context.Context) error { return nil }

func (e *InMemoryExporter) Spans() []sdktrace.ReadOnlySpan {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]sdktrace.ReadOnlySpan, len(e.spans))
	copy(out, e.spans)
	return out
}

// NewTestTracerProvider installs an in-memory exporter for tests.
func NewTestTracerProvider() (*sdktrace.TracerProvider, *InMemoryExporter) {
	exporter := NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	return tp, exporter
}

// RecordSpanError records a safe error on a span.
func RecordSpanError(span trace.Span, err error, category string) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, category)
	span.SetAttributes(attribute.String("error.category", category))
}
