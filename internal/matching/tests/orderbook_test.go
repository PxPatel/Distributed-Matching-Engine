package matching

import (
	"math"
	"testing"

	"github.com/PxPatel/trading-system/internal/matching"
)

// TestNewOrderBook tests the OrderBook constructor
func TestNewOrderBook(t *testing.T) {
	ob := matching.NewOrderBook()

	if ob == nil {
		t.Fatal("NewOrderBook() returned nil")
	}

	// Verify empty book
	bidPrice, bidOrders := ob.GetBestBid()
	if bidPrice != 0.0 || bidOrders != nil {
		t.Errorf("Expected empty bids, got price=%f, orders=%v", bidPrice, bidOrders)
	}

	askPrice, askOrders := ob.GetBestAsk()
	if askPrice != 0.0 || askOrders != nil {
		t.Errorf("Expected empty asks, got price=%f, orders=%v", askPrice, askOrders)
	}
}

// TestAddBidOrder tests adding bid orders
func TestAddBidOrder(t *testing.T) {
	ob := matching.NewOrderBook()

	tests := []struct {
		name  string
		order *matching.Order
	}{
		{"SingleBid", matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10)},
		{"HigherBid", matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Buy, 101.0, 20)},
		{"LowerBid", matching.NewOrder(3, "user_test", matching.LimitOrder, matching.Buy, 99.0, 15)},
		{"SamePriceBid", matching.NewOrder(4, "user_test", matching.LimitOrder, matching.Buy, 100.0, 5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success := ob.AddBidOrder(tt.order)
			if !success {
				t.Error("AddBidOrder() returned false")
			}

			// Verify order was added
			found := ob.SearchById(tt.order.ID)
			if found == nil {
				t.Errorf("Order %d not found after adding", tt.order.ID)
			}
			if found != nil && found.ID != tt.order.ID {
				t.Errorf("Expected order ID %d, got %d", tt.order.ID, found.ID)
			}
		})
	}
}

// TestAddAskOrder tests adding ask orders
func TestAddAskOrder(t *testing.T) {
	ob := matching.NewOrderBook()

	tests := []struct {
		name  string
		order *matching.Order
	}{
		{"SingleAsk", matching.NewOrder(10, "user_test", matching.LimitOrder, matching.Sell, 102.0, 10)},
		{"LowerAsk", matching.NewOrder(11, "user_test", matching.LimitOrder, matching.Sell, 101.0, 20)},
		{"HigherAsk", matching.NewOrder(12, "user_test", matching.LimitOrder, matching.Sell, 103.0, 15)},
		{"SamePriceAsk", matching.NewOrder(13, "user_test", matching.LimitOrder, matching.Sell, 102.0, 5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success := ob.AddAskOrder(tt.order)
			if !success {
				t.Error("AddAskOrder() returned false")
			}

			// Verify order was added
			found := ob.SearchById(tt.order.ID)
			if found == nil {
				t.Errorf("Order %d not found after adding", tt.order.ID)
			}
			if found != nil && found.ID != tt.order.ID {
				t.Errorf("Expected order ID %d, got %d", tt.order.ID, found.ID)
			}
		})
	}
}

// TestGetBestBid tests retrieving the best bid
func TestGetBestBid(t *testing.T) {
	ob := matching.NewOrderBook()

	// Empty book
	price, orders := ob.GetBestBid()
	if price != 0.0 || orders != nil {
		t.Error("Expected empty best bid for empty book")
	}

	// Add bids at different prices
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Buy, 101.0, 20))
	ob.AddBidOrder(matching.NewOrder(3, "user_test", matching.LimitOrder, matching.Buy, 99.0, 15))
	ob.AddBidOrder(matching.NewOrder(4, "user_test", matching.LimitOrder, matching.Buy, 101.0, 5)) // Same as highest

	// Best bid should be 101.0
	price, orders = ob.GetBestBid()
	if price != 101.0 {
		t.Errorf("Expected best bid price 101.0, got %f", price)
	}
	if len(orders) != 2 {
		t.Errorf("Expected 2 orders at best bid, got %d", len(orders))
	}
}

// TestGetBestAsk tests retrieving the best ask
func TestGetBestAsk(t *testing.T) {
	ob := matching.NewOrderBook()

	// Empty book
	price, orders := ob.GetBestAsk()
	if price != 0.0 || orders != nil {
		t.Error("Expected empty best ask for empty book")
	}

	// Add asks at different prices
	ob.AddAskOrder(matching.NewOrder(10, "user_test", matching.LimitOrder, matching.Sell, 102.0, 10))
	ob.AddAskOrder(matching.NewOrder(11, "user_test", matching.LimitOrder, matching.Sell, 101.0, 20))
	ob.AddAskOrder(matching.NewOrder(12, "user_test", matching.LimitOrder, matching.Sell, 103.0, 15))
	ob.AddAskOrder(matching.NewOrder(13, "user_test", matching.LimitOrder, matching.Sell, 101.0, 5)) // Same as lowest

	// Best ask should be 101.0
	price, orders = ob.GetBestAsk()
	if price != 101.0 {
		t.Errorf("Expected best ask price 101.0, got %f", price)
	}
	if len(orders) != 2 {
		t.Errorf("Expected 2 orders at best ask, got %d", len(orders))
	}
}

// TestSearchById tests searching for orders by ID
func TestSearchById(t *testing.T) {
	ob := matching.NewOrderBook()

	// Search in empty book
	found := ob.SearchById(999)
	if found != nil {
		t.Error("Expected nil for non-existent order in empty book")
	}

	// Add some orders
	bid1 := matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10)
	bid2 := matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Buy, 101.0, 20)
	ask1 := matching.NewOrder(10, "user_test", matching.LimitOrder, matching.Sell, 102.0, 10)
	ask2 := matching.NewOrder(11, "user_test", matching.LimitOrder, matching.Sell, 103.0, 15)

	ob.AddBidOrder(bid1)
	ob.AddBidOrder(bid2)
	ob.AddAskOrder(ask1)
	ob.AddAskOrder(ask2)

	// Test finding each order
	tests := []struct {
		name     string
		id       uint64
		expected *matching.Order
	}{
		{"FindBid1", 1, bid1},
		{"FindBid2", 2, bid2},
		{"FindAsk1", 10, ask1},
		{"FindAsk2", 11, ask2},
		{"NotFound", 999, nil},
		{"ZeroID", 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := ob.SearchById(tt.id)
			if tt.expected == nil && found != nil {
				t.Errorf("Expected nil, got order %d", found.ID)
			}
			if tt.expected != nil && found == nil {
				t.Error("Expected to find order, got nil")
			}
			if tt.expected != nil && found != nil && found.ID != tt.expected.ID {
				t.Errorf("Expected order ID %d, got %d", tt.expected.ID, found.ID)
			}
		})
	}
}

// TestGetOrdersByPrice tests retrieving orders at a specific price level
func TestGetOrdersByPrice(t *testing.T) {
	ob := matching.NewOrderBook()

	// Test on empty book
	plo := ob.GetOrdersByPrice(100.0)
	if plo.Bids != nil || plo.Asks != nil {
		t.Error("Expected nil orders for non-existent price level")
	}

	// Add orders at price 100.0
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Buy, 100.0, 20))
	ob.AddAskOrder(matching.NewOrder(10, "user_test", matching.LimitOrder, matching.Sell, 100.0, 15))

	// Test retrieving orders at 100.0
	plo = ob.GetOrdersByPrice(100.0)
	if len(plo.Bids) != 2 {
		t.Errorf("Expected 2 bid orders at 100.0, got %d", len(plo.Bids))
	}
	if len(plo.Asks) != 1 {
		t.Errorf("Expected 1 ask order at 100.0, got %d", len(plo.Asks))
	}

	// Test retrieving orders at non-existent price
	plo = ob.GetOrdersByPrice(200.0)
	if plo.Bids != nil || plo.Asks != nil {
		t.Error("Expected nil orders for non-existent price level 200.0")
	}
}

// TestDeleteBidOrder tests deleting bid orders
func TestDeleteBidOrder(t *testing.T) {
	ob := matching.NewOrderBook()

	// Try to delete from empty book
	success := ob.DeleteBidOrder(999)
	if success {
		t.Error("Expected false when deleting from empty book")
	}

	// Add orders
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Buy, 100.0, 20))
	ob.AddBidOrder(matching.NewOrder(3, "user_test", matching.LimitOrder, matching.Buy, 101.0, 15))

	// Delete order 1
	success = ob.DeleteBidOrder(1)
	if !success {
		t.Error("Expected true when deleting existing order")
	}

	// Verify deletion
	found := ob.SearchById(1)
	if found != nil {
		t.Error("Order should be deleted")
	}

	// Verify other orders at same price still exist
	plo := ob.GetOrdersByPrice(100.0)
	if len(plo.Bids) != 1 {
		t.Errorf("Expected 1 bid remaining at 100.0, got %d", len(plo.Bids))
	}

	// Delete last order at a price level
	ob.DeleteBidOrder(2)
	plo = ob.GetOrdersByPrice(100.0)
	if plo.Bids != nil {
		t.Error("Expected price level to be removed when all orders deleted")
	}
}

// TestDeleteAskOrder tests deleting ask orders
func TestDeleteAskOrder(t *testing.T) {
	ob := matching.NewOrderBook()

	// Try to delete from empty book
	success := ob.DeleteAskOrder(999)
	if success {
		t.Error("Expected false when deleting from empty book")
	}

	// Add orders
	ob.AddAskOrder(matching.NewOrder(10, "user_test", matching.LimitOrder, matching.Sell, 102.0, 10))
	ob.AddAskOrder(matching.NewOrder(11, "user_test", matching.LimitOrder, matching.Sell, 102.0, 20))
	ob.AddAskOrder(matching.NewOrder(12, "user_test", matching.LimitOrder, matching.Sell, 103.0, 15))

	// Delete order 10
	success = ob.DeleteAskOrder(10)
	if !success {
		t.Error("Expected true when deleting existing order")
	}

	// Verify deletion
	found := ob.SearchById(10)
	if found != nil {
		t.Error("Order should be deleted")
	}

	// Verify other orders at same price still exist
	plo := ob.GetOrdersByPrice(102.0)
	if len(plo.Asks) != 1 {
		t.Errorf("Expected 1 ask remaining at 102.0, got %d", len(plo.Asks))
	}

	// Delete last order at a price level
	ob.DeleteAskOrder(11)
	plo = ob.GetOrdersByPrice(102.0)
	if plo.Asks != nil {
		t.Error("Expected price level to be removed when all orders deleted")
	}
}

// TestDeleteOrderById tests deleting orders by ID (either side)
func TestDeleteOrderById(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add both bid and ask orders
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddAskOrder(matching.NewOrder(10, "user_test", matching.LimitOrder, matching.Sell, 102.0, 10))

	// Delete bid order
	success := ob.DeleteOrderById(1)
	if !success {
		t.Error("Expected true when deleting bid order")
	}
	if ob.SearchById(1) != nil {
		t.Error("Bid order should be deleted")
	}

	// Delete ask order
	success = ob.DeleteOrderById(10)
	if !success {
		t.Error("Expected true when deleting ask order")
	}
	if ob.SearchById(10) != nil {
		t.Error("Ask order should be deleted")
	}

	// Delete non-existent order
	success = ob.DeleteOrderById(999)
	if success {
		t.Error("Expected false when deleting non-existent order")
	}
}

// TestDeleteBidBlock tests deleting entire bid price level
func TestDeleteBidBlock(t *testing.T) {
	ob := matching.NewOrderBook()

	// Try to delete non-existent block
	success := ob.DeleteBidBlock(100.0)
	if success {
		t.Error("Expected false when deleting non-existent block")
	}

	// Add multiple orders at same price
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddBidOrder(matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Buy, 100.0, 20))
	ob.AddBidOrder(matching.NewOrder(3, "user_test", matching.LimitOrder, matching.Buy, 100.0, 15))

	// Delete entire block
	success = ob.DeleteBidBlock(100.0)
	if !success {
		t.Error("Expected true when deleting existing block")
	}

	// Verify all orders at that price are gone
	plo := ob.GetOrdersByPrice(100.0)
	if plo.Bids != nil {
		t.Error("Expected price level to be completely removed")
	}

	// Verify individual orders are gone
	if ob.SearchById(1) != nil || ob.SearchById(2) != nil || ob.SearchById(3) != nil {
		t.Error("All orders in block should be deleted")
	}
}

// TestDeleteAskBlock tests deleting entire ask price level
func TestDeleteAskBlock(t *testing.T) {
	ob := matching.NewOrderBook()

	// Try to delete non-existent block
	success := ob.DeleteAskBlock(102.0)
	if success {
		t.Error("Expected false when deleting non-existent block")
	}

	// Add multiple orders at same price
	ob.AddAskOrder(matching.NewOrder(10, "user_test", matching.LimitOrder, matching.Sell, 102.0, 10))
	ob.AddAskOrder(matching.NewOrder(11, "user_test", matching.LimitOrder, matching.Sell, 102.0, 20))
	ob.AddAskOrder(matching.NewOrder(12, "user_test", matching.LimitOrder, matching.Sell, 102.0, 15))

	// Delete entire block
	success = ob.DeleteAskBlock(102.0)
	if !success {
		t.Error("Expected true when deleting existing block")
	}

	// Verify all orders at that price are gone
	plo := ob.GetOrdersByPrice(102.0)
	if plo.Asks != nil {
		t.Error("Expected price level to be completely removed")
	}

	// Verify individual orders are gone
	if ob.SearchById(10) != nil || ob.SearchById(11) != nil || ob.SearchById(12) != nil {
		t.Error("All orders in block should be deleted")
	}
}

// TestPriceTimePriority tests that orders are maintained in time priority at each price
func TestPriceTimePriority(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add orders at same price in sequence
	order1 := matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10)
	order2 := matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Buy, 100.0, 20)
	order3 := matching.NewOrder(3, "user_test", matching.LimitOrder, matching.Buy, 100.0, 15)

	ob.AddBidOrder(order1)
	ob.AddBidOrder(order2)
	ob.AddBidOrder(order3)

	// Get orders at price level
	plo := ob.GetOrdersByPrice(100.0)

	// Verify order sequence (FIFO at price level)
	if len(plo.Bids) != 3 {
		t.Fatalf("Expected 3 orders, got %d", len(plo.Bids))
	}

	// Orders should be in the order they were added
	if plo.Bids[0].ID != 1 || plo.Bids[1].ID != 2 || plo.Bids[2].ID != 3 {
		t.Error("Orders not maintained in time priority (FIFO)")
	}
}

// TestLargeOrderBook tests performance with many orders
func TestLargeOrderBook(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add 1000 bid orders at different prices
	for i := 0; i < 1000; i++ {
		price := 100.0 + float64(i)*0.01
		ob.AddBidOrder(matching.NewOrder(uint64(i), "user_test", matching.LimitOrder, matching.Buy, price, 10))
	}

	// Add 1000 ask orders at different prices
	for i := 1000; i < 2000; i++ {
		price := 110.0 + float64(i-1000)*0.01
		ob.AddAskOrder(matching.NewOrder(uint64(i), "user_test", matching.LimitOrder, matching.Sell, price, 10))
	}

	// Test best bid (should be highest)
	bidPrice, _ := ob.GetBestBid()
	expectedBidPrice := 100.0 + 999*0.01
	if math.Abs(bidPrice-expectedBidPrice) > 0.001 {
		t.Errorf("Expected best bid ~%f, got %f", expectedBidPrice, bidPrice)
	}

	// Test best ask (should be lowest)
	askPrice, _ := ob.GetBestAsk()
	expectedAskPrice := 110.0
	if math.Abs(askPrice-expectedAskPrice) > 0.001 {
		t.Errorf("Expected best ask ~%f, got %f", expectedAskPrice, askPrice)
	}

	// Test search for order in middle
	found := ob.SearchById(500)
	if found == nil {
		t.Error("Should find order 500")
	}

	// Test deletion
	success := ob.DeleteOrderById(750)
	if !success {
		t.Error("Should successfully delete order 750")
	}

	found = ob.SearchById(750)
	if found != nil {
		t.Error("Order 750 should be deleted")
	}
}

// TestMultipleOrdersAtSamePrice tests handling multiple orders at same price
func TestMultipleOrdersAtSamePrice(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add 10 orders at same price
	for i := 0; i < 10; i++ {
		ob.AddBidOrder(matching.NewOrder(uint64(i), "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	}

	plo := ob.GetOrdersByPrice(100.0)
	if len(plo.Bids) != 10 {
		t.Errorf("Expected 10 orders at price 100.0, got %d", len(plo.Bids))
	}

	// Delete middle order
	success := ob.DeleteBidOrder(5)
	if !success {
		t.Error("Should successfully delete order 5")
	}

	plo = ob.GetOrdersByPrice(100.0)
	if len(plo.Bids) != 9 {
		t.Errorf("Expected 9 orders after deletion, got %d", len(plo.Bids))
	}

	// Verify order 5 is not in the list
	for _, order := range plo.Bids {
		if order.ID == 5 {
			t.Error("Order 5 should be deleted")
		}
	}
}

// TestEdgeCasePrices tests edge case price values
func TestEdgeCasePrices(t *testing.T) {
	ob := matching.NewOrderBook()

	// Test very small price
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 0.0001, 10))
	found := ob.SearchById(1)
	if found == nil || found.Price != 0.0001 {
		t.Error("Should handle very small prices")
	}

	// Test very large price
	largePrice := 999999999.99
	ob.AddAskOrder(matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Sell, largePrice, 10))
	found = ob.SearchById(2)
	if found == nil || found.Price != largePrice {
		t.Error("Should handle very large prices")
	}
}

// TestCrossedBook tests when bid price >= ask price
func TestCrossedBook(t *testing.T) {
	ob := matching.NewOrderBook()

	// Create crossed book (this is allowed at orderbook level, engine should handle)
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 102.0, 10))
	ob.AddAskOrder(matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Sell, 100.0, 10))

	bidPrice, _ := ob.GetBestBid()
	askPrice, _ := ob.GetBestAsk()

	if bidPrice < askPrice {
		t.Error("Book should be crossed (bid >= ask)")
	}

	// Both orders should exist
	if ob.SearchById(1) == nil || ob.SearchById(2) == nil {
		t.Error("Both orders should exist in crossed book")
	}
}

// TestEmptyBookOperations tests various operations on empty book
func TestEmptyBookOperations(t *testing.T) {
	ob := matching.NewOrderBook()

	// Test all operations on empty book
	if ob.SearchById(1) != nil {
		t.Error("Search should return nil for empty book")
	}

	if ob.DeleteOrderById(1) {
		t.Error("Delete should return false for empty book")
	}

	if ob.DeleteBidOrder(1) {
		t.Error("DeleteBidOrder should return false for empty book")
	}

	if ob.DeleteAskOrder(1) {
		t.Error("DeleteAskOrder should return false for empty book")
	}

	if ob.DeleteBidBlock(100.0) {
		t.Error("DeleteBidBlock should return false for empty book")
	}

	if ob.DeleteAskBlock(100.0) {
		t.Error("DeleteAskBlock should return false for empty book")
	}

	bidPrice, bidOrders := ob.GetBestBid()
	if bidPrice != 0.0 || bidOrders != nil {
		t.Error("GetBestBid should return 0.0 and nil for empty book")
	}

	askPrice, askOrders := ob.GetBestAsk()
	if askPrice != 0.0 || askOrders != nil {
		t.Error("GetBestAsk should return 0.0 and nil for empty book")
	}

	plo := ob.GetOrdersByPrice(100.0)
	if plo.Bids != nil || plo.Asks != nil {
		t.Error("GetOrdersByPrice should return nil for empty book")
	}
}

// TestDuplicateOrderIDs tests handling of duplicate order IDs
func TestDuplicateOrderIDs(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add two orders with same ID at different prices
	order1 := matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10)
	order2 := matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 101.0, 20)

	ob.AddBidOrder(order1)
	ob.AddBidOrder(order2)

	// Search should find one of them (first one encountered)
	found := ob.SearchById(1)
	if found == nil {
		t.Error("Should find an order with ID 1")
	}

	// Both orders should exist in book at their respective prices
	plo100 := ob.GetOrdersByPrice(100.0)
	plo101 := ob.GetOrdersByPrice(101.0)

	if len(plo100.Bids) != 1 || len(plo101.Bids) != 1 {
		t.Error("Both orders should exist at their respective prices")
	}
}

// TestModifyOrderInBook tests modifying an order after it's been added
func TestModifyOrderInBook(t *testing.T) {
	ob := matching.NewOrderBook()

	order := matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10)
	ob.AddBidOrder(order)

	// Modify the order directly
	order.SetSize(20)

	// Retrieve and verify modification
	found := ob.SearchById(1)
	if found == nil {
		t.Fatal("Order should be found")
	}

	if found.Size != 20 {
		t.Errorf("Expected modified size 20, got %d", found.Size)
	}
}

// TestConcurrentAccess tests basic concurrent operations (not comprehensive)
func TestConcurrentAccess(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add initial orders
	for i := 0; i < 100; i++ {
		ob.AddBidOrder(matching.NewOrder(uint64(i), "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	}

	// Note: This is a basic test. For production, proper synchronization would be needed
	done := make(chan bool, 10)

	// Concurrent searches
	for i := 0; i < 10; i++ {
		go func(id uint64) {
			ob.SearchById(id)
			done <- true
		}(uint64(i))
	}

	// Wait for all searches to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSpreadCalculation tests calculating bid-ask spread
func TestSpreadCalculation(t *testing.T) {
	ob := matching.NewOrderBook()

	// Add orders
	ob.AddBidOrder(matching.NewOrder(1, "user_test", matching.LimitOrder, matching.Buy, 100.0, 10))
	ob.AddAskOrder(matching.NewOrder(2, "user_test", matching.LimitOrder, matching.Sell, 101.0, 10))

	bidPrice, _ := ob.GetBestBid()
	askPrice, _ := ob.GetBestAsk()

	spread := askPrice - bidPrice
	expectedSpread := 1.0

	if math.Abs(spread-expectedSpread) > 0.001 {
		t.Errorf("Expected spread %f, got %f", expectedSpread, spread)
	}
}
