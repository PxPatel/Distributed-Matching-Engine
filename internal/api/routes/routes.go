package routes

import (
	"net/http"

	"github.com/PxPatel/trading-system/internal/api/handlers"
	"github.com/PxPatel/trading-system/internal/api/middleware"
)

// SetupRoutes configures all API routes with middleware
func SetupRoutes(engineHolder *handlers.EngineHolder) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/api/v1/health", handlers.HealthHandler)

	// Order endpoints
	mux.HandleFunc("/api/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			engineHolder.SubmitOrderHandler(w, r)
		case http.MethodGet:
			engineHolder.GetAllOrdersHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/orders/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			engineHolder.BatchOrderHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/orders/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			engineHolder.GetOrderHandler(w, r)
		case http.MethodDelete:
			engineHolder.CancelOrderHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Order book endpoints
	mux.HandleFunc("/api/v1/orderbook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			engineHolder.GetOrderBookHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/orderbook/top", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			engineHolder.GetTopOfBookHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Trade endpoints
	mux.HandleFunc("/api/v1/trades", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			engineHolder.GetTradesHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Apply middleware (order matters: Recovery -> CORS -> Logging -> Handler)
	handler := middleware.Recovery(mux)
	handler = middleware.CORS(handler)
	handler = middleware.Logging(handler)

	return handler
}
