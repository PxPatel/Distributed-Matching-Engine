package matching

import (
	"testing"

	"github.com/PxPatel/trading-system/internal/matching"
)

// TestGetAllBids tests retrieving all bid price levels
func TestGetAllBids(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add bids at different prices
	ob.AddBidOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Buy, 99.0, 5))
	ob.AddBidOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Buy, 101.0, 8))

	prices := ob.GetAllBids()

	// Should have 3 price levels
	if len(prices) != 3 {
		t.Errorf("Expected 3 bid levels, got %d", len(prices))
	}

	// Should be sorted descending (highest first)
	if prices[0] != 101.0 || prices[1] != 100.0 || prices[2] != 99.0 {
		t.Errorf("Bids not sorted correctly: %v", prices)
	}
}

// TestGetAllAsks tests retrieving all ask price levels
func TestGetAllAsks(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add asks at different prices
	ob.AddAskOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Sell, 102.0, 10))
	ob.AddAskOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Sell, 101.0, 5))
	ob.AddAskOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Sell, 103.0, 8))

	prices := ob.GetAllAsks()

	// Should have 3 price levels
	if len(prices) != 3 {
		t.Errorf("Expected 3 ask levels, got %d", len(prices))
	}

	// Should be sorted ascending (lowest first)
	if prices[0] != 101.0 || prices[1] != 102.0 || prices[2] != 103.0 {
		t.Errorf("Asks not sorted correctly: %v", prices)
	}
}

// TestGetBidsAtPrice tests retrieving bids at specific price
func TestGetBidsAtPrice(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add multiple bids at same price
	ob.AddBidOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Buy, 100.0, 5))
	ob.AddBidOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Buy, 99.0, 8))

	// Get bids at 100.0
	orders := ob.GetBidsAtPrice(100.0)
	if len(orders) != 2 {
		t.Errorf("Expected 2 orders at 100.0, got %d", len(orders))
	}

	// Verify FIFO order
	if orders[0].ID != 1 || orders[1].ID != 2 {
		t.Error("Orders not in FIFO sequence")
	}
}

// TestGetAsksAtPrice tests retrieving asks at specific price
func TestGetAsksAtPrice(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add multiple asks at same price
	ob.AddAskOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Sell, 101.0, 10))
	ob.AddAskOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Sell, 101.0, 5))
	ob.AddAskOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Sell, 102.0, 8))

	// Get asks at 101.0
	orders := ob.GetAsksAtPrice(101.0)
	if len(orders) != 2 {
		t.Errorf("Expected 2 orders at 101.0, got %d", len(orders))
	}

	// Verify FIFO order
	if orders[0].ID != 1 || orders[1].ID != 2 {
		t.Error("Orders not in FIFO sequence")
	}
}

// TestEmptyOrderBookPriceLevels tests price level functions on empty book
func TestEmptyOrderBookPriceLevels(t *testing.T) {
	ob := matching.NewOrderBook()

	bidPrices := ob.GetAllBids()
	askPrices := ob.GetAllAsks()

	if len(bidPrices) != 0 {
		t.Errorf("Expected 0 bid levels, got %d", len(bidPrices))
	}

	if len(askPrices) != 0 {
		t.Errorf("Expected 0 ask levels, got %d", len(askPrices))
	}
}

// TestGetPriceAtNonExistentLevel tests querying non-existent price
func TestGetPriceAtNonExistentLevel(t *testing.T) {
	ob := matching.NewOrderBook()

	ob.AddBidOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10))

	// Query non-existent price
	orders := ob.GetBidsAtPrice(99.0)
	if orders != nil {
		t.Error("Expected nil for non-existent price level")
	}
}

// TestLargePriceLevelCount tests performance with many price levels
func TestLargePriceLevelCount(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add 1000 different price levels
	for i := 0; i < 1000; i++ {
		price := 100.0 + float64(i)*0.01
		ob.AddBidOrder(matching.NewOrder(uint64(i), "user", matching.LimitOrder, matching.Buy, price, 10))
	}

	prices := ob.GetAllBids()

	if len(prices) != 1000 {
		t.Errorf("Expected 1000 price levels, got %d", len(prices))
	}

	// Verify sorted (descending)
	for i := 0; i < len(prices)-1; i++ {
		if prices[i] < prices[i+1] {
			t.Error("Bid prices not sorted in descending order")
			break
		}
	}
}

// TestMixedPriceLevels tests with both bids and asks
func TestMixedPriceLevels(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add mixed orders
	ob.AddBidOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 99.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Buy, 98.0, 5))
	ob.AddAskOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Sell, 101.0, 8))
	ob.AddAskOrder(matching.NewOrder(4, "user4", matching.LimitOrder, matching.Sell, 102.0, 12))

	bidPrices := ob.GetAllBids()
	askPrices := ob.GetAllAsks()

	if len(bidPrices) != 2 {
		t.Errorf("Expected 2 bid levels, got %d", len(bidPrices))
	}

	if len(askPrices) != 2 {
		t.Errorf("Expected 2 ask levels, got %d", len(askPrices))
	}

	// Verify best prices are at expected positions
	if bidPrices[0] != 99.0 {
		t.Errorf("Expected best bid 99.0, got %f", bidPrices[0])
	}

	if askPrices[0] != 101.0 {
		t.Errorf("Expected best ask 101.0, got %f", askPrices[0])
	}
}

// TestPriceLevelAfterDeletion tests price levels after orders are deleted
func TestPriceLevelAfterDeletion(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add orders
	ob.AddBidOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Buy, 99.0, 5))

	// Delete one price level completely
	ob.DeleteBidOrder(1)

	prices := ob.GetAllBids()
	if len(prices) != 1 {
		t.Errorf("Expected 1 price level after deletion, got %d", len(prices))
	}

	if prices[0] != 99.0 {
		t.Errorf("Expected remaining price 99.0, got %f", prices[0])
	}
}

// TestMultipleOrdersAtSamePriceLevel tests FIFO within price level
func TestMultipleOrdersAtSamePriceLevel(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add 5 orders at same price
	for i := uint64(1); i <= 5; i++ {
		ob.AddBidOrder(matching.NewOrder(i, "user", matching.LimitOrder, matching.Buy, 100.0, 10))
	}

	orders := ob.GetBidsAtPrice(100.0)
	if len(orders) != 5 {
		t.Errorf("Expected 5 orders at price level, got %d", len(orders))
	}

	// Verify FIFO order (IDs should be 1, 2, 3, 4, 5)
	for i, order := range orders {
		expectedID := uint64(i + 1)
		if order.ID != expectedID {
			t.Errorf("Expected ID %d at position %d, got %d", expectedID, i, order.ID)
		}
	}
}

// TestPriceLevelQuantityAggregation tests total quantity at price level
func TestPriceLevelQuantityAggregation(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add multiple orders at same price with different sizes
	ob.AddBidOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Buy, 100.0, 20))
	ob.AddBidOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Buy, 100.0, 15))

	orders := ob.GetBidsAtPrice(100.0)

	// Calculate total quantity
	totalQty := 0
	for _, order := range orders {
		totalQty += order.Size
	}

	if totalQty != 45 {
		t.Errorf("Expected total quantity 45, got %d", totalQty)
	}
}

// TestPriceLevelStability tests that price levels remain stable during operations
func TestPriceLevelStability(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add initial orders
	ob.AddBidOrder(matching.NewOrder(1, "user1", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user2", matching.LimitOrder, matching.Buy, 99.0, 10))
	ob.AddBidOrder(matching.NewOrder(3, "user3", matching.LimitOrder, matching.Buy, 98.0, 10))

	// Get initial state
	initialPrices := ob.GetAllBids()

	// Query multiple times
	for i := 0; i < 100; i++ {
		prices := ob.GetAllBids()
		if len(prices) != len(initialPrices) {
			t.Error("Price level count changed unexpectedly")
		}

		for j, price := range prices {
			if price != initialPrices[j] {
				t.Error("Price level order changed unexpectedly")
			}
		}
	}
}
