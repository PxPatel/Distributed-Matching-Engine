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
	"github.com/PxPatel/trading-system/internal/storage"
	"github.com/PxPatel/trading-system/internal/storage/file"
	"github.com/PxPatel/trading-system/internal/storage/memory"
	"github.com/PxPatel/trading-system/internal/storage/postgres"
	"github.com/PxPatel/trading-system/internal/storage/redis"
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

	// Build storage layers based on configuration
	orderStore, tradeStore := buildStorageLayers(cfg)

	// Create matching engine with storage
	engine := matching.NewEngineWithStores(orderStore, tradeStore)
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

// buildStorageLayers constructs the storage layers based on configuration.
// Returns composite stores that layer memory, Redis, and Postgres storage.
func buildStorageLayers(cfg *config.Config) (storage.OrderStore, storage.TradeStore) {
	var orderStores []storage.OrderStore
	var tradeStores []storage.TradeStore

	// L1: In-memory (fastest) - if enabled
	if cfg.Memory.Enabled {
		memOrderStore := memory.NewInMemoryOrderStore(cfg.Memory.MaxOrders)
		memTradeStore := memory.NewInMemoryTradeStore(cfg.Memory.MaxTrades)

		orderStores = append(orderStores, memOrderStore)
		tradeStores = append(tradeStores, memTradeStore)

		logger.Info("In-memory storage layer enabled", map[string]interface{}{
			"max_orders": cfg.Memory.MaxOrders,
			"max_trades": cfg.Memory.MaxTrades,
		})
	}

	// L2: Redis (distributed cache) - if enabled
	if cfg.Redis.Enabled {
		redisCfg := redis.RedisConfig{
			Host:         cfg.Redis.Host,
			Port:         cfg.Redis.Port,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			MaxRetries:   cfg.Redis.MaxRetries,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			OrderTTL:     cfg.Redis.OrderTTL,
			MaxOrders:    cfg.Redis.MaxOrders,
			MaxTrades:    cfg.Redis.MaxTrades,
		}

		redisOrderStore, err := redis.NewRedisOrderStore(redisCfg)
		if err != nil {
			logger.Warn("Failed to connect to Redis, continuing without distributed cache", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			logger.Info("Redis cache connected successfully", map[string]interface{}{
				"host": cfg.Redis.Host,
				"port": cfg.Redis.Port,
			})
			orderStores = append(orderStores, redisOrderStore)

			redisTradeStore, _ := redis.NewRedisTradeStore(redisCfg)
			tradeStores = append(tradeStores, redisTradeStore)
		}
	}

	// L3: PostgreSQL (persistent storage) - if enabled
	if cfg.Database.Enabled {
		pgCfg := postgres.PostgresConfig{
			Host:            cfg.Database.Host,
			Port:            cfg.Database.Port,
			Database:        cfg.Database.Name,
			User:            cfg.Database.User,
			Password:        cfg.Database.Password,
			MaxConns:        cfg.Database.MaxConns,
			MaxIdleConns:    cfg.Database.MaxIdleConns,
			ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
			SSLMode:         cfg.Database.SSLMode,
		}

		pgOrderStore, err := postgres.NewPostgresOrderStore(pgCfg)
		if err != nil {
			logger.Warn("Failed to connect to PostgreSQL, continuing without persistent storage", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			logger.Info("PostgreSQL connected successfully", map[string]interface{}{
				"host":     cfg.Database.Host,
				"database": cfg.Database.Name,
			})
			orderStores = append(orderStores, pgOrderStore)

			pgTradeStore, _ := postgres.NewPostgresTradeStore(pgCfg)
			tradeStores = append(tradeStores, pgTradeStore)
		}
	}

	// L4: File storage (audit log) - always enabled
	if fileTradeStore, err := file.NewFileTradeStore(cfg.Engine.TradeLogPath); err == nil {
		tradeStores = append(tradeStores, fileTradeStore)
		logger.Info("Trade file log enabled", map[string]interface{}{
			"path": cfg.Engine.TradeLogPath,
		})
	}

	// Build composite stores
	var orderStore storage.OrderStore
	var tradeStore storage.TradeStore

	if len(orderStores) == 1 {
		orderStore = orderStores[0]
	} else {
		orderStore = storage.NewCompositeOrderStore(orderStores...)
	}

	if len(tradeStores) == 1 {
		tradeStore = tradeStores[0]
	} else {
		tradeStore = storage.NewCompositeTradeStore(tradeStores...)
	}

	logger.Info("Storage layers initialized", map[string]interface{}{
		"order_layers": len(orderStores),
		"trade_layers": len(tradeStores),
	})

	return orderStore, tradeStore
}
