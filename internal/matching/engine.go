package matching

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type Engine struct {
	orderBook      *OrderBook
	incomingOrders chan *Order
	trades         chan *Trade
	orderTracker   map[uint64]*Order // Track all orders for O(1) lookup
	trackerMutex   sync.RWMutex      // Protect order tracker
	tradeHistory   []*Trade          // Recent trades in memory
	historyMutex   sync.RWMutex      // Protect trade history
	maxHistory     int               // Max trades to keep in memory
	nextOrderID    uint64            // Atomic counter for order IDs
	tradePersister *TradePersister   // Handles trade persistence to disk
}

type Trade struct {
	BuyOrderID  uint64
	SellOrderID uint64
	Price       float64
	Size        int
	Timestamp   time.Time
}

// TradePersister handles writing trades to disk
type TradePersister struct {
	file   *os.File
	mutex  sync.Mutex
	logger *json.Encoder
}

// NewTradePersister creates a new trade persister
func NewTradePersister(filePath string) (*TradePersister, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open trade log: %w", err)
	}

	return &TradePersister{
		file:   file,
		logger: json.NewEncoder(file),
	}, nil
}

// WriteTrade writes a trade to disk
func (tp *TradePersister) WriteTrade(trade *Trade) error {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()
	return tp.logger.Encode(trade)
}

// Close closes the trade persister
func (tp *TradePersister) Close() error {
	return tp.file.Close()
}

// EngineConfig holds configuration for the engine
type EngineConfig struct {
	TradeHistorySize int
	TradeLogPath     string
}

func NewEngine() *Engine {
	return NewEngineWithConfig(&EngineConfig{
		TradeHistorySize: 1000,
		TradeLogPath:     "trades.log",
	})
}

// NewEngineWithConfig creates a new engine with custom configuration
func NewEngineWithConfig(cfg *EngineConfig) *Engine {
	persister, err := NewTradePersister(cfg.TradeLogPath)
	if err != nil {
		// Fallback to no persistence if file can't be opened
		persister = nil
	}

	return &Engine{
		orderBook:      NewOrderBook(),
		incomingOrders: make(chan *Order),
		trades:         make(chan *Trade),
		orderTracker:   make(map[uint64]*Order),
		tradeHistory:   make([]*Trade, 0, cfg.TradeHistorySize),
		maxHistory:     cfg.TradeHistorySize,
		nextOrderID:    1,
		tradePersister: persister,
	}
}

// GenerateOrderID generates a unique order ID
func (e *Engine) GenerateOrderID() uint64 {
	return atomic.AddUint64(&e.nextOrderID, 1)
}

// TrackOrder adds an order to the tracker
func (e *Engine) TrackOrder(order *Order) {
	e.trackerMutex.Lock()
	defer e.trackerMutex.Unlock()
	e.orderTracker[order.ID] = order
}

// UntrackOrder removes an order from the tracker
func (e *Engine) UntrackOrder(orderID uint64) {
	e.trackerMutex.Lock()
	defer e.trackerMutex.Unlock()
	delete(e.orderTracker, orderID)
}

// GetOrder retrieves an order by ID
func (e *Engine) GetOrder(orderID uint64) *Order {
	e.trackerMutex.RLock()
	defer e.trackerMutex.RUnlock()
	return e.orderTracker[orderID]
}

// GetAllOrders returns all tracked orders
func (e *Engine) GetAllOrders() []*Order {
	e.trackerMutex.RLock()
	defer e.trackerMutex.RUnlock()

	orders := make([]*Order, 0, len(e.orderTracker))
	for _, order := range e.orderTracker {
		orders = append(orders, order)
	}
	return orders
}

// GetOrdersByUser returns all orders for a specific user
func (e *Engine) GetOrdersByUser(userID string) []*Order {
	e.trackerMutex.RLock()
	defer e.trackerMutex.RUnlock()

	orders := make([]*Order, 0)
	for _, order := range e.orderTracker {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders
}

// GetOrdersBySide returns all orders for a specific side
func (e *Engine) GetOrdersBySide(side SideType) []*Order {
	e.trackerMutex.RLock()
	defer e.trackerMutex.RUnlock()

	orders := make([]*Order, 0)
	for _, order := range e.orderTracker {
		if order.Side == side {
			orders = append(orders, order)
		}
	}
	return orders
}

// AddTradeToHistory adds a trade to the in-memory history
func (e *Engine) AddTradeToHistory(trade *Trade) {
	e.historyMutex.Lock()
	defer e.historyMutex.Unlock()

	e.tradeHistory = append(e.tradeHistory, trade)

	// Keep only the most recent trades
	if len(e.tradeHistory) > e.maxHistory {
		e.tradeHistory = e.tradeHistory[len(e.tradeHistory)-e.maxHistory:]
	}

	// Persist to disk
	if e.tradePersister != nil {
		go e.tradePersister.WriteTrade(trade)
	}
}

// GetRecentTrades returns recent trades from memory
func (e *Engine) GetRecentTrades(limit int) []*Trade {
	e.historyMutex.RLock()
	defer e.historyMutex.RUnlock()

	if limit <= 0 || limit > len(e.tradeHistory) {
		limit = len(e.tradeHistory)
	}

	start := len(e.tradeHistory) - limit
	trades := make([]*Trade, limit)
	copy(trades, e.tradeHistory[start:])
	slices.Reverse(trades)

	return trades
}

// Close cleanly shuts down the engine
func (e *Engine) Close() error {
	if e.tradePersister != nil {
		return e.tradePersister.Close()
	}
	return nil
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
