package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/PxPatel/trading-system/internal/types"
)

const (
	orderKeyPrefix     = "order:"
	userOrdersPrefix   = "user_orders:"
	sideOrdersPrefix   = "side_orders:"
	orderTTL           = 24 * time.Hour // Orders expire after 24 hours
)

// RedisOrderStore implements OrderStore using Redis
type RedisOrderStore struct {
	client *redis.Client
}

// NewRedisOrderStore creates a new Redis-backed order store
func NewRedisOrderStore(cfg RedisConfig) (*RedisOrderStore, error) {
	client, err := NewRedisClient(cfg)
	if err != nil {
		return nil, err
	}

	return &RedisOrderStore{client: client}, nil
}

func (s *RedisOrderStore) Save(order *types.Order) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Serialize order to JSON
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()

	// Store order hash
	orderKey := fmt.Sprintf("%s%d", orderKeyPrefix, order.ID)
	pipe.Set(ctx, orderKey, data, orderTTL)

	// Add to user index
	userKey := fmt.Sprintf("%s%s", userOrdersPrefix, order.UserID)
	pipe.SAdd(ctx, userKey, order.ID)
	pipe.Expire(ctx, userKey, orderTTL)

	// Add to side index
	sideKey := fmt.Sprintf("%s%d", sideOrdersPrefix, order.Side)
	pipe.SAdd(ctx, sideKey, order.ID)
	pipe.Expire(ctx, sideKey, orderTTL)

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisOrderStore) Get(orderID uint64) (*types.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	orderKey := fmt.Sprintf("%s%d", orderKeyPrefix, orderID)
	data, err := s.client.Get(ctx, orderKey).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("order %d not found", orderID)
	}
	if err != nil {
		return nil, err
	}

	var order types.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (s *RedisOrderStore) Remove(orderID uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get order first to clean up indexes
	order, err := s.Get(orderID)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()

	// Remove order
	orderKey := fmt.Sprintf("%s%d", orderKeyPrefix, orderID)
	pipe.Del(ctx, orderKey)

	// Remove from user index
	userKey := fmt.Sprintf("%s%s", userOrdersPrefix, order.UserID)
	pipe.SRem(ctx, userKey, orderID)

	// Remove from side index
	sideKey := fmt.Sprintf("%s%d", sideOrdersPrefix, order.Side)
	pipe.SRem(ctx, sideKey, orderID)

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisOrderStore) Update(order *types.Order) error {
	// For Redis, update is same as save (upsert)
	return s.Save(order)
}

func (s *RedisOrderStore) GetAll() []*types.Order {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Scan for all order keys (note: can be slow with many keys)
	pattern := orderKeyPrefix + "*"
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return []*types.Order{}
	}

	return s.getOrdersByKeys(ctx, keys)
}

func (s *RedisOrderStore) GetByUser(userID string) []*types.Order {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get order IDs from user set
	userKey := fmt.Sprintf("%s%s", userOrdersPrefix, userID)
	orderIDs, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return []*types.Order{}
	}

	// Build keys
	keys := make([]string, len(orderIDs))
	for i, id := range orderIDs {
		keys[i] = orderKeyPrefix + id
	}

	return s.getOrdersByKeys(ctx, keys)
}

func (s *RedisOrderStore) GetBySide(side types.SideType) []*types.Order {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get order IDs from side set
	sideKey := fmt.Sprintf("%s%d", sideOrdersPrefix, side)
	orderIDs, err := s.client.SMembers(ctx, sideKey).Result()
	if err != nil {
		return []*types.Order{}
	}

	// Build keys
	keys := make([]string, len(orderIDs))
	for i, id := range orderIDs {
		keys[i] = orderKeyPrefix + id
	}

	return s.getOrdersByKeys(ctx, keys)
}

func (s *RedisOrderStore) Close() error {
	return s.client.Close()
}

// getOrdersByKeys is a helper to fetch multiple orders by their keys
func (s *RedisOrderStore) getOrdersByKeys(ctx context.Context, keys []string) []*types.Order {
	if len(keys) == 0 {
		return []*types.Order{}
	}

	// Use MGET for efficient batch retrieval
	results, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return []*types.Order{}
	}

	var orders []*types.Order
	for _, result := range results {
		if result == nil {
			continue
		}

		data, ok := result.(string)
		if !ok {
			continue
		}

		var order types.Order
		if err := json.Unmarshal([]byte(data), &order); err != nil {
			continue
		}

		orders = append(orders, &order)
	}

	return orders
}
