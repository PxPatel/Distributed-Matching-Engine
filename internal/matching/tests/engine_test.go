package matching

import (
	"testing"

	"github.com/PxPatel/trading-system/internal/matching"
)

// TestNewEngine tests the Engine constructor
func TestNewEngine(t *testing.T) {
	engine := matching.NewEngine()

	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
}

// TestPlaceMarketOrderBuy tests placing a market buy order
func TestPlaceMarketOrderBuy(t *testing.T) {
	engine := matching.NewEngine()

	// Add some ask orders (liquidity to buy against)
	ask1 := matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10)
	ask2 := matching.NewOrder(2, matching.LimitOrder, matching.Sell, 102.0, 20)
	ask3 := matching.NewOrder(3, matching.LimitOrder, matching.Sell, 103.0, 15)

	engine.PlaceOrder(ask1)
	engine.PlaceOrder(ask2)
	engine.PlaceOrder(ask3)

	// Place market buy order that fully fills against best ask
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 10)
	trades := engine.PlaceOrder(marketBuy)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}

	if trades[0].Price != 101.0 {
		t.Errorf("Expected trade price 101.0, got %f", trades[0].Price)
	}

	if trades[0].Size != 10 {
		t.Errorf("Expected trade size 10, got %d", trades[0].Size)
	}

	if trades[0].BuyOrderID != 100 || trades[0].SellOrderID != 1 {
		t.Errorf("Trade order IDs incorrect: buy=%d, sell=%d", trades[0].BuyOrderID, trades[0].SellOrderID)
	}
}

// TestPlaceMarketOrderSell tests placing a market sell order
func TestPlaceMarketOrderSell(t *testing.T) {
	engine := matching.NewEngine()

	// Add some bid orders (liquidity to sell against)
	bid1 := matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 10)
	bid2 := matching.NewOrder(2, matching.LimitOrder, matching.Buy, 99.0, 20)
	bid3 := matching.NewOrder(3, matching.LimitOrder, matching.Buy, 98.0, 15)

	engine.PlaceOrder(bid1)
	engine.PlaceOrder(bid2)
	engine.PlaceOrder(bid3)

	// Place market sell order
	marketSell := matching.NewOrder(100, matching.MarketOrder, matching.Sell, 0.0, 10)
	trades := engine.PlaceOrder(marketSell)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}

	if trades[0].Price != 100.0 {
		t.Errorf("Expected trade price 100.0 (best bid), got %f", trades[0].Price)
	}

	if trades[0].Size != 10 {
		t.Errorf("Expected trade size 10, got %d", trades[0].Size)
	}

	if trades[0].BuyOrderID != 1 || trades[0].SellOrderID != 100 {
		t.Errorf("Trade order IDs incorrect: buy=%d, sell=%d", trades[0].BuyOrderID, trades[0].SellOrderID)
	}
}

// TestMarketOrderPartialFill tests market order with partial fills
func TestMarketOrderPartialFill(t *testing.T) {
	engine := matching.NewEngine()

	// Add smaller ask orders
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 5))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Sell, 102.0, 10))
	engine.PlaceOrder(matching.NewOrder(3, matching.LimitOrder, matching.Sell, 103.0, 8))

	// Place market buy that requires multiple fills
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 20)
	trades := engine.PlaceOrder(marketBuy)

	// Should create 3 trades: 5 @ 101, 10 @ 102, 5 @ 103
	if len(trades) != 3 {
		t.Fatalf("Expected 3 trades, got %d", len(trades))
	}

	// Verify first trade
	if trades[0].Price != 101.0 || trades[0].Size != 5 {
		t.Errorf("Trade 0: expected 5@101.0, got %d@%f", trades[0].Size, trades[0].Price)
	}

	// Verify second trade
	if trades[1].Price != 102.0 || trades[1].Size != 10 {
		t.Errorf("Trade 1: expected 10@102.0, got %d@%f", trades[1].Size, trades[1].Price)
	}

	// Verify third trade
	if trades[2].Price != 103.0 || trades[2].Size != 5 {
		t.Errorf("Trade 2: expected 5@103.0, got %d@%f", trades[2].Size, trades[2].Price)
	}

	// Verify total size
	totalSize := trades[0].Size + trades[1].Size + trades[2].Size
	if totalSize != 20 {
		t.Errorf("Expected total size 20, got %d", totalSize)
	}
}

// TestMarketOrderNoLiquidity tests market order with no liquidity
func TestMarketOrderNoLiquidity(t *testing.T) {
	engine := matching.NewEngine()

	// Place market buy with no asks in book
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 10)
	trades := engine.PlaceOrder(marketBuy)

	// Should create no trades
	if len(trades) != 0 {
		t.Errorf("Expected 0 trades with no liquidity, got %d", len(trades))
	}
}

// TestMarketOrderInsufficientLiquidity tests market order with insufficient liquidity
func TestMarketOrderInsufficientLiquidity(t *testing.T) {
	engine := matching.NewEngine()

	// Add limited liquidity
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 5))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Sell, 102.0, 8))

	// Place market buy larger than available liquidity
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 20)
	trades := engine.PlaceOrder(marketBuy)

	// Should only fill what's available: 5 + 8 = 13
	if len(trades) != 2 {
		t.Errorf("Expected 2 trades, got %d", len(trades))
	}

	totalFilled := 0
	for _, trade := range trades {
		totalFilled += trade.Size
	}

	if totalFilled != 13 {
		t.Errorf("Expected total filled 13, got %d", totalFilled)
	}
}

// TestPlaceLimitOrderBuyImmediate tests limit buy that matches immediately
func TestPlaceLimitOrderBuyImmediate(t *testing.T) {
	engine := matching.NewEngine()

	// Add ask order
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10))

	// Place limit buy at or above best ask
	limitBuy := matching.NewOrder(100, matching.LimitOrder, matching.Buy, 101.0, 10)
	trades := engine.PlaceOrder(limitBuy)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}

	if trades[0].Price != 101.0 || trades[0].Size != 10 {
		t.Errorf("Expected 10@101.0, got %d@%f", trades[0].Size, trades[0].Price)
	}
}

// TestPlaceLimitOrderSellImmediate tests limit sell that matches immediately
func TestPlaceLimitOrderSellImmediate(t *testing.T) {
	engine := matching.NewEngine()

	// Add bid order
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 10))

	// Place limit sell at or below best bid
	limitSell := matching.NewOrder(100, matching.LimitOrder, matching.Sell, 100.0, 10)
	trades := engine.PlaceOrder(limitSell)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}

	if trades[0].Price != 100.0 || trades[0].Size != 10 {
		t.Errorf("Expected 10@100.0, got %d@%f", trades[0].Size, trades[0].Price)
	}
}

// TestPlaceLimitOrderAddToBook tests limit order that doesn't match and is added to book
func TestPlaceLimitOrderAddToBook(t *testing.T) {
	engine := matching.NewEngine()

	// Place limit buy below any asks
	limitBuy := matching.NewOrder(100, matching.LimitOrder, matching.Buy, 99.0, 10)
	trades := engine.PlaceOrder(limitBuy)

	// Should create no trades
	if len(trades) != 0 {
		t.Errorf("Expected 0 trades, got %d", len(trades))
	}

	// Place limit sell above any bids (should also be added to book)
	limitSell := matching.NewOrder(101, matching.LimitOrder, matching.Sell, 102.0, 15)
	trades = engine.PlaceOrder(limitSell)

	if len(trades) != 0 {
		t.Errorf("Expected 0 trades, got %d", len(trades))
	}

	// Now place matching orders
	// Market buy should match against the sell we added
	marketBuy := matching.NewOrder(200, matching.MarketOrder, matching.Buy, 0.0, 10)
	trades = engine.PlaceOrder(marketBuy)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade from market buy, got %d", len(trades))
	}

	if trades[0].Price != 102.0 || trades[0].Size != 10 {
		t.Errorf("Expected 10@102.0, got %d@%f", trades[0].Size, trades[0].Price)
	}
}

// TestPlaceLimitOrderPartialFillAndRest tests limit order with partial fill and rest added to book
func TestPlaceLimitOrderPartialFillAndRest(t *testing.T) {
	engine := matching.NewEngine()

	// Add smaller ask order
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 5))

	// Place limit buy that partially matches
	limitBuy := matching.NewOrder(100, matching.LimitOrder, matching.Buy, 101.0, 15)
	trades := engine.PlaceOrder(limitBuy)

	// Should create 1 trade for 5 units
	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}

	if trades[0].Size != 5 {
		t.Errorf("Expected trade size 5, got %d", trades[0].Size)
	}

	// Remaining 10 units should be added to book
	// Place market sell to verify
	marketSell := matching.NewOrder(200, matching.MarketOrder, matching.Sell, 0.0, 8)
	trades = engine.PlaceOrder(marketSell)

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade from market sell, got %d", len(trades))
	}

	if trades[0].Size != 8 {
		t.Errorf("Expected trade size 8, got %d", trades[0].Size)
	}
}

// TestCancelOrder tests canceling an order
func TestCancelOrder(t *testing.T) {
	engine := matching.NewEngine()

	// Place limit order
	limitBuy := matching.NewOrder(100, matching.LimitOrder, matching.Buy, 99.0, 10)
	engine.PlaceOrder(limitBuy)

	// Cancel the order
	success := engine.CancelOrder(100)
	if !success {
		t.Error("Expected successful cancellation")
	}

	// Try to cancel again
	success = engine.CancelOrder(100)
	if success {
		t.Error("Expected failed cancellation of already-canceled order")
	}

	// Verify order is no longer in book
	// Place market sell that would match if order still existed
	marketSell := matching.NewOrder(200, matching.MarketOrder, matching.Sell, 0.0, 5)
	trades := engine.PlaceOrder(marketSell)

	if len(trades) != 0 {
		t.Errorf("Expected 0 trades after order canceled, got %d", len(trades))
	}
}

// TestCancelOrderViaOrderType tests canceling using CancelOrder type
func TestCancelOrderViaOrderType(t *testing.T) {
	engine := matching.NewEngine()

	// Place limit order
	limitBuy := matching.NewOrder(100, matching.LimitOrder, matching.Buy, 99.0, 10)
	engine.PlaceOrder(limitBuy)

	// Cancel using CancelOrder type
	cancelOrder := matching.NewOrder(100, matching.CancelOrder, matching.Buy, 0.0, 0)
	trades := engine.PlaceOrder(cancelOrder)

	// Cancel orders don't produce trades
	if trades != nil {
		t.Errorf("Expected nil trades for cancel order, got %v", trades)
	}
}

// TestCancelNonExistentOrder tests canceling an order that doesn't exist
func TestCancelNonExistentOrder(t *testing.T) {
	engine := matching.NewEngine()

	success := engine.CancelOrder(999)
	if success {
		t.Error("Expected failed cancellation of non-existent order")
	}
}

// TestPricePriority tests that better prices match first
func TestPricePriority(t *testing.T) {
	engine := matching.NewEngine()

	// Add asks at different prices
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 103.0, 10))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Sell, 101.0, 10))
	engine.PlaceOrder(matching.NewOrder(3, matching.LimitOrder, matching.Sell, 102.0, 10))

	// Market buy should match with best (lowest) ask first
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 10)
	trades := engine.PlaceOrder(marketBuy)

	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	if trades[0].Price != 101.0 {
		t.Errorf("Expected to match best ask 101.0, got %f", trades[0].Price)
	}

	if trades[0].SellOrderID != 2 {
		t.Errorf("Expected to match order 2, got order %d", trades[0].SellOrderID)
	}
}

// TestTimePriority tests that earlier orders at same price match first
func TestTimePriority(t *testing.T) {
	engine := matching.NewEngine()

	// Add multiple asks at same price
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 5))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Sell, 101.0, 5))
	engine.PlaceOrder(matching.NewOrder(3, matching.LimitOrder, matching.Sell, 101.0, 5))

	// Market buy should match in time priority (FIFO)
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 5)
	trades := engine.PlaceOrder(marketBuy)

	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	// Should match with first order (ID 1)
	if trades[0].SellOrderID != 1 {
		t.Errorf("Expected to match first order (ID 1), got order %d", trades[0].SellOrderID)
	}
}

// TestMultipleTrades tests placing an order that generates multiple trades
func TestMultipleTrades(t *testing.T) {
	engine := matching.NewEngine()

	// Add multiple asks at different prices
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Sell, 102.0, 15))
	engine.PlaceOrder(matching.NewOrder(3, matching.LimitOrder, matching.Sell, 103.0, 20))

	// Large market buy
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 40)
	trades := engine.PlaceOrder(marketBuy)

	// Should create 3 trades
	if len(trades) != 3 {
		t.Fatalf("Expected 3 trades, got %d", len(trades))
	}

	// Verify trade details
	expectedTrades := []struct {
		price float64
		size  int
	}{
		{101.0, 10},
		{102.0, 15},
		{103.0, 15}, // Only 15 of the 20 available
	}

	for i, expected := range expectedTrades {
		if trades[i].Price != expected.price {
			t.Errorf("Trade %d: expected price %f, got %f", i, expected.price, trades[i].Price)
		}
		if trades[i].Size != expected.size {
			t.Errorf("Trade %d: expected size %d, got %d", i, expected.size, trades[i].Size)
		}
	}
}

// TestLimitOrderPriceImprovement tests that limit orders get price improvement
func TestLimitOrderPriceImprovement(t *testing.T) {
	engine := matching.NewEngine()

	// Add ask at 101.0
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10))

	// Place limit buy willing to pay up to 105.0
	limitBuy := matching.NewOrder(100, matching.LimitOrder, matching.Buy, 105.0, 10)
	trades := engine.PlaceOrder(limitBuy)

	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	// Should execute at resting order price (101.0), not limit price (105.0)
	if trades[0].Price != 101.0 {
		t.Errorf("Expected price improvement to 101.0, got %f", trades[0].Price)
	}
}

// TestAggressiveLimitOrders tests limit orders that cross the spread
func TestAggressiveLimitOrders(t *testing.T) {
	engine := matching.NewEngine()

	// Create spread: bids at 99-100, asks at 102-103
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 10))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Buy, 99.0, 10))
	engine.PlaceOrder(matching.NewOrder(3, matching.LimitOrder, matching.Sell, 102.0, 10))
	engine.PlaceOrder(matching.NewOrder(4, matching.LimitOrder, matching.Sell, 103.0, 10))

	// Aggressive limit buy crosses the spread
	limitBuy := matching.NewOrder(100, matching.LimitOrder, matching.Buy, 102.5, 15)
	trades := engine.PlaceOrder(limitBuy)

	// Should match only one ask
	if len(trades) != 1 {
		t.Errorf("Expected 1 trades, got %d", len(trades))
	}

	// First trade at 102.0 (best ask)
	if trades[0].Price != 102.0 || trades[0].Size != 10 {
		t.Errorf("Trade 0: expected 10@102.0, got %d@%f", trades[0].Size, trades[0].Price)
	}
}

// FIXME: Determine the handling for order matching between the same participant
// TestSelfMatch tests that orders from same participant can match (engine allows this)
// func TestSelfMatch(t *testing.T) {
// 	engine := matching.NewEngine()

// 	// Place bid and ask with same ID prefix (simulating same participant)
// 	// Note: The engine doesn't prevent self-matching - that's typically handled at a higher level
// 	engine.PlaceOrder(matching.NewOrder(100, matching.LimitOrder, matching.Buy, 101.0, 10))
// 	engine.PlaceOrder(matching.NewOrder(101, matching.LimitOrder, matching.Sell, 101.0, 10))

// 	// These should match
// 	trades := engine.PlaceOrder(matching.NewOrder(102, matching.MarketOrder, matching.Sell, 0.0, 5))

// 	if len(trades) == 0 {
// 		t.Error("Expected orders to match (engine doesn't prevent self-matching)")
// 	}
// }

// TestFullBookExecution tests a complex scenario with multiple orders
func TestFullBookExecution(t *testing.T) {
	engine := matching.NewEngine()

	// Build a realistic order book
	// Bids: 100(10), 99(20), 98(30)
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 10))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Buy, 99.0, 20))
	engine.PlaceOrder(matching.NewOrder(3, matching.LimitOrder, matching.Buy, 98.0, 30))

	// Asks: 101(15), 102(25), 103(35)
	engine.PlaceOrder(matching.NewOrder(11, matching.LimitOrder, matching.Sell, 101.0, 15))
	engine.PlaceOrder(matching.NewOrder(12, matching.LimitOrder, matching.Sell, 102.0, 25))
	engine.PlaceOrder(matching.NewOrder(13, matching.LimitOrder, matching.Sell, 103.0, 35))

	// Large market buy: should sweep through all asks
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 70)
	trades := engine.PlaceOrder(marketBuy)

	// Should match all asks: 15 + 25 + 30 = 70
	if len(trades) != 3 {
		t.Errorf("Expected 3 trades, got %d", len(trades))
	}

	totalFilled := 0
	for _, trade := range trades {
		totalFilled += trade.Size
	}

	if totalFilled != 70 {
		t.Errorf("Expected total filled 70, got %d", totalFilled)
	}

	// Now place large market sell: should sweep through all bids
	marketSell := matching.NewOrder(200, matching.MarketOrder, matching.Sell, 0.0, 60)
	trades = engine.PlaceOrder(marketSell)

	// Should match all bids: 10 + 20 + 30 = 60
	if len(trades) != 3 {
		t.Errorf("Expected 3 trades, got %d", len(trades))
	}

	totalFilled = 0
	for _, trade := range trades {
		totalFilled += trade.Size
	}

	if totalFilled != 60 {
		t.Errorf("Expected total filled 60, got %d", totalFilled)
	}
}

// TestOrderSizeReduction tests that matched orders have their size reduced correctly
func TestOrderSizeReduction(t *testing.T) {
	engine := matching.NewEngine()

	// Add ask order
	ask := matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 20)
	engine.PlaceOrder(ask)

	// Partially fill with market buy
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 8)
	trades := engine.PlaceOrder(marketBuy)

	if len(trades) != 1 || trades[0].Size != 8 {
		t.Fatal("Expected 1 trade of size 8")
	}

	// Place another market buy to verify remaining size
	marketBuy2 := matching.NewOrder(101, matching.MarketOrder, matching.Buy, 0.0, 12)
	trades2 := engine.PlaceOrder(marketBuy2)

	if len(trades2) != 1 || trades2[0].Size != 12 {
		t.Errorf("Expected 1 trade of size 12, got %d trades with size %d",
			len(trades2), trades2[0].Size)
	}
}

// TestEmptyEngineOperations tests operations on empty engine
func TestEmptyEngineOperations(t *testing.T) {
	engine := matching.NewEngine()

	// Market order with no liquidity
	marketBuy := matching.NewOrder(1, matching.MarketOrder, matching.Buy, 0.0, 10)
	trades := engine.PlaceOrder(marketBuy)

	if len(trades) != 0 {
		t.Error("Expected no trades on empty book")
	}

	// Limit order should be added
	limitBuy := matching.NewOrder(2, matching.LimitOrder, matching.Buy, 100.0, 10)
	trades = engine.PlaceOrder(limitBuy)

	if len(trades) != 0 {
		t.Error("Expected no trades, order should be added to book")
	}

	// Cancel non-existent order
	success := engine.CancelOrder(999)
	if success {
		t.Error("Expected false when canceling non-existent order")
	}
}

// TestNoActionOrderType tests that NoActionOrder type doesn't execute
func TestNoActionOrderType(t *testing.T) {
	engine := matching.NewEngine()

	// Place NoActionOrder
	noAction := matching.NewOrder(1, matching.NoActionOrder, matching.Buy, 100.0, 10)
	trades := engine.PlaceOrder(noAction)

	// Should return nil (default case in switch)
	if trades != nil {
		t.Error("Expected nil trades for NoActionOrder type")
	}
}

// TestZeroSizeOrder tests orders with zero size
func TestZeroSizeOrder(t *testing.T) {
	engine := matching.NewEngine()

	// Add ask order
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10))

	// Market buy with zero size
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 0)
	trades := engine.PlaceOrder(marketBuy)

	// Should produce no trades
	if len(trades) != 0 {
		t.Error("Expected no trades for zero size order")
	}
}

// TestLargeOrderExecution tests execution of very large orders
func TestLargeOrderExecution(t *testing.T) {
	engine := matching.NewEngine()

	// Add many small asks
	for i := 0; i < 100; i++ {
		price := 101.0 + float64(i)*0.01
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Sell, price, 10))
	}

	// Large market buy
	marketBuy := matching.NewOrder(10000, matching.MarketOrder, matching.Buy, 0.0, 1000)
	trades := engine.PlaceOrder(marketBuy)

	// Should create 100 trades
	if len(trades) != 100 {
		t.Errorf("Expected 100 trades, got %d", len(trades))
	}

	// Verify total filled
	totalFilled := 0
	for _, trade := range trades {
		totalFilled += trade.Size
	}

	if totalFilled != 1000 {
		t.Errorf("Expected total filled 1000, got %d", totalFilled)
	}
}

// TestTradeTimestamps tests that trades have timestamps
func TestTradeTimestamps(t *testing.T) {
	engine := matching.NewEngine()

	// Add ask
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10))

	// Execute market buy
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 10)
	trades := engine.PlaceOrder(marketBuy)

	if len(trades) != 1 {
		t.Fatal("Expected 1 trade")
	}

	// Verify timestamp is set
	if trades[0].Timestamp.IsZero() {
		t.Error("Trade timestamp should be set")
	}
}

// TestSequentialTrades tests multiple sequential trades
func TestSequentialTrades(t *testing.T) {
	engine := matching.NewEngine()

	// Place initial orders
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 10))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Sell, 101.0, 10))

	// Execute multiple trades in sequence
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			// Market sell
			sell := matching.NewOrder(uint64(100+i), matching.MarketOrder, matching.Sell, 0.0, 5)
			trades := engine.PlaceOrder(sell)
			if len(trades) > 1 {
				t.Errorf("Trade %d: expected at most 1 trade, got %d", i, len(trades))
			}
			// Add new bid
			engine.PlaceOrder(matching.NewOrder(uint64(10+i), matching.LimitOrder, matching.Buy, 100.0, 5))
		} else {
			// Market buy
			buy := matching.NewOrder(uint64(100+i), matching.MarketOrder, matching.Buy, 0.0, 5)
			trades := engine.PlaceOrder(buy)
			if len(trades) > 1 {
				t.Errorf("Trade %d: expected at most 1 trade, got %d", i, len(trades))
			}
			// Add new ask
			engine.PlaceOrder(matching.NewOrder(uint64(10+i), matching.LimitOrder, matching.Sell, 101.0, 5))
		}
	}
}

// TestEdgeCaseExactFill tests exact fills at various levels
func TestEdgeCaseExactFill(t *testing.T) {
	engine := matching.NewEngine()

	// Add asks with specific sizes
	engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10))
	engine.PlaceOrder(matching.NewOrder(2, matching.LimitOrder, matching.Sell, 102.0, 20))
	engine.PlaceOrder(matching.NewOrder(3, matching.LimitOrder, matching.Sell, 103.0, 30))

	// Market buy that exactly fills first two levels
	marketBuy := matching.NewOrder(100, matching.MarketOrder, matching.Buy, 0.0, 30)
	trades := engine.PlaceOrder(marketBuy)

	if len(trades) != 2 {
		t.Errorf("Expected 2 trades, got %d", len(trades))
	}

	// Verify the third ask is still available
	marketBuy2 := matching.NewOrder(101, matching.MarketOrder, matching.Buy, 0.0, 30)
	trades2 := engine.PlaceOrder(marketBuy2)

	if len(trades2) != 1 {
		t.Errorf("Expected 1 trade with third ask, got %d", len(trades2))
	}

	if trades2[0].SellOrderID != 3 {
		t.Error("Should match with third ask order")
	}
}

// TestStopOrderTypes tests that stop orders don't execute (not implemented yet)
func TestStopOrderTypes(t *testing.T) {
	engine := matching.NewEngine()

	// Place stop market order (not implemented, should return nil)
	stopMarket := matching.NewOrder(1, matching.StopMarketOrder, matching.Buy, 0.0, 10)
	trades := engine.PlaceOrder(stopMarket)

	if trades != nil {
		t.Error("Stop orders not implemented, should return nil")
	}

	// Place stop limit order
	stopLimit := matching.NewOrder(2, matching.StopLimitOrder, matching.Buy, 100.0, 10)
	trades = engine.PlaceOrder(stopLimit)

	if trades != nil {
		t.Error("Stop orders not implemented, should return nil")
	}
}

// TestConcurrentOrders tests basic concurrent order placement
func TestConcurrentOrders(t *testing.T) {
	engine := matching.NewEngine()

	// Note: This is a basic test. For production, proper synchronization would be needed
	done := make(chan bool, 100)

	// Add initial liquidity
	for i := 0; i < 50; i++ {
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Sell, 101.0+float64(i)*0.01, 10))
	}

	// Concurrent market buys
	for i := 0; i < 100; i++ {
		go func(id uint64) {
			marketBuy := matching.NewOrder(id+1000, matching.MarketOrder, matching.Buy, 0.0, 1)
			engine.PlaceOrder(marketBuy)
			done <- true
		}(uint64(i))
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}
