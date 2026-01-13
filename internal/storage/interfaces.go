package storage

import "github.com/PxPatel/trading-system/internal/types"

// OrderStore abstracts order storage and retrieval operations.
// Implementations can be in-memory (map), Redis, PostgreSQL, etc.
type OrderStore interface {
	// Save stores a new order
	Save(order *types.Order) error

	// Get retrieves an order by ID
	Get(orderID uint64) (*types.Order, error)

	// Remove deletes an order from storage
	Remove(orderID uint64) error

	// Update modifies an existing order (for partial fills, status changes)
	Update(order *types.Order) error

	// GetAll returns all tracked orders
	GetAll() []*types.Order

	// GetByUser returns all orders for a specific user
	GetByUser(userID string) []*types.Order

	// GetBySide returns all orders for a specific side (BUY or SELL)
	GetBySide(side types.SideType) []*types.Order

	// Close releases any resources held by the store
	Close() error
}

// TradeStore abstracts trade storage and retrieval operations.
// Implementations can be in-memory buffer, file log, Redis, PostgreSQL, etc.
type TradeStore interface {
	// Save persists a single trade
	Save(trade *types.Trade) error

	// SaveBatch persists multiple trades (useful for database batch inserts)
	SaveBatch(trades []*types.Trade) error

	// GetRecent retrieves the N most recent trades
	GetRecent(limit int) ([]*types.Trade, error)

	// Close releases any resources held by the store
	Close() error
}
