package idempotency_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/stretchr/testify/require"
)

func TestParseKeyValidation(t *testing.T) {
	t.Parallel()
	_, err := idempotency.ParseKey("")
	require.ErrorIs(t, err, idempotency.ErrKeyRequired)

	_, err = idempotency.ParseKey("bad key!")
	require.ErrorIs(t, err, idempotency.ErrKeyInvalid)

	key, err := idempotency.ParseKey("client-key-123")
	require.NoError(t, err)
	require.Equal(t, "client-key-123", key)
}

func TestFingerprintCanonicalJSON(t *testing.T) {
	t.Parallel()
	a := []byte(`{"b":2,"a":1}`)
	b := []byte("{\n  \"a\": 1,\n  \"b\": 2\n}")
	fp1, err := idempotency.Fingerprint(idempotency.FingerprintInput{Method: "POST", RoutePattern: "/x", Body: a})
	require.NoError(t, err)
	fp2, err := idempotency.Fingerprint(idempotency.FingerprintInput{Method: "POST", RoutePattern: "/x", Body: b})
	require.NoError(t, err)
	require.Equal(t, fp1, fp2)
}

func TestMemoryStoreClaimReplayConflict(t *testing.T) {
	t.Parallel()
	store := idempotency.NewMemoryStore()
	ctx := context.Background()
	out, err := store.Claim(ctx, "k1", "fp1", time.Minute)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimAcquired, out.Result)

	out, err = store.Claim(ctx, "k1", "fp1", time.Minute)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimInProgress, out.Result)

	require.NoError(t, store.Complete(ctx, "k1", "fp1", idempotency.Record{Status: 201, Body: []byte(`{"id":"1"}`), ContentType: "application/json"}, time.Hour))

	out, err = store.Claim(ctx, "k1", "fp1", time.Minute)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimReplay, out.Result)
	require.Equal(t, 201, out.Record.Status)

	out, err = store.Claim(ctx, "k1", "fp2", time.Minute)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimConflict, out.Result)
}

func TestMiddlewareConcurrentSingleExecution(t *testing.T) {
	store := idempotency.NewMemoryStore()
	var executions int32
	handler := idempotency.Middleware(idempotency.HTTPOptions{
		Store:  store,
		Config: idempotency.DefaultConfig("test"),
		Policies: idempotency.PolicyMap{
			"POST /mutate": idempotency.Required,
		},
		ResolveScope: func(r *http.Request) (idempotency.Scope, bool) {
			return idempotency.Scope{Service: "test", PrincipalType: "user", PrincipalID: "u1"}, true
		},
		RoutePattern: func(r *http.Request) string { return "/mutate" },
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&executions, 1)
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	const workers = 16
	var wg sync.WaitGroup
	statuses := make([]int, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader([]byte(`{"x":1}`)))
			req.Header.Set(idempotency.HeaderKey, "same-key")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			statuses[i] = rec.Code
		}(i)
	}
	wg.Wait()

	require.Equal(t, int32(1), atomic.LoadInt32(&executions))
	created := 0
	replayed := 0
	conflict := 0
	for _, code := range statuses {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusOK:
			replayed++
		case http.StatusConflict:
			conflict++
		}
	}
	require.Equal(t, 1, created+replayed)
	require.GreaterOrEqual(t, conflict+replayed, workers-1)
}

func TestMiddlewareFingerprintMismatch(t *testing.T) {
	store := idempotency.NewMemoryStore()
	handler := idempotency.Middleware(idempotency.HTTPOptions{
		Store:  store,
		Config: idempotency.DefaultConfig("test"),
		Policies: idempotency.PolicyMap{
			"POST /mutate": idempotency.Required,
		},
		ResolveScope: func(r *http.Request) (idempotency.Scope, bool) {
			return idempotency.Scope{Service: "test", PrincipalType: "user", PrincipalID: "u1"}, true
		},
		RoutePattern: func(r *http.Request) string { return "/mutate" },
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req1 := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader([]byte(`{"a":1}`)))
	req1.Header.Set(idempotency.HeaderKey, "k1")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusCreated, rec1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader([]byte(`{"a":2}`)))
	req2.Header.Set(idempotency.HeaderKey, "k1")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusConflict, rec2.Code)
}

func TestMiddlewareRequiredMissingKey(t *testing.T) {
	handler := idempotency.Middleware(idempotency.HTTPOptions{
		Store:  idempotency.NewMemoryStore(),
		Config: idempotency.DefaultConfig("test"),
		Policies: idempotency.PolicyMap{
			"POST /mutate": idempotency.Required,
		},
		ResolveScope: func(r *http.Request) (idempotency.Scope, bool) {
			return idempotency.Scope{Service: "test", PrincipalType: "user", PrincipalID: "u1"}, true
		},
		RoutePattern: func(r *http.Request) string { return "/mutate" },
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mutate", io.NopCloser(bytes.NewReader(nil)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
