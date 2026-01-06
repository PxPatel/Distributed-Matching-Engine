package storage

import (
	"sync"

	"github.com/PxPatel/trading-system/internal/types"
)

// InMemoryTradeStore implements TradeStore using a circular buffer.
// Keeps only the N most recent trades in memory.
type InMemoryTradeStore struct {
	trades  []*types.Trade
	maxSize int
	mutex   sync.RWMutex
}

// NewInMemoryTradeStore creates a new in-memory trade store with a size limit
func NewInMemoryTradeStore(maxSize int) *InMemoryTradeStore {
	return &InMemoryTradeStore{
		trades:  make([]*types.Trade, 0, maxSize),
		maxSize: maxSize,
	}
}

func (s *InMemoryTradeStore) Save(trade *types.Trade) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.trades = append(s.trades, trade)

	// Trim to max size (circular buffer behavior)
	if len(s.trades) > s.maxSize {
		s.trades = s.trades[len(s.trades)-s.maxSize:]
	}

	return nil
}

func (s *InMemoryTradeStore) SaveBatch(trades []*types.Trade) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.trades = append(s.trades, trades...)

	// Trim to max size
	if len(s.trades) > s.maxSize {
		s.trades = s.trades[len(s.trades)-s.maxSize:]
	}

	return nil
}

func (s *InMemoryTradeStore) GetRecent(limit int) ([]*types.Trade, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Clamp limit to actual size
	if limit <= 0 || limit > len(s.trades) {
		limit = len(s.trades)
	}

	// Return last 'limit' trades
	start := len(s.trades) - limit
	result := make([]*types.Trade, limit)
	copy(result, s.trades[start:])

	return result, nil
}

func (s *InMemoryTradeStore) Close() error {
	// No cleanup needed for in-memory store
	return nil
}
