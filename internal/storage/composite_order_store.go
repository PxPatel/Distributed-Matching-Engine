package storage

import (
	"github.com/PxPatel/trading-system/internal/types"
)

// CompositeOrderStore combines multiple OrderStore implementations.
// Writes go to ALL stores, reads come from the FIRST store that succeeds.
// Example: CompositeOrderStore([memoryStore, redisStore, postgresStore])
// writes to all three, reads from memory (fastest), falls back to redis, then postgres.
type CompositeOrderStore struct {
	stores []OrderStore
}

// NewCompositeOrderStore creates a composite store from multiple stores
func NewCompositeOrderStore(stores ...OrderStore) *CompositeOrderStore {
	return &CompositeOrderStore{
		stores: stores,
	}
}

func (c *CompositeOrderStore) Save(order *types.Order) error {
	// Write to all stores (write-through)
	var lastErr error
	for _, store := range c.stores {
		if err := store.Save(order); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *CompositeOrderStore) Get(orderID uint64) (*types.Order, error) {
	// Read from first store that succeeds
	for _, store := range c.stores {
		order, err := store.Get(orderID)
		if err == nil && order != nil {
			return order, nil
		}
	}
	return nil, nil
}

func (c *CompositeOrderStore) Remove(orderID uint64) error {
	// Remove from all stores
	var lastErr error
	for _, store := range c.stores {
		if err := store.Remove(orderID); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *CompositeOrderStore) Update(order *types.Order) error {
	// Update in all stores
	var lastErr error
	for _, store := range c.stores {
		if err := store.Update(order); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *CompositeOrderStore) GetAll() []*types.Order {
	// Read from first store that returns data
	for _, store := range c.stores {
		orders := store.GetAll()
		if len(orders) > 0 {
			return orders
		}
	}
	return []*types.Order{}
}

func (c *CompositeOrderStore) GetByUser(userID string) []*types.Order {
	// Read from first store that returns data
	for _, store := range c.stores {
		orders := store.GetByUser(userID)
		if len(orders) > 0 {
			return orders
		}
	}
	return []*types.Order{}
}

func (c *CompositeOrderStore) GetBySide(side types.SideType) []*types.Order {
	// Read from first store that returns data
	for _, store := range c.stores {
		orders := store.GetBySide(side)
		if len(orders) > 0 {
			return orders
		}
	}
	return []*types.Order{}
}

func (c *CompositeOrderStore) Close() error {
	// Close all stores
	var lastErr error
	for _, store := range c.stores {
		if err := store.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
