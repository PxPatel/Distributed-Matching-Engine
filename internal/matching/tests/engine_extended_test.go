package matching

import (
	"testing"

	"github.com/PxPatel/trading-system/internal/matching"
)

// TestGenerateOrderID tests unique order ID generation
func TestGenerateOrderID(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Generate multiple IDs
	ids := make(map[uint64]bool)
	for i := 0; i < 1000; i++ {
		id := engine.GenerateOrderID()
		if ids[id] {
			t.Errorf("Duplicate order ID generated: %d", id)
		}
		ids[id] = true
	}

	// Should have 1000 unique IDs
	if len(ids) != 1000 {
		t.Errorf("Expected 1000 unique IDs, got %d", len(ids))
	}
}

// TestTrackOrder tests order tracking functionality
func TestTrackOrder(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	order := matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10)

	// Track the order
	engine.TrackOrder(order)

	// Retrieve the order
	retrieved := engine.GetOrder(1)
	if retrieved == nil {
		t.Fatal("Order should be tracked")
	}

	if retrieved.ID != 1 || retrieved.UserID != "user1" {
		t.Errorf("Retrieved order doesn't match: got ID=%d, UserID=%s", retrieved.ID, retrieved.UserID)
	}
}

// TestUntrackOrder tests order untracking
func TestUntrackOrder(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	order := matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10)
	engine.TrackOrder(order)

	// Verify tracked
	if engine.GetOrder(1) == nil {
		t.Fatal("Order should be tracked")
	}

	// Untrack
	engine.UntrackOrder(1)

	// Verify untracked
	if engine.GetOrder(1) != nil {
		t.Error("Order should be untracked")
	}
}

// TestGetAllOrders tests retrieving all tracked orders
func TestGetAllOrders(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Track multiple orders
	for i := uint64(1); i <= 5; i++ {
		order := matching.NewOrder(i, "user1", matching.LimitOrder, matching.Buy, 100.0, 10)
		engine.TrackOrder(order)
	}

	// Get all orders
	orders := engine.GetAllOrders()
	if len(orders) != 5 {
		t.Errorf("Expected 5 orders, got %d", len(orders))
	}
}

// TestGetOrdersByUser tests filtering orders by user
func TestGetOrdersByUser(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Track orders for different users
	engine.TrackOrder(matching.NewOrder(1, "alice", matching.LimitOrder, matching.Buy, 100.0, 10))
	engine.TrackOrder(matching.NewOrder(2, "bob", matching.LimitOrder, matching.Buy, 100.0, 10))
	engine.TrackOrder(matching.NewOrder(3, "alice", matching.LimitOrder, matching.Sell, 101.0, 10))
	engine.TrackOrder(matching.NewOrder(4, "charlie", matching.LimitOrder, matching.Buy, 99.0, 10))

	// Get Alice's orders
	aliceOrders := engine.GetOrdersByUser("alice")
	if len(aliceOrders) != 2 {
		t.Errorf("Expected 2 orders for Alice, got %d", len(aliceOrders))
	}

	// Verify they're Alice's
	for _, order := range aliceOrders {
		if order.UserID != "alice" {
			t.Errorf("Expected Alice's order, got %s", order.UserID)
		}
	}
}

// TestGetOrdersBySide tests filtering orders by side
func TestGetOrdersBySide(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Track mixed orders
	engine.TrackOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10))
	engine.TrackOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Sell, 101.0, 10))
	engine.TrackOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Buy, 99.0, 10))

	// Get buy orders
	buyOrders := engine.GetOrdersBySide(matching.Buy)
	if len(buyOrders) != 2 {
		t.Errorf("Expected 2 buy orders, got %d", len(buyOrders))
	}

	// Get sell orders
	sellOrders := engine.GetOrdersBySide(matching.Sell)
	if len(sellOrders) != 1 {
		t.Errorf("Expected 1 sell order, got %d", len(sellOrders))
	}
}

// TestAddTradeToHistory tests trade history management
func TestAddTradeToHistory(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Create and add trades
	for i := 0; i < 10; i++ {
		trade := &matching.Trade{
			BuyOrderID:  uint64(i),
			SellOrderID: uint64(i + 100),
			Price:       100.0,
			Size:        10,
		}
		engine.AddTradeToHistory(trade)
	}

	// Retrieve recent trades
	trades := engine.GetRecentTrades(5)
	if len(trades) != 5 {
		t.Errorf("Expected 5 recent trades, got %d", len(trades))
	}

	// Verify order (most recent first)
	if trades[0].BuyOrderID != 9 {
		t.Errorf("Expected most recent trade first, got BuyOrderID=%d", trades[0].BuyOrderID)
	}
}

// TestTradeHistoryLimit tests that history respects max size
func TestTradeHistoryLimit(t *testing.T) {
	// Create engine with small history
	engine := matching.NewEngineWithConfig(&matching.EngineConfig{
		TradeHistorySize: 5,
		TradeLogPath:     "test_trades.log",
	})
	defer engine.Close()

	// Add more trades than limit
	for i := 0; i < 10; i++ {
		trade := &matching.Trade{
			BuyOrderID:  uint64(i),
			SellOrderID: uint64(i + 100),
			Price:       100.0,
			Size:        10,
		}
		engine.AddTradeToHistory(trade)
	}

	// Should only keep last 5
	trades := engine.GetRecentTrades(100)
	if len(trades) != 5 {
		t.Errorf("Expected 5 trades (max history), got %d", len(trades))
	}

	// Should be trades 5-9
	if trades[0].BuyOrderID != 5 {
		t.Errorf("Expected oldest trade to be BuyOrderID=5, got %d", trades[0].BuyOrderID)
	}
}

// TestPlaceOrderTracking tests that PlaceOrder tracks orders correctly
func TestPlaceOrderTracking(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Place limit order (should be tracked)
	limitOrder := matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 99.0, 10)
	engine.PlaceOrder(limitOrder)

	// Verify tracked
	if engine.GetOrder(1) == nil {
		t.Error("Limit order should be tracked")
	}

	// Place another limit order for matching
	sellOrder := matching.NewOrder(2, "user2", matching.LimitOrder, matching.Sell, 99.0, 10)
	engine.PlaceOrder(sellOrder)

	// Both should be untracked after full fill
	if engine.GetOrder(1) != nil {
		t.Error("Fully filled buy order should be untracked")
	}
	if engine.GetOrder(2) != nil {
		t.Error("Fully filled sell order should be untracked")
	}
}

// TestMarketOrderNotTracked tests that fully filled market orders are untracked
func TestMarketOrderNotTracked(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Add liquidity
	engine.PlaceOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Sell, 100.0, 10))

	// Place market order (should be fully filled and untracked)
	marketOrder := matching.NewOrder(2, "user2", matching.MarketOrder, matching.Buy, 0, 10)
	trades := engine.PlaceOrder(marketOrder)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}

	// Market order should be untracked (fully filled)
	if engine.GetOrder(2) != nil {
		t.Error("Fully filled market order should be untracked")
	}
}

// TestPartialFillTracking tests that partially filled orders remain tracked
func TestPartialFillTracking(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Add small liquidity
	engine.PlaceOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Sell, 100.0, 5))

	// Place larger limit order
	bigOrder := matching.NewOrder(2, "user2", matching.LimitOrder, matching.Buy, 100.0, 20)
	trades := engine.PlaceOrder(bigOrder)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}

	// Order should still be tracked (partial fill)
	tracked := engine.GetOrder(2)
	if tracked == nil {
		t.Fatal("Partially filled order should still be tracked")
	}

	// Remaining size should be in the book
	if tracked.Size != 15 {
		t.Errorf("Expected remaining size 15, got %d", tracked.Size)
	}
}

// TestGetOrderBookAccess tests accessing the orderbook
func TestGetOrderBookAccess(t *testing.T) {
	engine := matching.NewEngine()
	defer engine.Close()

	// Place some orders
	engine.PlaceOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 99.0, 10))
	engine.PlaceOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Sell, 101.0, 10))

	// Access orderbook
	ob := engine.GetOrderBook()
	if ob == nil {
		t.Fatal("GetOrderBook should return non-nil")
	}

	// Verify content
	bidPrice, bidOrders := ob.GetBestBid()
	if bidPrice != 99.0 || len(bidOrders) != 1 {
		t.Errorf("Expected best bid at 99.0 with 1 order, got price=%f, count=%d", bidPrice, len(bidOrders))
	}

	askPrice, askOrders := ob.GetBestAsk()
	if askPrice != 101.0 || len(askOrders) != 1 {
		t.Errorf("Expected best ask at 101.0 with 1 order, got price=%f, count=%d", askPrice, len(askOrders))
	}
}
