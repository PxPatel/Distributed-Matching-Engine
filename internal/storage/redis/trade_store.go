package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/PxPatel/trading-system/internal/types"
)

const (
	tradesKey = "trades:recent"
)

// RedisTradeStore implements TradeStore using Redis sorted sets with FIFO eviction
type RedisTradeStore struct {
	client    *redis.Client
	maxTrades int
}

// NewRedisTradeStore creates a new Redis-backed trade store
func NewRedisTradeStore(cfg RedisConfig) (*RedisTradeStore, error) {
	client, err := NewRedisClient(cfg)
	if err != nil {
		return nil, err
	}

	return &RedisTradeStore{
		client:    client,
		maxTrades: cfg.MaxTrades,
	}, nil
}

func (s *RedisTradeStore) Save(trade *types.Trade) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Serialize trade to JSON
	data, err := json.Marshal(trade)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()

	// Add to sorted set (score = timestamp in unix nanoseconds)
	score := float64(trade.Timestamp.UnixNano())
	pipe.ZAdd(ctx, tradesKey, redis.Z{
		Score:  score,
		Member: data,
	})

	// Trim to keep only last N trades
	pipe.ZRemRangeByRank(ctx, tradesKey, 0, int64(-s.maxTrades-1))

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisTradeStore) SaveBatch(trades []*types.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe := s.client.Pipeline()

	// Add all trades to sorted set
	for _, trade := range trades {
		data, err := json.Marshal(trade)
		if err != nil {
			continue
		}

		score := float64(trade.Timestamp.UnixNano())
		pipe.ZAdd(ctx, tradesKey, redis.Z{
			Score:  score,
			Member: data,
		})
	}

	// Trim to keep only last N trades
	pipe.ZRemRangeByRank(ctx, tradesKey, 0, int64(-s.maxTrades-1))

	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisTradeStore) GetRecent(limit int) ([]*types.Trade, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 100
	}

	// Get last N trades (descending order)
	results, err := s.client.ZRevRange(ctx, tradesKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	trades := make([]*types.Trade, 0, len(results))
	for _, data := range results {
		var trade types.Trade
		if err := json.Unmarshal([]byte(data), &trade); err != nil {
			continue
		}
		trades = append(trades, &trade)
	}

	return trades, nil
}

func (s *RedisTradeStore) Close() error {
	return s.client.Close()
}
