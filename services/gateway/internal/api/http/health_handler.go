package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
)

// HealthHandler serves health and readiness endpoints.
type HealthHandler struct {
	serviceName string
	readiness   ports.ReadinessChecker
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(serviceName string, readiness ports.ReadinessChecker) *HealthHandler {
	return &HealthHandler{serviceName: serviceName, readiness: readiness}
}

// Health returns liveness status.
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": h.serviceName})
}

// Ready returns readiness status.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{"gateway": "healthy"}
	if h.readiness != nil {
		for k, v := range h.readiness.Check(r.Context()) {
			checks[k] = v
		}
	}
	status := http.StatusOK
	for _, v := range checks {
		if v != "healthy" && v != "disabled" {
			status = http.StatusServiceUnavailable
			break
		}
	}
	writeJSON(w, status, map[string]any{"status": statusText(status), "checks": checks})
}

func statusText(code int) string {
	if code == http.StatusOK {
		return "ready"
	}
	return "not_ready"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
