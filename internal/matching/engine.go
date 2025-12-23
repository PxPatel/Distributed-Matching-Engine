package matching

import (
	"time"
)

type Engine struct {
	orderBook      *OrderBook
	incomingOrders chan *Order
	trades         chan *Trade
}

type Trade struct {
	BuyOrderID  uint64
	SellOrderID uint64
	Price       float64
	Size        int
	Timestamp   time.Time
}

func NewEngine() *Engine {
	return &Engine{
		orderBook:      NewOrderBook(),
		incomingOrders: make(chan *Order),
		trades:         make(chan *Trade),
	}
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
	return e.orderBook.DeleteOrderById(orderId)
}

func (e *Engine) PlaceOrder(incomingOrder *Order) []*Trade {
	switch incomingOrder.OrderType {
	case MarketOrder:
		return e.executeMarketOrder(incomingOrder)
	case LimitOrder:
		return e.executeLimitOrder(incomingOrder)
	case CancelOrder:
		e.CancelOrder(incomingOrder.ID)
		return nil
	default:
		return nil
	}
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
		}
	}

	// Add remaining to book
	if sizeRemaining > 0 {
		incomingOrder.Size = sizeRemaining
		addOrder(incomingOrder)
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
