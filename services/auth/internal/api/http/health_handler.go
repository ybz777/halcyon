package httpapi

import (
	"net/http"

	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
)

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	serviceName string
	readiness   ports.ReadinessChecker
}

// NewHealthHandler creates a HealthHandler for the given service.
func NewHealthHandler(serviceName string, readiness ports.ReadinessChecker) *HealthHandler {
	return &HealthHandler{
		serviceName: serviceName,
		readiness:   readiness,
	}
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type readyResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Checks  map[string]string `json:"checks"`
}

// Health returns liveness status. Always 200 if the process is running.
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Service: h.serviceName,
	})
}

// Ready returns readiness status with explicit dependency checks.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, readyResponse{
		Status:  "ready",
		Service: h.serviceName,
		Checks:  h.readiness.Check(r.Context()),
	})
}
