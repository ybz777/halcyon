package httputil

import (
	"net/http"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// ServerTimeouts configures standard HTTP server timeouts.
type ServerTimeouts struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// DefaultServerTimeouts returns production-safe defaults for internal services.
func DefaultServerTimeouts() ServerTimeouts {
	return ServerTimeouts{
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// ApplyServerTimeouts sets timeout fields on an http.Server when values are positive.
func ApplyServerTimeouts(s *http.Server, t ServerTimeouts) {
	if s == nil {
		return
	}
	if t.ReadTimeout > 0 {
		s.ReadTimeout = t.ReadTimeout
	}
	if t.WriteTimeout > 0 {
		s.WriteTimeout = t.WriteTimeout
	}
	if t.IdleTimeout > 0 {
		s.IdleTimeout = t.IdleTimeout
	}
	if t.ReadHeaderTimeout > 0 {
		s.ReadHeaderTimeout = t.ReadHeaderTimeout
	}
}

// ApplyHTTPConfig applies centralized HTTP configuration to a server.
func ApplyHTTPConfig(s *http.Server, cfg platformconfig.HTTP) {
	ApplyServerTimeouts(s, ServerTimeouts{
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	})
}
