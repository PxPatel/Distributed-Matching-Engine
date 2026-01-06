package storage

import (
	"github.com/PxPatel/trading-system/internal/types"
)

// CompositeTradeStore combines multiple TradeStore implementations.
// Writes go to ALL stores, reads come from the FIRST store that has data.
// Example: CompositeTradeStore([memoryStore, fileStore]) writes to both,
// reads from memory (fast), and persists to file (durable).
type CompositeTradeStore struct {
	stores []TradeStore
}

// NewCompositeTradeStore creates a composite store from multiple stores
func NewCompositeTradeStore(stores ...TradeStore) *CompositeTradeStore {
	return &CompositeTradeStore{
		stores: stores,
	}
}

func (c *CompositeTradeStore) Save(trade *types.Trade) error {
	// Write to all stores
	var lastErr error
	for _, store := range c.stores {
		if err := store.Save(trade); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *CompositeTradeStore) SaveBatch(trades []*types.Trade) error {
	// Write to all stores
	var lastErr error
	for _, store := range c.stores {
		if err := store.SaveBatch(trades); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *CompositeTradeStore) GetRecent(limit int) ([]*types.Trade, error) {
	// Read from first store that returns data
	for _, store := range c.stores {
		trades, err := store.GetRecent(limit)
		if err != nil {
			continue
		}
		if len(trades) > 0 {
			return trades, nil
		}
	}
	return []*types.Trade{}, nil
}

func (c *CompositeTradeStore) Close() error {
	// Close all stores
	var lastErr error
	for _, store := range c.stores {
		if err := store.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
