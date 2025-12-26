package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/PxPatel/trading-system/config"
	"github.com/PxPatel/trading-system/internal/api/handlers"
	"github.com/PxPatel/trading-system/internal/api/logger"
	"github.com/PxPatel/trading-system/internal/api/routes"
	"github.com/PxPatel/trading-system/internal/matching"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger with config
	logLevel := logger.INFO
	switch cfg.Logger.Level {
	case "DEBUG":
		logLevel = logger.DEBUG
	case "WARN":
		logLevel = logger.WARN
	case "ERROR":
		logLevel = logger.ERROR
	}
	logger.SetMinLevel(logLevel)

	logger.Info("Starting Distributed Matching Engine API Server", map[string]interface{}{
		"version": "1.0.0",
	})

	// Create matching engine with config
	engine := matching.NewEngineWithConfig(&matching.EngineConfig{
		TradeHistorySize: cfg.Engine.TradeHistorySize,
		TradeLogPath:     cfg.Engine.TradeLogPath,
	})
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

	// Create HTTP server with config
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", map[string]interface{}{
			"port":    cfg.Server.Port,
			"address": fmt.Sprintf("http://localhost:%s", cfg.Server.Port),
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", map[string]interface{}{
				"error": err.Error(),
			})
			os.Exit(1)
		}
	}()

	logger.Info("Server started successfully", map[string]interface{}{
		"port": cfg.Server.Port,
	})

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...", nil)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	logger.Info("Server exited successfully", nil)
}
