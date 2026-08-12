package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
)

// HealthHandler serves health endpoints.
type HealthHandler struct {
	serviceName string
	readiness   ports.ReadinessChecker
}

func NewHealthHandler(serviceName string, readiness ports.ReadinessChecker) *HealthHandler {
	return &HealthHandler{serviceName: serviceName, readiness: readiness}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": h.serviceName})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{"service": "ok"}
	if h.readiness != nil {
		for key, value := range h.readiness.Check(r.Context()) {
			checks[key] = value
		}
	}
	status := http.StatusOK
	for _, value := range checks {
		if value != "ok" {
			status = http.StatusServiceUnavailable
			break
		}
	}
	writeJSON(w, status, map[string]any{"status": checks})
}

func writeHealthJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
