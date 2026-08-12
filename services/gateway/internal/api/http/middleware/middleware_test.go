package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gwmiddleware "github.com/halcyon-io/halcyon/services/gateway/internal/api/http/middleware"
)

func TestRequestID_GeneratedWhenMissing(t *testing.T) {
	t.Parallel()
	handler := gwmiddleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") == "" {
			t.Fatal("expected request id header on request")
		}
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected response request id")
	}
}

func TestCorrelation_GeneratedWhenMissing(t *testing.T) {
	t.Parallel()
	handler := gwmiddleware.Correlation(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Correlation-ID") == "" {
		t.Fatal("expected correlation id")
	}
}
