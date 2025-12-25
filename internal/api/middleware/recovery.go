package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/PxPatel/trading-system/internal/api/logger"
	"github.com/PxPatel/trading-system/internal/api/models"
)

// Recovery middleware recovers from panics and returns a 500 error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				logger.Error("Panic recovered", map[string]interface{}{
					"error":      fmt.Sprintf("%v", err),
					"method":     r.Method,
					"path":       r.URL.Path,
					"stacktrace": string(debug.Stack()),
				})

				// Return 500 error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				response := models.BaseResponse{
					Success:   false,
					Timestamp: time.Now().UTC(),
					Message:   "Internal server error",
					Error: &models.APIError{
						Code:    models.ErrInternalError,
						Message: "An unexpected error occurred",
					},
				}

				json.NewEncoder(w).Encode(response)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
