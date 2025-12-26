package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig
	Engine EngineConfig
	API    APIConfig
	Logger LoggerConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// EngineConfig holds matching engine configuration
type EngineConfig struct {
	TradeHistorySize int
	TradeLogPath     string
	OrderCleanupEnabled bool
	OrderCleanupInterval time.Duration
}

// APIConfig holds API-specific configuration
type APIConfig struct {
	DefaultOrderLimit     int
	MaxOrderLimit         int
	DefaultTradeLimit     int
	MaxTradeLimit         int
	DefaultOrderBookDepth int
	MaxOrderBookDepth     int
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level string // DEBUG, INFO, WARN, ERROR
}

var instance *Config

// Load loads configuration from .env file (if exists) and environment variables
func Load() (*Config, error) {
	// Try to load .env file (optional)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Engine: EngineConfig{
			TradeHistorySize: getEnvInt("TRADE_HISTORY_SIZE", 1000),
			TradeLogPath:     getEnv("TRADE_LOG_PATH", "trades.log"),
			OrderCleanupEnabled: getEnvBool("ORDER_CLEANUP_ENABLED", false),
			OrderCleanupInterval: getEnvDuration("ORDER_CLEANUP_INTERVAL", 5*time.Minute),
		},
		API: APIConfig{
			DefaultOrderLimit:     getEnvInt("DEFAULT_ORDER_LIMIT", 100),
			MaxOrderLimit:         getEnvInt("MAX_ORDER_LIMIT", 1000),
			DefaultTradeLimit:     getEnvInt("DEFAULT_TRADE_LIMIT", 100),
			MaxTradeLimit:         getEnvInt("MAX_TRADE_LIMIT", 1000),
			DefaultOrderBookDepth: getEnvInt("DEFAULT_ORDERBOOK_DEPTH", 10),
			MaxOrderBookDepth:     getEnvInt("MAX_ORDERBOOK_DEPTH", 10),
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "INFO"),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	instance = cfg
	return cfg, nil
}

// Get returns the singleton config instance
func Get() *Config {
	if instance == nil {
		panic("config not loaded - call config.Load() first")
	}
	return instance
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port == "" {
		return fmt.Errorf("PORT cannot be empty")
	}

	// Validate engine config
	if c.Engine.TradeHistorySize < 0 {
		return fmt.Errorf("TRADE_HISTORY_SIZE must be >= 0")
	}
	if c.Engine.TradeLogPath == "" {
		return fmt.Errorf("TRADE_LOG_PATH cannot be empty")
	}

	// Validate API config
	if c.API.DefaultOrderLimit < 1 {
		return fmt.Errorf("DEFAULT_ORDER_LIMIT must be > 0")
	}
	if c.API.MaxOrderLimit < c.API.DefaultOrderLimit {
		return fmt.Errorf("MAX_ORDER_LIMIT must be >= DEFAULT_ORDER_LIMIT")
	}
	if c.API.DefaultTradeLimit < 1 {
		return fmt.Errorf("DEFAULT_TRADE_LIMIT must be > 0")
	}
	if c.API.MaxTradeLimit < c.API.DefaultTradeLimit {
		return fmt.Errorf("MAX_TRADE_LIMIT must be >= DEFAULT_TRADE_LIMIT")
	}
	if c.API.DefaultOrderBookDepth < 1 {
		return fmt.Errorf("DEFAULT_ORDERBOOK_DEPTH must be > 0")
	}
	if c.API.MaxOrderBookDepth < c.API.DefaultOrderBookDepth {
		return fmt.Errorf("MAX_ORDERBOOK_DEPTH must be >= DEFAULT_ORDERBOOK_DEPTH")
	}

	// Validate logger config
	validLevels := map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true}
	if !validLevels[c.Logger.Level] {
		return fmt.Errorf("LOG_LEVEL must be one of: DEBUG, INFO, WARN, ERROR")
	}

	return nil
}

// Helper functions to read environment variables with defaults

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
