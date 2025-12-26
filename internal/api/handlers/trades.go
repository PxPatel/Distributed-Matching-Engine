package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/PxPatel/trading-system/internal/api/logger"
	"github.com/PxPatel/trading-system/internal/api/models"
)

// GetTradesHandler handles retrieving recent trades
func (eh *EngineHolder) GetTradesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")

	// Default limit: 100, max: 1000
	limit := 100
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	// Get recent trades from engine
	trades := eh.Engine.GetRecentTrades(limit)

	// Convert to DTOs
	tradeDTOs := convertTradesToDTO(trades)

	logger.Info("Retrieved trades", map[string]interface{}{
		"count": len(tradeDTOs),
		"limit": limit,
	})

	// Return response
	response := models.GetTradesResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
		},
		Trades: tradeDTOs,
		Count:  len(tradeDTOs),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
