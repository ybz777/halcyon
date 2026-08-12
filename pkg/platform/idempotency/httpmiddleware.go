package idempotency

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
)

// HTTPOptions configures idempotency HTTP middleware.
type HTTPOptions struct {
	Store          Store
	Config         Config
	Metrics        *Metrics
	Logger         *slog.Logger
	Policies       PolicyMap
	ResolveScope   func(r *http.Request) (Scope, bool)
	RequestID      func(r *http.Request) string
	RoutePattern   func(r *http.Request) string
}

// Middleware enforces idempotency for configured mutation routes.
func Middleware(opts HTTPOptions) func(http.Handler) http.Handler {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.RequestID == nil {
		opts.RequestID = func(r *http.Request) string { return r.Header.Get("X-Request-ID") }
	}
	if opts.RoutePattern == nil {
		opts.RoutePattern = routePatternFromContext
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !opts.Config.Enabled || opts.Store == nil {
				next.ServeHTTP(w, r)
				return
			}

			route := opts.RoutePattern(r)
			policy := opts.Policies.Lookup(r.Method, route)
			if policy == Disabled {
				next.ServeHTTP(w, r)
				return
			}

			rawKey := r.Header.Get(HeaderKey)
			if rawKey == "" {
				if policy == Required {
					writeProblem(w, opts.RequestID(r), ProblemKeyRequired, "Idempotency Key Required", http.StatusBadRequest, "Idempotency-Key header is required for this endpoint")
					recordMetric(opts, route, r.Method, policy, "key_required")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			key, err := ParseKey(rawKey)
			if err != nil {
				writeProblem(w, opts.RequestID(r), ProblemKeyInvalid, "Invalid Idempotency Key", http.StatusBadRequest, "Idempotency-Key header is invalid")
				recordMetric(opts, route, r.Method, policy, "key_invalid")
				return
			}

			scope, ok := opts.ResolveScope(r)
			if !ok {
				if opts.Config.FailClosed && policy == Required {
					writeProblem(w, opts.RequestID(r), ProblemStoreUnavailable, "Idempotency Unavailable", http.StatusServiceUnavailable, "unable to resolve idempotency scope")
					recordMetric(opts, route, r.Method, policy, "scope_missing")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			body, err := ReadBody(r, opts.Config.MaxBodyBytes)
			if err != nil {
				writeProblem(w, opts.RequestID(r), ProblemKeyInvalid, "Invalid Request", http.StatusBadRequest, err.Error())
				return
			}

			fp, err := Fingerprint(FingerprintInput{
				Method: r.Method, RoutePattern: route,
				PrincipalType: scope.PrincipalType, PrincipalID: scope.PrincipalID,
				OrganizationID: scope.OrganizationID, Query: r.URL.RawQuery, Body: body,
			})
			if err != nil {
				writeProblem(w, opts.RequestID(r), ProblemKeyInvalid, "Invalid Request", http.StatusBadRequest, "unable to fingerprint request")
				return
			}

			storageKey := StorageKey(opts.Config.RedisPrefix, scope, key)
			outcome, err := opts.Store.Claim(r.Context(), storageKey, fp, opts.Config.InProgressTTL)
			if err != nil {
				opts.Logger.Warn("idempotency store error", "route", route, "error", err)
				if opts.Metrics != nil {
					opts.Metrics.StoreErrorsTotal.Inc()
				}
				if opts.Config.FailClosed && policy == Required {
					writeProblem(w, opts.RequestID(r), ProblemStoreUnavailable, "Idempotency Unavailable", http.StatusServiceUnavailable, "idempotency store unavailable")
					recordMetric(opts, route, r.Method, policy, "store_error")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			switch outcome.Result {
			case ClaimReplay:
				if opts.Metrics != nil {
					opts.Metrics.ReplaysTotal.Inc()
				}
				recordMetric(opts, route, r.Method, policy, "replay")
				replayResponse(w, outcome.Record)
				return
			case ClaimConflict:
				if opts.Metrics != nil {
					opts.Metrics.ConflictsTotal.Inc()
				}
				recordMetric(opts, route, r.Method, policy, "conflict")
				writeProblem(w, opts.RequestID(r), ProblemConflict, "Idempotency Conflict", http.StatusConflict, "idempotency key was already used with a different request")
				return
			case ClaimInProgress:
				if opts.Metrics != nil {
					opts.Metrics.InProgressTotal.Inc()
				}
				recordMetric(opts, route, r.Method, policy, "in_progress")
				writeProblem(w, opts.RequestID(r), ProblemInProgress, "Idempotency In Progress", http.StatusConflict, "a request with this idempotency key is already being processed")
				return
			}

			capture := newCaptureWriter(w, opts.Config.MaxResponseBytes, true)
			next.ServeHTTP(capture, r)

			if isTransientStatus(capture.status) {
				_ = opts.Store.Release(r.Context(), storageKey, fp)
				recordMetric(opts, route, r.Method, policy, "transient")
				capture.flush()
				return
			}

			if shouldCacheResponse(capture.status) && int64(len(capture.body)) <= opts.Config.MaxResponseBytes {
				contentType := capture.header.Get("Content-Type")
				_ = opts.Store.Complete(r.Context(), storageKey, fp, Record{
					Status: capture.status, Body: append([]byte(nil), capture.body...), ContentType: contentType,
				}, opts.Config.CompletedTTL)
				recordMetric(opts, route, r.Method, policy, "completed")
			} else {
				_ = opts.Store.Release(r.Context(), storageKey, fp)
				recordMetric(opts, route, r.Method, policy, "not_cached")
			}

			w.Header().Set(HeaderReplay, "false")
			capture.flush()
		})
	}
}

func routePatternFromContext(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := strings.TrimSpace(rctx.RoutePattern()); pattern != "" {
			return pattern
		}
	}
	return r.URL.Path
}

func writeProblem(w http.ResponseWriter, instance, problemType, title string, status int, detail string) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type: problemType, Title: title, Status: status, Detail: detail, Instance: instance,
	})
}

func recordMetric(opts HTTPOptions, route, method string, policy Policy, result string) {
	if opts.Metrics != nil {
		opts.Metrics.Record(route, method, policy, result)
	}
}
