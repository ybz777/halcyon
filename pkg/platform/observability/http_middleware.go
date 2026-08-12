package observability

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HTTPTraceMiddleware creates server spans for inbound HTTP requests.
// RoutePattern should be the logical template (e.g. /api/v1/jobs/{id}).
func HTTPTraceMiddleware(serviceName, routePattern string) func(http.Handler) http.Handler {
	operation := "HTTP " + routePattern
	return func(next http.Handler) http.Handler {
		handler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		}), operation,
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return r.Method + " " + routePattern
			}),
			otelhttp.WithFilter(func(r *http.Request) bool {
				path := r.URL.Path
				return path != "/health" && path != "/ready" && path != "/metrics" &&
					path != "/api/v1/health" && path != "/api/v1/ready"
			}),
		)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
		})
	}
}

// RecordHandlerError marks the current span with a safe error status.
func RecordHandlerError(ctx context.Context, err error, category string) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, category)
	span.SetAttributes(attribute.String("error.category", category))
}

// StartSpan starts a child span with safe attributes only.
func StartSpan(ctx context.Context, tracer trace.Tracer, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// SafeHTTPAttributes returns bounded HTTP attributes without sensitive data.
func SafeHTTPAttributes(method, route string, status int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.status_code", status),
	}
}
