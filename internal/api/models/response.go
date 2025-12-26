package models

import "time"

// BaseResponse is the base structure for all API responses
type BaseResponse struct {
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
	Error     *APIError `json:"error,omitempty"`
}

// TradeDTO represents a trade in API responses
type TradeDTO struct {
	BuyOrderID  uint64    `json:"buy_order_id"`
	SellOrderID uint64    `json:"sell_order_id"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

// SubmitOrderResponse represents the response for order submission
type SubmitOrderResponse struct {
	BaseResponse
	OrderID uint64      `json:"order_id,omitempty"`
	Trades  []TradeDTO  `json:"trades,omitempty"`
}

// BatchOrderResult represents a single order result in batch submission
type BatchOrderResult struct {
	Index   int        `json:"index"`
	Success bool       `json:"success"`
	OrderID uint64     `json:"order_id,omitempty"`
	Trades  []TradeDTO `json:"trades,omitempty"`
	Error   *APIError  `json:"error,omitempty"`
}

// BatchOrderSummary provides summary statistics for batch submission
type BatchOrderSummary struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
}

// BatchOrderResponse represents the response for batch order submission
type BatchOrderResponse struct {
	BaseResponse
	Results []BatchOrderResult `json:"results"`
	Summary BatchOrderSummary  `json:"summary"`
}

// CancelOrderResponse represents the response for order cancellation
type CancelOrderResponse struct {
	BaseResponse
	OrderID uint64 `json:"order_id,omitempty"`
}

// OrderDTO represents an order in API responses
type OrderDTO struct {
	OrderID           uint64    `json:"order_id"`
	UserID            string    `json:"user_id"`
	Symbol            string    `json:"symbol"`
	OrderType         string    `json:"order_type"`
	Side              string    `json:"side"`
	Price             float64   `json:"price"`
	Quantity          int       `json:"quantity"`
	FilledQuantity    int       `json:"filled_quantity,omitempty"`
	RemainingQuantity int       `json:"remaining_quantity,omitempty"`
	Status            string    `json:"status,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
}

// GetOrderResponse represents the response for getting a single order
type GetOrderResponse struct {
	BaseResponse
	Order *OrderDTO `json:"order,omitempty"`
}

// GetOrdersResponse represents the response for getting multiple orders
type GetOrdersResponse struct {
	BaseResponse
	Orders []OrderDTO `json:"orders"`
	Count  int        `json:"count"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price      float64 `json:"price"`
	Quantity   int     `json:"quantity"`
	OrderCount int     `json:"order_count"`
}

// OrderBookResponse represents the full order book
type OrderBookResponse struct {
	BaseResponse
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Spread    float64      `json:"spread,omitempty"`
	MidPrice  float64      `json:"mid_price,omitempty"`
}

// BestQuote represents the best bid or ask
type BestQuote struct {
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

// TopOfBookResponse represents the best bid and ask
type TopOfBookResponse struct {
	BaseResponse
	Symbol   string     `json:"symbol"`
	BestBid  *BestQuote `json:"best_bid,omitempty"`
	BestAsk  *BestQuote `json:"best_ask,omitempty"`
	Spread   float64    `json:"spread,omitempty"`
	MidPrice float64    `json:"mid_price,omitempty"`
}

// GetTradesResponse represents the response for getting trades
type GetTradesResponse struct {
	BaseResponse
	Trades []TradeDTO `json:"trades"`
	Count  int        `json:"count"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	Version       string    `json:"version"`
}
