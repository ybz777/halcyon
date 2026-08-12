package proxy

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
	gwmiddleware "github.com/halcyon-io/halcyon/services/gateway/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
)

// Upstream proxies requests to a configured internal service.
type Upstream struct {
	name           string
	baseURL        *url.URL
	proxy          *httputil.ReverseProxy
	breaker        *CircuitBreaker
	signer         ports.ServiceTokenSigner
	serviceID      string
	metrics        ports.ProxyMetrics
}

// NewUpstream creates an upstream reverse proxy.
func NewUpstream(name, rawURL string, client *http.Client, breaker *CircuitBreaker, signer ports.ServiceTokenSigner, serviceID string, metrics ports.ProxyMetrics) (*Upstream, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	u := &Upstream{
		name:      name,
		baseURL:   target,
		breaker:   breaker,
		signer:    signer,
		serviceID: serviceID,
		metrics:   metrics,
	}
	u.proxy = &httputil.ReverseProxy{
		Director:       u.director,
		Transport:      client.Transport,
		ModifyResponse: u.modifyResponse,
		ErrorHandler:   u.errorHandler,
	}
	return u, nil
}

func (u *Upstream) director(req *http.Request) {
	req.URL.Scheme = u.baseURL.Scheme
	req.URL.Host = u.baseURL.Host
	req.Host = u.baseURL.Host
	req.URL.Path = singleJoiningSlash(u.baseURL.Path, req.URL.Path)
	req.URL.RawQuery = req.URL.Query().Encode()
}

func (u *Upstream) modifyResponse(resp *http.Response) error {
	return nil
}

func (u *Upstream) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if u.breaker != nil {
		u.breaker.RecordFailure()
	}
	if u.metrics != nil {
		u.metrics.RecordUpstreamError(u.name, "proxy_error")
	}
	slog.Warn("upstream proxy error", "service", u.name, "error", err, "request_id", middleware.GetReqID(r.Context()))
	http.Error(w, "upstream unavailable", http.StatusBadGateway)
}

// ServeHTTP proxies the request when the circuit allows.
func (u *Upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if u.breaker != nil && !u.breaker.Allow() {
		if u.metrics != nil {
			u.metrics.RecordUpstreamError(u.name, "circuit_open")
		}
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	if u.metrics != nil {
		u.metrics.IncInFlight(u.name, 1)
		defer u.metrics.IncInFlight(u.name, -1)
	}

	u.attachInternalHeaders(r)
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	u.proxy.ServeHTTP(rec, r)

	if u.breaker != nil {
		if rec.status >= 500 {
			u.breaker.RecordFailure()
		} else {
			u.breaker.RecordSuccess()
		}
	}
	if u.metrics != nil {
		u.metrics.RecordRequest(r.Method, r.URL.Path, u.name, http.StatusText(rec.status))
		u.metrics.ObserveDuration(r.Method, r.URL.Path, u.name, time.Since(start))
	}
}

func (u *Upstream) attachInternalHeaders(r *http.Request) {
	clientOrg := strings.TrimSpace(r.Header.Get("X-Halcyon-Organization-ID"))
	if clientOrg == "" {
		clientOrg = strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	}
	r.Header.Del("X-Organization-ID")
	r.Header.Del("X-Halcyon-Organization-ID")
	r.Header.Del(servicetoken.HeaderName)

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		r.Header.Set("X-Request-ID", reqID)
	}
	if corrID := contextx.CorrelationIDFromContext(r.Context()); corrID != "" {
		r.Header.Set("X-Correlation-ID", corrID)
	}
	if u.signer != nil {
		if token, err := u.signer.Sign(u.serviceID); err == nil {
			r.Header.Set(servicetoken.HeaderName, token)
		}
	}
	if principal, ok := gwmiddleware.PrincipalFromContext(r.Context()); ok && principal.UserID() != "" {
		r.Header.Set("X-Halcyon-User-ID", principal.UserID())
		if principal.Email() != "" {
			r.Header.Set("X-Halcyon-User-Email", principal.Email())
		}
		if clientOrg != "" {
			r.Header.Set("X-Halcyon-Organization-ID", clientOrg)
		}
	}
	if principal, ok := gwmiddleware.PrincipalFromContext(r.Context()); ok && principal.Type() == domain.PrincipalTypeAPIKey {
		if orgID := principal.OrganizationID(); orgID != "" {
			r.Header.Set("X-Halcyon-Organization-ID", orgID)
		}
		if id := principal.ID(); id != "" {
			r.Header.Set("X-Halcyon-API-Key-ID", id)
		}
		if scopes := principal.Scopes(); len(scopes) > 0 {
			r.Header.Set("X-Halcyon-API-Key-Scopes", strings.Join(scopes, ","))
		}
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// NewHTTPClient returns a pooled HTTP client for upstream calls.
func NewHTTPClient(timeout time.Duration, tlsInsecure bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = timeout
	if tlsInsecure {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

// DrainAndClose drains a response body for connection reuse.
func DrainAndClose(ctx context.Context, body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 1<<20))
	_ = body.Close()
}
