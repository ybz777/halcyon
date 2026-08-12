package ratelimit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPMiddleware_RateLimited(t *testing.T) {
	limiter := NewMemoryLimiter()
	policies := PolicySet{
		API: Policy{Name: PolicyAPI, Limit: 1, Window: time.Minute},
	}
	handler := Middleware(HTTPOptions{
		Limiter: limiter,
		Enabled: true,
		Checks: func(r *http.Request) []DimensionCheck {
			return []DimensionCheck{{Identity: IdentityKey(IdentityIP, "127.0.0.1"), Policy: policies.API}}
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestBodyLimitMiddleware(t *testing.T) {
	handler := BodyLimitMiddleware(BodyLimitOptions{MaxBytes: 10})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("short"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("this payload is too large"))
	req.ContentLength = int64(len("this payload is too large"))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestBodyLimitMiddleware_MaxBytesReader(t *testing.T) {
	handler := RecoverBodyLimitMiddleware(BodyLimitOptions{MaxBytes: 5})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestMemoryLimiter_AllowAllRollback(t *testing.T) {
	limiter := NewMemoryLimiter()
	policy := Policy{Name: "api", Limit: 1, Window: time.Minute}
	ctx := context.Background()

	_, err := limiter.Allow(ctx, "a", policy)
	require.NoError(t, err)

	d, err := limiter.AllowAll(ctx, []string{"a", "b"}, policy)
	require.NoError(t, err)
	require.False(t, d.Allowed)

	// "b" must not retain a partial increment from the failed batch.
	d, err = limiter.Allow(ctx, "b", policy)
	require.NoError(t, err)
	require.True(t, d.Allowed)
}
