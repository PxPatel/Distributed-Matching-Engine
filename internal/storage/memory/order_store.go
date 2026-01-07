package memory

import (
	"fmt"
	"sync"

	"github.com/PxPatel/trading-system/internal/types"
)

// InMemoryOrderStore implements OrderStore using an in-memory map with FIFO eviction.
// Thread-safe for concurrent access via RWMutex.
// When maxSize is reached, oldest orders are evicted to maintain size limit.
type InMemoryOrderStore struct {
	orders    map[uint64]*types.Order
	orderIDs  []uint64 // FIFO queue for eviction
	maxSize   int
	mutex     sync.RWMutex
}

// NewInMemoryOrderStore creates a new in-memory order store with a size limit
func NewInMemoryOrderStore(maxSize int) *InMemoryOrderStore {
	return &InMemoryOrderStore{
		orders:   make(map[uint64]*types.Order),
		orderIDs: make([]uint64, 0, maxSize),
		maxSize:  maxSize,
	}
}

func (s *InMemoryOrderStore) Save(order *types.Order) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if order already exists (update case)
	if _, exists := s.orders[order.ID]; !exists {
		// New order: add to FIFO queue
		s.orderIDs = append(s.orderIDs, order.ID)

		// Evict oldest order if size limit exceeded
		if len(s.orderIDs) > s.maxSize {
			oldestID := s.orderIDs[0]
			delete(s.orders, oldestID)
			s.orderIDs = s.orderIDs[1:]
		}
	}

	s.orders[order.ID] = order
	return nil
}

func (s *InMemoryOrderStore) Get(orderID uint64) (*types.Order, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	order, exists := s.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order %d not found", orderID)
	}
	return order, nil
}

func (s *InMemoryOrderStore) Remove(orderID uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.orders, orderID)

	// Remove from FIFO queue
	for i, id := range s.orderIDs {
		if id == orderID {
			s.orderIDs = append(s.orderIDs[:i], s.orderIDs[i+1:]...)
			break
		}
	}

	return nil
}

func (s *InMemoryOrderStore) Update(order *types.Order) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.orders[order.ID]; !exists {
		return fmt.Errorf("order %d not found", order.ID)
	}
	s.orders[order.ID] = order
	return nil
}

func (s *InMemoryOrderStore) GetAll() []*types.Order {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	orders := make([]*types.Order, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, order)
	}
	return orders
}

func (s *InMemoryOrderStore) GetByUser(userID string) []*types.Order {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var orders []*types.Order
	for _, order := range s.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders
}

func (s *InMemoryOrderStore) GetBySide(side types.SideType) []*types.Order {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var orders []*types.Order
	for _, order := range s.orders {
		if order.Side == side {
			orders = append(orders, order)
		}
	}
	return orders
}

func (s *InMemoryOrderStore) Close() error {
	// No cleanup needed for in-memory store
	return nil
}
