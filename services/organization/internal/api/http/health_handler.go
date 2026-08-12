package httpapi

import (
	"net/http"

	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type readyResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Checks  map[string]string `json:"checks"`
}

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	service   string
	readiness ports.ReadinessChecker
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(service string, readiness ports.ReadinessChecker) *HealthHandler {
	return &HealthHandler{service: service, readiness: readiness}
}

// Health handles liveness checks.
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok", Service: h.service})
}

// Ready handles readiness checks.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{"database": "skipped"}
	if h.readiness != nil {
		checks = h.readiness.Check(r.Context())
	}
	httputil.WriteJSON(w, http.StatusOK, readyResponse{Status: "ready", Service: h.service, Checks: checks})
}
