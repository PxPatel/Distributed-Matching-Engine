package testutils

import (
	"github.com/PxPatel/trading-system/internal/api/models"
)

// OrderRequest builders for common test cases

// NewMarketBuyOrder creates a market buy order request
func NewMarketBuyOrder(userID string, quantity int) models.SubmitOrderRequest {
	return models.SubmitOrderRequest{
		UserID:    userID,
		OrderType: "market",
		Side:      "buy",
		Price:     0,
		Quantity:  quantity,
	}
}

// NewMarketSellOrder creates a market sell order request
func NewMarketSellOrder(userID string, quantity int) models.SubmitOrderRequest {
	return models.SubmitOrderRequest{
		UserID:    userID,
		OrderType: "market",
		Side:      "sell",
		Price:     0,
		Quantity:  quantity,
	}
}

// NewLimitBuyOrder creates a limit buy order request
func NewLimitBuyOrder(userID string, price float64, quantity int) models.SubmitOrderRequest {
	return models.SubmitOrderRequest{
		UserID:    userID,
		OrderType: "limit",
		Side:      "buy",
		Price:     price,
		Quantity:  quantity,
	}
}

// NewLimitSellOrder creates a limit sell order request
func NewLimitSellOrder(userID string, price float64, quantity int) models.SubmitOrderRequest {
	return models.SubmitOrderRequest{
		UserID:    userID,
		OrderType: "limit",
		Side:      "sell",
		Price:     price,
		Quantity:  quantity,
	}
}

// NewBatchRequest creates a batch order request
func NewBatchRequest(orders ...models.SubmitOrderRequest) models.BatchOrderRequest {
	return models.BatchOrderRequest{
		Orders: orders,
	}
}
