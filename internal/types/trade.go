package types

import "time"

// Trade represents a matched trade between a buy and sell order
type Trade struct {
	TradeID     uint64    `json:"trade_id,omitempty"`
	BuyOrderID  uint64    `json:"buy_order_id"`
	SellOrderID uint64    `json:"sell_order_id"`
	Price       float64   `json:"price"`
	Size        int       `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}
