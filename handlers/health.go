// handlers/health.go
package handlers

import (
	"net/http"
)

// HealthCheck is a simple health check endpoint
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
