// handlers/handlers.go
package handlers

import "net/http"

// Note: The main Handlers struct is now defined in payment_handlers.go
// This file can contain shared handler utilities

// HealthCheck is a simple health check endpoint
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}
