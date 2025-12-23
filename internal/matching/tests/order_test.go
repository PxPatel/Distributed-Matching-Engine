package matching

import (
	"testing"
	"time"

	"github.com/PxPatel/trading-system/internal/matching"
)

// TestOrderTypeConstants tests that order type constants are correctly defined
func TestOrderTypeConstants(t *testing.T) {
	tests := []struct {
		name      string
		orderType matching.OrderType
		expected  int
	}{
		{"NoActionOrder", matching.NoActionOrder, 0},
		{"MarketOrder", matching.MarketOrder, 1},
		{"LimitOrder", matching.LimitOrder, 2},
		{"CancelOrder", matching.CancelOrder, 3},
		{"StopMarketOrder", matching.StopMarketOrder, 4},
		{"StopLimitOrder", matching.StopLimitOrder, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.orderType) != tt.expected {
				t.Errorf("Expected %s to be %d, got %d", tt.name, tt.expected, int(tt.orderType))
			}
		})
	}
}

// TestSideTypeConstants tests that side type constants are correctly defined
func TestSideTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		sideType matching.SideType
		expected int
	}{
		{"NoActionSide", matching.NoActionSide, 0},
		{"Buy", matching.Buy, 1},
		{"Sell", matching.Sell, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.sideType) != tt.expected {
				t.Errorf("Expected %s to be %d, got %d", tt.name, tt.expected, int(tt.sideType))
			}
		})
	}
}

// TestNewOrder tests the NewOrder constructor
func TestNewOrder(t *testing.T) {
	tests := []struct {
		name      string
		id        uint64
		orderType matching.OrderType
		side      matching.SideType
		price     float64
		quantity  int
	}{
		{"ValidLimitBuy", 1, matching.LimitOrder, matching.Buy, 100.0, 10},
		{"ValidLimitSell", 2, matching.LimitOrder, matching.Sell, 101.0, 20},
		{"ValidMarketBuy", 3, matching.MarketOrder, matching.Buy, 0.0, 15},
		{"ValidMarketSell", 4, matching.MarketOrder, matching.Sell, 0.0, 25},
		{"LargeQuantity", 5, matching.LimitOrder, matching.Buy, 99.5, 1000000},
		{"SmallPrice", 6, matching.LimitOrder, matching.Sell, 0.01, 100},
		{"HighPrice", 7, matching.LimitOrder, matching.Buy, 999999.99, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			order := matching.NewOrder(tt.id, tt.orderType, tt.side, tt.price, tt.quantity)
			after := time.Now()

			if order.ID != tt.id {
				t.Errorf("Expected ID %d, got %d", tt.id, order.ID)
			}
			if order.OrderType != tt.orderType {
				t.Errorf("Expected OrderType %d, got %d", tt.orderType, order.OrderType)
			}
			if order.Side != tt.side {
				t.Errorf("Expected Side %d, got %d", tt.side, order.Side)
			}
			if order.Price != tt.price {
				t.Errorf("Expected Price %f, got %f", tt.price, order.Price)
			}
			if order.Size != tt.quantity {
				t.Errorf("Expected Size %d, got %d", tt.quantity, order.Size)
			}
			if order.Symbol != "COOTX" {
				t.Errorf("Expected Symbol COOTX, got %s", order.Symbol)
			}
			if order.TimeStamp.Before(before) || order.TimeStamp.After(after) {
				t.Errorf("TimeStamp not set correctly: %v (expected between %v and %v)",
					order.TimeStamp, before, after)
			}
		})
	}
}

// TestOrderIsValid tests the IsValid method with various order configurations
func TestOrderIsValid(t *testing.T) {
	tests := []struct {
		name      string
		order     *matching.Order
		wantValid bool
	}{
		{
			name: "ValidLimitBuy",
			order: &matching.Order{
				ID:        1,
				OrderType: matching.LimitOrder,
				Side:      matching.Buy,
				Price:     100.0,
				Size:      10,
			},
			wantValid: true,
		},
		{
			name: "ValidLimitSell",
			order: &matching.Order{
				ID:        2,
				OrderType: matching.LimitOrder,
				Side:      matching.Sell,
				Price:     101.0,
				Size:      20,
			},
			wantValid: true,
		},
		{
			name: "ValidMarketBuy",
			order: &matching.Order{
				ID:        3,
				OrderType: matching.MarketOrder,
				Side:      matching.Buy,
				Price:     0.0,
				Size:      15,
			},
			wantValid: true,
		},
		{
			name: "ValidMarketSell",
			order: &matching.Order{
				ID:        4,
				OrderType: matching.MarketOrder,
				Side:      matching.Sell,
				Price:     0.0,
				Size:      25,
			},
			wantValid: true,
		},
		{
			name: "InvalidNoActionOrder",
			order: &matching.Order{
				ID:        5,
				OrderType: matching.NoActionOrder,
				Side:      matching.Buy,
				Price:     100.0,
				Size:      10,
			},
			wantValid: false,
		},
		{
			name: "InvalidNoActionSide",
			order: &matching.Order{
				ID:        6,
				OrderType: matching.LimitOrder,
				Side:      matching.NoActionSide,
				Price:     100.0,
				Size:      10,
			},
			wantValid: false,
		},
		{
			name: "InvalidZeroSize",
			order: &matching.Order{
				ID:        7,
				OrderType: matching.LimitOrder,
				Side:      matching.Buy,
				Price:     100.0,
				Size:      0,
			},
			wantValid: false,
		},
		{
			name: "InvalidNegativeSize",
			order: &matching.Order{
				ID:        8,
				OrderType: matching.LimitOrder,
				Side:      matching.Buy,
				Price:     100.0,
				Size:      -10,
			},
			wantValid: false,
		},
		{
			name: "InvalidLimitOrderZeroPrice",
			order: &matching.Order{
				ID:        9,
				OrderType: matching.LimitOrder,
				Side:      matching.Buy,
				Price:     0.0,
				Size:      10,
			},
			wantValid: false,
		},
		{
			name: "InvalidLimitOrderNegativePrice",
			order: &matching.Order{
				ID:        10,
				OrderType: matching.LimitOrder,
				Side:      matching.Sell,
				Price:     -50.0,
				Size:      10,
			},
			wantValid: false,
		},
		{
			name: "ValidStopMarketOrder",
			order: &matching.Order{
				ID:        11,
				OrderType: matching.StopMarketOrder,
				Side:      matching.Buy,
				StopPrice: 105.0,
				Size:      10,
			},
			wantValid: true,
		},
		{
			name: "ValidStopLimitOrder",
			order: &matching.Order{
				ID:        12,
				OrderType: matching.StopLimitOrder,
				Side:      matching.Sell,
				Price:     100.0,
				StopPrice: 95.0,
				Size:      10,
			},
			wantValid: true,
		},
		{
			name: "ValidCancelOrder",
			order: &matching.Order{
				ID:        13,
				OrderType: matching.CancelOrder,
				Side:      matching.Buy,
				Size:      1,
			},
			wantValid: true,
		},
		{
			name: "EdgeCaseVerySmallPrice",
			order: &matching.Order{
				ID:        14,
				OrderType: matching.LimitOrder,
				Side:      matching.Buy,
				Price:     0.0001,
				Size:      10,
			},
			wantValid: true,
		},
		{
			name: "EdgeCaseVeryLargePrice",
			order: &matching.Order{
				ID:        15,
				OrderType: matching.LimitOrder,
				Side:      matching.Sell,
				Price:     999999999.99,
				Size:      1,
			},
			wantValid: true,
		},
		{
			name: "EdgeCaseVeryLargeQuantity",
			order: &matching.Order{
				ID:        16,
				OrderType: matching.MarketOrder,
				Side:      matching.Buy,
				Size:      2147483647, // Max int32
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.order.IsValid()
			if got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v for order: %+v", got, tt.wantValid, tt.order)
			}
		})
	}
}

// TestOrderSetSize tests the SetSize method
func TestOrderSetSize(t *testing.T) {
	tests := []struct {
		name         string
		initialSize  int
		newSize      int
		expectedSize int
	}{
		{"IncreaseSize", 10, 20, 20},
		{"DecreaseSize", 20, 10, 10},
		{"SetToZero", 10, 0, 0},
		{"NegativeSizeRejected", 10, -5, 10}, // Should not change
		{"SetToSameValue", 15, 15, 15},
		{"SetToLargeValue", 5, 1000000, 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, tt.initialSize)
			order.SetSize(tt.newSize)

			if order.Size != tt.expectedSize {
				t.Errorf("Expected Size %d, got %d", tt.expectedSize, order.Size)
			}
		})
	}
}

// TestOrderImmutability tests that order fields can be accessed
func TestOrderFieldAccess(t *testing.T) {
	order := matching.NewOrder(12345, matching.LimitOrder, matching.Buy, 99.99, 100)

	// Test all fields are accessible
	if order.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", order.ID)
	}
	if order.Symbol != "COOTX" {
		t.Errorf("Expected Symbol COOTX, got %s", order.Symbol)
	}
	if order.OrderType != matching.LimitOrder {
		t.Errorf("Expected OrderType LimitOrder, got %d", order.OrderType)
	}
	if order.Side != matching.Buy {
		t.Errorf("Expected Side Buy, got %d", order.Side)
	}
	if order.Price != 99.99 {
		t.Errorf("Expected Price 99.99, got %f", order.Price)
	}
	if order.Size != 100 {
		t.Errorf("Expected Size 100, got %d", order.Size)
	}
	if order.TimeStamp.IsZero() {
		t.Error("Expected TimeStamp to be set")
	}
}

// TestOrderModification tests that orders can be modified after creation
func TestOrderModification(t *testing.T) {
	order := matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 50)

	// Modify fields
	order.Price = 105.0
	order.StopPrice = 95.0
	order.Side = matching.Sell
	order.OrderType = matching.StopLimitOrder

	if order.Price != 105.0 {
		t.Errorf("Expected modified Price 105.0, got %f", order.Price)
	}
	if order.StopPrice != 95.0 {
		t.Errorf("Expected StopPrice 95.0, got %f", order.StopPrice)
	}
	if order.Side != matching.Sell {
		t.Errorf("Expected modified Side Sell, got %d", order.Side)
	}
	if order.OrderType != matching.StopLimitOrder {
		t.Errorf("Expected modified OrderType StopLimitOrder, got %d", order.OrderType)
	}
}

// TestOrderConcurrentCreation tests creating multiple orders concurrently
func TestOrderConcurrentCreation(t *testing.T) {
	const numOrders = 1000
	done := make(chan bool, numOrders)

	for i := 0; i < numOrders; i++ {
		go func(id uint64) {
			order := matching.NewOrder(id, matching.LimitOrder, matching.Buy, 100.0, 10)
			if order.ID != id {
				t.Errorf("Expected ID %d, got %d", id, order.ID)
			}
			done <- true
		}(uint64(i))
	}

	// Wait for all goroutines to complete
	for i := 0; i < numOrders; i++ {
		<-done
	}
}

// TestOrderTimestampAccuracy tests that timestamps are set correctly
func TestOrderTimestampAccuracy(t *testing.T) {
	before := time.Now()
	time.Sleep(1 * time.Millisecond) // Small delay to ensure timestamp is different

	order1 := matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 10)

	time.Sleep(1 * time.Millisecond)
	order2 := matching.NewOrder(2, matching.LimitOrder, matching.Sell, 101.0, 10)

	time.Sleep(1 * time.Millisecond)
	after := time.Now()

	// Verify timestamps are in order
	if !order1.TimeStamp.After(before) {
		t.Error("Order1 timestamp should be after 'before' time")
	}
	if !order2.TimeStamp.After(order1.TimeStamp) {
		t.Error("Order2 timestamp should be after Order1 timestamp")
	}
	if !after.After(order2.TimeStamp) {
		t.Error("'after' time should be after Order2 timestamp")
	}
}

// TestOrderWithZeroID tests that orders can have zero ID
func TestOrderWithZeroID(t *testing.T) {
	order := matching.NewOrder(0, matching.LimitOrder, matching.Buy, 100.0, 10)

	if order.ID != 0 {
		t.Errorf("Expected ID 0, got %d", order.ID)
	}
	if !order.IsValid() {
		t.Error("Order with ID 0 should still be valid if other fields are correct")
	}
}

// TestOrderEdgeCasePrices tests edge case price values
func TestOrderEdgeCasePrices(t *testing.T) {
	tests := []struct {
		name  string
		price float64
		valid bool
	}{
		{"MaxFloat64", 1.7976931348623157e+308, true},
		{"VerySmallPositive", 1e-10, true},
		{"ExactlyOne", 1.0, true},
		{"LargeRoundNumber", 1000000.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := matching.NewOrder(1, matching.LimitOrder, matching.Buy, tt.price, 10)
			if order.IsValid() != tt.valid {
				t.Errorf("Expected IsValid() = %v for price %f", tt.valid, tt.price)
			}
		})
	}
}
