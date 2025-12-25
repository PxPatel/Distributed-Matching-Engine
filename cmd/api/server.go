package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PxPatel/trading-system/internal/api/handlers"
	"github.com/PxPatel/trading-system/internal/api/logger"
	"github.com/PxPatel/trading-system/internal/api/routes"
	"github.com/PxPatel/trading-system/internal/matching"
)

const (
	defaultPort    = "8080"
	shutdownTimeout = 10 * time.Second
)

func main() {
	// Initialize logger
	logger.SetMinLevel(logger.INFO)

	logger.Info("Starting Distributed Matching Engine API Server", map[string]interface{}{
		"version": "1.0.0",
	})

	// Create matching engine
	engine := matching.NewEngine()
	defer func() {
		if err := engine.Close(); err != nil {
			logger.Error("Failed to close engine", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Create engine holder for dependency injection
	engineHolder := handlers.NewEngineHolder(engine)

	// Setup routes with middleware
	handler := routes.SetupRoutes(engineHolder)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", map[string]interface{}{
			"port":    port,
			"address": fmt.Sprintf("http://localhost:%s", port),
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", map[string]interface{}{
				"error": err.Error(),
			})
			os.Exit(1)
		}
	}()

	logger.Info("Server started successfully", map[string]interface{}{
		"port": port,
	})

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...", nil)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	logger.Info("Server exited successfully", nil)
}
