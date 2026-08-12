package ratelimit

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
)

// DimensionCheck binds an identity to a policy.
type DimensionCheck struct {
	Identity string
	Policy   Policy
}

// HTTPOptions configures HTTP rate-limit middleware.
type HTTPOptions struct {
	Limiter     Limiter
	Enabled     bool
	TrustProxy  bool
	Metrics     *Metrics
	Logger      *slog.Logger
	RouteLabel  func(r *http.Request) string
	Skip        func(r *http.Request) bool
	Checks      func(r *http.Request) []DimensionCheck
	OnRejected  func(r *http.Request, policy Policy, decision Decision)
	RequestID   func(r *http.Request) string
}

// Middleware enforces distributed rate limits for HTTP handlers.
func Middleware(opts HTTPOptions) func(http.Handler) http.Handler {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.RouteLabel == nil {
		opts.RouteLabel = func(r *http.Request) string { return normalizeRoute(r.URL.Path) }
	}
	if opts.RequestID == nil {
		opts.RequestID = func(r *http.Request) string { return r.Header.Get("X-Request-ID") }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !opts.Enabled || opts.Limiter == nil || opts.Checks == nil {
				next.ServeHTTP(w, r)
				return
			}
			if opts.Skip != nil && opts.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			checks := opts.Checks(r)
			route := opts.RouteLabel(r)
			for _, check := range checks {
				if strings.TrimSpace(check.Identity) == "" {
					continue
				}
				decision, err := opts.Limiter.Allow(r.Context(), check.Identity, check.Policy)
				if err != nil && decision.Allowed {
					opts.Logger.WarnContext(r.Context(), "rate limit check degraded",
						slog.String("event", "rate_limit.fail_open"),
						slog.String("policy", check.Policy.Name),
						slog.String("route", route),
						slog.String("method", r.Method),
					)
				}
				if opts.Metrics != nil {
					opts.Metrics.Record(route, r.Method, check.Policy.Name, decision.Allowed)
				}
				if !decision.Allowed {
					if opts.OnRejected != nil {
						opts.OnRejected(r, check.Policy, decision)
					}
					opts.Logger.InfoContext(r.Context(), "rate limit rejected",
						slog.String("event", "security.rate_limited"),
						slog.String("policy", check.Policy.Name),
						slog.String("route", route),
						slog.String("method", r.Method),
						slog.String("decision", "rejected"),
						slog.Duration("retry_after", decision.RetryAfter),
					)
					writeRateLimited(w, opts.RequestID(r), decision)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ConcurrencyMiddleware limits concurrent expensive operations.
func ConcurrencyMiddleware(limiter *ConcurrencyLimiter, enabled bool, resolve func(*http.Request) (string, bool), metrics *Metrics, requestID func(*http.Request) string) func(http.Handler) http.Handler {
	if requestID == nil {
		requestID = func(r *http.Request) string { return r.Header.Get("X-Request-ID") }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled || limiter == nil || resolve == nil {
				next.ServeHTTP(w, r)
				return
			}
			identity, ok := resolve(r)
			if !ok || identity == "" {
				next.ServeHTTP(w, r)
				return
			}
			allowed, retryAfter, err := limiter.Acquire(r.Context(), identity)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				if metrics != nil {
					metrics.ConcurrencyRejected.Inc()
				}
				writeRateLimited(w, requestID(r), Decision{Allowed: false, RetryAfter: retryAfter})
				return
			}
			defer func() { _ = limiter.Release(r.Context(), identity) }()
			next.ServeHTTP(w, r)
		})
	}
}

func writeRateLimited(w http.ResponseWriter, requestID string, decision Decision) {
	retryAfter := decision.RetryAfter
	if retryAfter <= 0 {
		retryAfter = time.Second
	}
	seconds := int(retryAfter.Round(time.Second).Seconds())
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(decision.Remaining, 10))
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type:     "https://halcyon.io/problems/rate-limited",
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   "rate limit exceeded",
		Instance: requestID,
	})
}

func normalizeRoute(path string) string {
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return "/"
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if isUUIDLike(part) {
			parts[i] = "{id}"
		}
	}
	return "/" + strings.Join(parts, "/")
}

func isUUIDLike(v string) bool {
	if len(v) < 8 {
		return false
	}
	for _, ch := range v {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') || ch == '-' {
			continue
		}
		return false
	}
	return strings.Count(v, "-") >= 2
}

// BodyLimitOptions configures request body size enforcement.
type BodyLimitOptions struct {
	MaxBytes int64
	Metrics  *Metrics
	RequestID func(r *http.Request) string
}

// BodyLimitMiddleware rejects oversized request bodies.
func BodyLimitMiddleware(opts BodyLimitOptions) func(http.Handler) http.Handler {
	if opts.RequestID == nil {
		opts.RequestID = func(r *http.Request) string { return r.Header.Get("X-Request-ID") }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength > opts.MaxBytes {
				if opts.Metrics != nil {
					opts.Metrics.BodyRejectedTotal.Inc()
				}
				httperrors.WriteProblem(w, httperrors.ProblemDetails{
					Type:     "https://halcyon.io/problems/payload-too-large",
					Title:    "Payload Too Large",
					Status:   http.StatusRequestEntityTooLarge,
					Detail:   "request body exceeds maximum allowed size",
					Instance: opts.RequestID(r),
				})
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, opts.MaxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RecoverBodyLimitMiddleware enforces body limits and converts MaxBytesReader panics into RFC 7807 responses.
func RecoverBodyLimitMiddleware(opts BodyLimitOptions) func(http.Handler) http.Handler {
	if opts.RequestID == nil {
		opts.RequestID = func(r *http.Request) string { return r.Header.Get("X-Request-ID") }
	}
	return func(next http.Handler) http.Handler {
		limited := BodyLimitMiddleware(opts)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if _, ok := rec.(error); ok {
						if opts.Metrics != nil {
							opts.Metrics.BodyRejectedTotal.Inc()
						}
						httperrors.WriteProblem(w, httperrors.ProblemDetails{
							Type:     "https://halcyon.io/problems/payload-too-large",
							Title:    "Payload Too Large",
							Status:   http.StatusRequestEntityTooLarge,
							Detail:   "request body exceeds maximum allowed size",
							Instance: opts.RequestID(r),
						})
						return
					}
					panic(rec)
				}
			}()
			limited.ServeHTTP(w, r)
		})
	}
}
