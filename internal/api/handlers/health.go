package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/PxPatel/trading-system/internal/api/models"
)

var startTime = time.Now()

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime)

	response := models.HealthResponse{
		Status:        "healthy",
		Timestamp:     time.Now().UTC(),
		UptimeSeconds: int64(uptime.Seconds()),
		Version:       "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
