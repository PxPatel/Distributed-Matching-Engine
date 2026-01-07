package matching

import (
	"slices"
	"sync/atomic"
	"time"

	"github.com/PxPatel/trading-system/internal/storage"
	"github.com/PxPatel/trading-system/internal/storage/file"
	"github.com/PxPatel/trading-system/internal/storage/memory"
)

type Engine struct {
	orderBook      *OrderBook
	incomingOrders chan *Order
	trades         chan *Trade
	nextOrderID    uint64 // Atomic counter for order IDs
	orderStore     storage.OrderStore
	tradeStore     storage.TradeStore
}

// EngineConfig holds configuration for the engine
type EngineConfig struct {
	MaxOrders        int
	MaxTrades        int
	TradeHistorySize int    // Deprecated: use MaxTrades
	TradeLogPath     string
}

// NewEngine creates a new engine with default configuration and in-memory+file storage
func NewEngine() *Engine {
	return NewEngineWithConfig(&EngineConfig{
		MaxOrders:        100000,
		MaxTrades:        1000,
		TradeHistorySize: 1000, // Deprecated
		TradeLogPath:     "trades.log",
	})
}

// NewEngineWithConfig creates a new engine with custom configuration.
// Uses composite storage: in-memory (fast reads) + file (durable writes).
func NewEngineWithConfig(cfg *EngineConfig) *Engine {
	// Default storage: memory + file composite
	orderStore := memory.NewInMemoryOrderStore(cfg.MaxOrders)

	// Create trade store with fallback if file can't be opened
	maxTrades := cfg.MaxTrades
	if maxTrades == 0 && cfg.TradeHistorySize > 0 {
		maxTrades = cfg.TradeHistorySize // Backward compatibility
	}
	memoryTradeStore := memory.NewInMemoryTradeStore(maxTrades)
	fileTradeStore, err := file.NewFileTradeStore(cfg.TradeLogPath)

	var tradeStore storage.TradeStore
	if err != nil {
		// Fallback to memory-only if file can't be opened
		tradeStore = memoryTradeStore
	} else {
		// Composite: writes to both memory and file
		tradeStore = storage.NewCompositeTradeStore(memoryTradeStore, fileTradeStore)
	}

	return NewEngineWithStores(orderStore, tradeStore)
}

// NewEngineWithStores creates a new engine with custom storage implementations.
// This constructor allows full control over storage backends (e.g., Redis, PostgreSQL).
func NewEngineWithStores(orderStore storage.OrderStore, tradeStore storage.TradeStore) *Engine {
	return &Engine{
		orderBook:      NewOrderBook(),
		incomingOrders: make(chan *Order),
		trades:         make(chan *Trade),
		nextOrderID:    1,
		orderStore:     orderStore,
		tradeStore:     tradeStore,
	}
}

// GenerateOrderID generates a unique order ID
func (e *Engine) GenerateOrderID() uint64 {
	return atomic.AddUint64(&e.nextOrderID, 1)
}

// TrackOrder adds an order to the store
func (e *Engine) TrackOrder(order *Order) {
	_ = e.orderStore.Save(order)
}

// UntrackOrder removes an order from the store
func (e *Engine) UntrackOrder(orderID uint64) {
	_ = e.orderStore.Remove(orderID)
}

// GetOrder retrieves an order by ID
func (e *Engine) GetOrder(orderID uint64) *Order {
	order, _ := e.orderStore.Get(orderID)
	return order
}

// GetAllOrders returns all tracked orders
func (e *Engine) GetAllOrders() []*Order {
	return e.orderStore.GetAll()
}

// GetOrdersByUser returns all orders for a specific user
func (e *Engine) GetOrdersByUser(userID string) []*Order {
	return e.orderStore.GetByUser(userID)
}

// GetOrdersBySide returns all orders for a specific side
func (e *Engine) GetOrdersBySide(side SideType) []*Order {
	return e.orderStore.GetBySide(side)
}

// AddTradeToHistory saves a trade to the store
func (e *Engine) AddTradeToHistory(trade *Trade) {
	_ = e.tradeStore.Save(trade)
}

// GetRecentTrades returns recent trades from the store
func (e *Engine) GetRecentTrades(limit int) []*Trade {
	trades, _ := e.tradeStore.GetRecent(limit)
	// Reverse for newest-first ordering (API convention)
	slices.Reverse(trades)
	return trades
}

// Close cleanly shuts down the engine and releases resources
func (e *Engine) Close() error {
	if err := e.orderStore.Close(); err != nil {
		return err
	}
	return e.tradeStore.Close()
}

// GetOrderBook returns the order book
func (e *Engine) GetOrderBook() *OrderBook {
	return e.orderBook
}

func (e *Engine) Start() {
	for order := range e.incomingOrders {
		trades := e.PlaceOrder(order)
		for _, trade := range trades {
			e.trades <- trade
		}
	}
}

func (e *Engine) CancelOrder(orderId uint64) bool {
	deleted := e.orderBook.DeleteOrderById(orderId)
	if deleted {
		e.UntrackOrder(orderId)
	}
	return deleted
}

func (e *Engine) PlaceOrder(incomingOrder *Order) []*Trade {
	// Track the order
	if incomingOrder.OrderType != CancelOrder {
		e.TrackOrder(incomingOrder)
	}

	var trades []*Trade

	switch incomingOrder.OrderType {
	case MarketOrder:
		trades = e.executeMarketOrder(incomingOrder)
	case LimitOrder:
		trades = e.executeLimitOrder(incomingOrder)
	case CancelOrder:
		e.CancelOrder(incomingOrder.ID)
		return nil
	default:
		return nil
	}

	// Add trades to history
	for _, trade := range trades {
		e.AddTradeToHistory(trade)
	}

	// If market order is fully filled, untrack it (it won't be in the book)
	if incomingOrder.OrderType == MarketOrder {
		e.UntrackOrder(incomingOrder.ID)
	}

	return trades
}

func (e *Engine) executeMarketOrder(incomingOrder *Order) []*Trade {
	var trades []*Trade
	sizeRemaining := incomingOrder.Size

	var getBestPrice func() (float64, []*Order)
	var deleteOrder func(uint64) bool

	if incomingOrder.Side == Buy {
		getBestPrice = e.orderBook.GetBestAsk
		deleteOrder = e.orderBook.DeleteAskOrder
	} else {
		getBestPrice = e.orderBook.GetBestBid
		deleteOrder = e.orderBook.DeleteBidOrder
	}

	for sizeRemaining > 0 {
		_, orderBlock := getBestPrice()

		// Check if liquidity available
		if len(orderBlock) == 0 {
			// No liquidity and market order fails (or handle otherwise)
			break
		}
		oppositeOrder := orderBlock[0]

		// Determine fill size
		fillSize := sizeRemaining
		if oppositeOrder.Size < sizeRemaining {
			fillSize = oppositeOrder.Size
		}

		// Create trade
		trade := e.createTrade(incomingOrder, oppositeOrder, fillSize)
		trades = append(trades, trade)

		// Update sizes
		sizeRemaining -= fillSize
		oppositeOrder.Size -= fillSize

		// Remove if fully filled
		if oppositeOrder.Size == 0 {
			deleteOrder(oppositeOrder.ID)
			e.UntrackOrder(oppositeOrder.ID)
		}
	}

	return trades
}

func (e *Engine) executeLimitOrder(incomingOrder *Order) []*Trade {
	var trades []*Trade

	sizeRemaining := incomingOrder.Size

	var getBestPrice func() (float64, []*Order)
	var addOrder func(*Order) bool
	var deleteOrder func(uint64) bool
	var canMatch func(float64, float64) bool

	if incomingOrder.Side == Buy {
		getBestPrice = e.orderBook.GetBestAsk
		addOrder = e.orderBook.AddBidOrder
		deleteOrder = e.orderBook.DeleteAskOrder
		canMatch = func(limitPrice, bestPrice float64) bool {
			return limitPrice >= bestPrice // Buy at or above ask
		}
	} else {
		getBestPrice = e.orderBook.GetBestBid
		addOrder = e.orderBook.AddAskOrder
		deleteOrder = e.orderBook.DeleteBidOrder
		canMatch = func(limitPrice, bestPrice float64) bool {
			return limitPrice <= bestPrice // Sell at or below bid
		}
	}

	// Try to match
	for sizeRemaining > 0 {
		bestPrice, orderBlock := getBestPrice()

		// Check if there is current liquidity or we can match
		if len(orderBlock) == 0 || !canMatch(incomingOrder.Price, bestPrice) {
			break
		}

		oppositeOrder := orderBlock[0]

		// Determine fill size
		fillSize := sizeRemaining
		if oppositeOrder.Size < sizeRemaining {
			fillSize = oppositeOrder.Size
		}

		// Create trade
		trade := e.createTrade(incomingOrder, oppositeOrder, fillSize)
		trades = append(trades, trade)

		// Update sizes
		sizeRemaining -= fillSize
		oppositeOrder.Size -= fillSize

		// Remove if fully filled
		if oppositeOrder.Size == 0 {
			deleteOrder(oppositeOrder.ID)
			e.UntrackOrder(oppositeOrder.ID)
		}
	}

	// Add remaining to book
	if sizeRemaining > 0 {
		incomingOrder.Size = sizeRemaining
		addOrder(incomingOrder)
	} else {
		// Fully filled, untrack the incoming order
		e.UntrackOrder(incomingOrder.ID)
	}

	return trades
}

func (e *Engine) createTrade(incoming *Order, opposite *Order, size int) *Trade {
	trade := &Trade{
		Price:     opposite.Price, // Always execute at resting order price
		Size:      size,
		Timestamp: time.Now(),
	}

	if incoming.Side == Buy {
		trade.BuyOrderID = incoming.ID
		trade.SellOrderID = opposite.ID
	} else {
		trade.BuyOrderID = opposite.ID
		trade.SellOrderID = incoming.ID
	}

	return trade
}
