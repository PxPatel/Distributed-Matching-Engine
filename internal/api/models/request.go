package models

import (
	"strings"
)

// SubmitOrderRequest represents a single order submission
type SubmitOrderRequest struct {
	UserID    string  `json:"user_id"`
	OrderType string  `json:"order_type"` // "market" | "limit"
	Side      string  `json:"side"`       // "buy" | "sell"
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

// Validate validates the order request
func (r *SubmitOrderRequest) Validate() *HTTPError {
	// Validate user_id
	if strings.TrimSpace(r.UserID) == "" {
		return ErrBadRequest("user_id cannot be empty", map[string]interface{}{"field": "user_id"})
	}

	// Validate order_type
	orderType := strings.ToLower(strings.TrimSpace(r.OrderType))
	if orderType != "market" && orderType != "limit" {
		return ErrInvalidOrderTypeError(r.OrderType)
	}

	// Validate side
	side := strings.ToLower(strings.TrimSpace(r.Side))
	if side != "buy" && side != "sell" {
		return ErrInvalidSideError(r.Side)
	}

	// Validate quantity
	if r.Quantity <= 0 {
		return ErrInvalidQuantityError(r.Quantity)
	}

	// Validate price for limit orders
	if orderType == "limit" {
		if r.Price <= 0 {
			return ErrInvalidPriceError(r.Price)
		}
	}

	return nil
}

// BatchOrderRequest represents a batch order submission
type BatchOrderRequest struct {
	Orders []SubmitOrderRequest `json:"orders"`
}

// Validate validates the batch request
func (r *BatchOrderRequest) Validate() *HTTPError {
	if len(r.Orders) == 0 {
		return ErrBadRequest("orders array cannot be empty", map[string]interface{}{"field": "orders"})
	}

	if len(r.Orders) > 1000 {
		return ErrBadRequest("batch size cannot exceed 1000 orders",
			map[string]interface{}{"field": "orders", "max_size": 1000, "provided_size": len(r.Orders)})
	}

	return nil
}
