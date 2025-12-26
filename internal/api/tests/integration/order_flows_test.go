package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/PxPatel/trading-system/internal/api/models"
	"github.com/PxPatel/trading-system/internal/api/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleMarketOrderFlow tests a basic market order execution flow
func TestSimpleMarketOrderFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Step 1: Place limit sell orders to create liquidity
	sell1 := ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 100.0, 10))
	require.Equal(t, http.StatusOK, sell1.StatusCode)

	sell2 := ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 101.0, 20))
	require.Equal(t, http.StatusOK, sell2.StatusCode)

	// Step 2: Place market buy order that should match
	buy := ts.Post("/api/v1/orders", testutils.NewMarketBuyOrder("bob", 10))
	require.Equal(t, http.StatusOK, buy.StatusCode)

	var buyResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, buy, &buyResp)

	// Assertions
	assert.True(t, buyResp.Success)
	assert.NotZero(t, buyResp.OrderID)
	assert.Len(t, buyResp.Trades, 1, "Should have 1 trade")
	assert.Equal(t, 100.0, buyResp.Trades[0].Price, "Should execute at best ask price")
	assert.Equal(t, 10, buyResp.Trades[0].Quantity)

	// Step 3: Verify orderbook still has the second sell order
	bidLevels, askLevels := ts.GetOrderBookDepth()
	assert.Equal(t, 0, bidLevels, "No bids should remain")
	assert.Equal(t, 1, askLevels, "One ask level should remain")
}

// TestLimitOrderAddToBookFlow tests limit orders being added to the book
func TestLimitOrderAddToBookFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Place limit buy order below market
	buy1 := ts.Post("/api/v1/orders", testutils.NewLimitBuyOrder("alice", 99.0, 10))
	require.Equal(t, http.StatusOK, buy1.StatusCode)

	var buyResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, buy1, &buyResp)

	assert.True(t, buyResp.Success)
	assert.Len(t, buyResp.Trades, 0, "Should not match immediately")

	// Place limit sell order above market
	sell1 := ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("bob", 101.0, 20))
	require.Equal(t, http.StatusOK, sell1.StatusCode)

	// Verify orderbook
	bidLevels, askLevels := ts.GetOrderBookDepth()
	assert.Equal(t, 1, bidLevels)
	assert.Equal(t, 1, askLevels)

	// Verify via API
	obResp := ts.Get("/api/v1/orderbook")
	require.Equal(t, http.StatusOK, obResp.StatusCode)

	var ob models.OrderBookResponse
	testutils.DecodeJSON(t, obResp, &ob)

	assert.True(t, ob.Success)
	assert.Len(t, ob.Bids, 1)
	assert.Len(t, ob.Asks, 1)
	assert.Equal(t, 99.0, ob.Bids[0].Price)
	assert.Equal(t, 101.0, ob.Asks[0].Price)
	assert.Equal(t, 2.0, ob.Spread)
	assert.Equal(t, 100.0, ob.MidPrice)
}

// TestAggressiveLimitOrderFlow tests limit orders that match immediately
func TestAggressiveLimitOrderFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Place limit sell at 100
	sell := ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 100.0, 15))
	require.Equal(t, http.StatusOK, sell.StatusCode)

	// Place aggressive limit buy at 100 (should match)
	buy := ts.Post("/api/v1/orders", testutils.NewLimitBuyOrder("bob", 100.0, 10))
	require.Equal(t, http.StatusOK, buy.StatusCode)

	var buyResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, buy, &buyResp)

	assert.True(t, buyResp.Success)
	assert.Len(t, buyResp.Trades, 1)
	assert.Equal(t, 100.0, buyResp.Trades[0].Price)
	assert.Equal(t, 10, buyResp.Trades[0].Quantity)

	// Verify remaining quantity in orderbook
	obResp := ts.Get("/api/v1/orderbook")
	var ob models.OrderBookResponse
	testutils.DecodeJSON(t, obResp, &ob)

	assert.Len(t, ob.Asks, 1)
	assert.Equal(t, 5, ob.Asks[0].Quantity, "Remaining 5 units should be in book")
}

// TestPartialFillFlow tests orders that are partially filled
func TestPartialFillFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Place small sell orders
	ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 100.0, 5))
	ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("bob", 101.0, 8))

	// Place large market buy that will partially fill
	buy := ts.Post("/api/v1/orders", testutils.NewMarketBuyOrder("charlie", 20))
	require.Equal(t, http.StatusOK, buy.StatusCode)

	var buyResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, buy, &buyResp)

	assert.True(t, buyResp.Success)
	assert.Len(t, buyResp.Trades, 2, "Should have 2 trades")

	// Verify total filled quantity
	totalFilled := buyResp.Trades[0].Quantity + buyResp.Trades[1].Quantity
	assert.Equal(t, 13, totalFilled, "Should fill 5 + 8 = 13")

	// Verify trades persist to disk
	persistedTrades := ts.ReadTradeLog()
	assert.Len(t, persistedTrades, 2, "Trades should be persisted")
}

// TestOrderCancellationFlow tests cancelling orders
func TestOrderCancellationFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Place limit order
	resp := ts.Post("/api/v1/orders", testutils.NewLimitBuyOrder("alice", 99.0, 10))
	var orderResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, resp, &orderResp)
	orderID := orderResp.OrderID

	// Verify order is in book
	assert.Equal(t, 1, ts.GetTrackedOrderCount())

	// Cancel the order
	cancelResp := ts.Delete(fmt.Sprintf("/api/v1/orders/%d", orderID))
	require.Equal(t, http.StatusOK, cancelResp.StatusCode)

	var cancelResult models.CancelOrderResponse
	testutils.DecodeJSON(t, cancelResp, &cancelResult)
	assert.True(t, cancelResult.Success)

	// Verify order removed
	bidLevels, _ := ts.GetOrderBookDepth()
	assert.Equal(t, 0, bidLevels, "Order should be removed from book")
}

// TestBatchOrderFlow tests submitting multiple orders at once
func TestBatchOrderFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Submit batch with mix of valid and invalid orders
	batch := testutils.NewBatchRequest(
		testutils.NewLimitBuyOrder("alice", 99.0, 10),
		testutils.NewLimitSellOrder("bob", 101.0, 20),
		testutils.NewLimitBuyOrder("charlie", -5.0, 5), // Invalid price
		testutils.NewLimitSellOrder("dave", 102.0, 15),
	)

	resp := ts.Post("/api/v1/orders/batch", batch)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var batchResp models.BatchOrderResponse
	testutils.DecodeJSON(t, resp, &batchResp)

	assert.True(t, batchResp.Success)
	assert.Equal(t, 4, batchResp.Summary.Total)
	assert.Equal(t, 3, batchResp.Summary.Successful)
	assert.Equal(t, 1, batchResp.Summary.Failed)

	// Verify results
	assert.True(t, batchResp.Results[0].Success)
	assert.True(t, batchResp.Results[1].Success)
	assert.False(t, batchResp.Results[2].Success, "Invalid order should fail")
	assert.True(t, batchResp.Results[3].Success)
}

// TestPriceTimePriorityFlow tests FIFO ordering at same price
func TestPriceTimePriorityFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Place multiple sell orders at same price (time priority)
	resp1 := ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 100.0, 5))
	var order1 models.SubmitOrderResponse
	testutils.DecodeJSON(t, resp1, &order1)

	resp2 := ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("bob", 100.0, 8))
	var order2 models.SubmitOrderResponse
	testutils.DecodeJSON(t, resp2, &order2)

	resp3 := ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("charlie", 100.0, 12))
	var order3 models.SubmitOrderResponse
	testutils.DecodeJSON(t, resp3, &order3)

	// Place market buy that fills first order completely
	buy := ts.Post("/api/v1/orders", testutils.NewMarketBuyOrder("dave", 5))
	var buyResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, buy, &buyResp)

	// Should match Alice's order (first in time)
	assert.Len(t, buyResp.Trades, 1)
	assert.Equal(t, order1.OrderID, buyResp.Trades[0].SellOrderID)
}

// TestCrossedOrderBookFlow tests when buy price >= sell price
func TestCrossedOrderBookFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Place sell at 100
	ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 100.0, 10))

	// Place aggressive buy at 105 (above ask price)
	buy := ts.Post("/api/v1/orders", testutils.NewLimitBuyOrder("bob", 105.0, 10))
	var buyResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, buy, &buyResp)

	// Should match at seller's price (100.0), not buyer's price
	assert.Len(t, buyResp.Trades, 1)
	assert.Equal(t, 100.0, buyResp.Trades[0].Price, "Should execute at resting order price")
	assert.Equal(t, 10, buyResp.Trades[0].Quantity)

	// Book should be empty
	bidLevels, askLevels := ts.GetOrderBookDepth()
	assert.Equal(t, 0, bidLevels)
	assert.Equal(t, 0, askLevels)
}

// TestMultiLevelExecutionFlow tests sweeping through multiple price levels
func TestMultiLevelExecutionFlow(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Create orderbook with multiple price levels
	ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 100.0, 5))
	ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("bob", 101.0, 10))
	ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("charlie", 102.0, 8))

	// Large market buy that sweeps multiple levels
	buy := ts.Post("/api/v1/orders", testutils.NewMarketBuyOrder("dave", 18))
	var buyResp models.SubmitOrderResponse
	testutils.DecodeJSON(t, buy, &buyResp)

	assert.True(t, buyResp.Success)
	assert.Len(t, buyResp.Trades, 3, "Should match 3 price levels")

	// Verify trade prices (price improvement - executes at resting order prices)
	assert.Equal(t, 100.0, buyResp.Trades[0].Price)
	assert.Equal(t, 5, buyResp.Trades[0].Quantity)
	assert.Equal(t, 101.0, buyResp.Trades[1].Price)
	assert.Equal(t, 10, buyResp.Trades[1].Quantity)
	assert.Equal(t, 102.0, buyResp.Trades[2].Price)
	assert.Equal(t, 3, buyResp.Trades[2].Quantity)

	// Verify remaining asks
	obResp := ts.Get("/api/v1/orderbook")
	var ob models.OrderBookResponse
	testutils.DecodeJSON(t, obResp, &ob)

	assert.Len(t, ob.Asks, 1, "One ask level should remain")
	assert.Equal(t, 102.0, ob.Asks[0].Price)
	assert.Equal(t, 5, ob.Asks[0].Quantity, "5 units remain from original 8")
}
