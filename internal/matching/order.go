package matching

import "time"

type OrderType int

const (
	NoActionOrder OrderType = iota
	MarketOrder
	LimitOrder
	CancelOrder
	StopMarketOrder
	StopLimitOrder
)

type SideType int

const (
	NoActionSide SideType = iota
	Buy
	Sell
)

type Order struct {
	ID        uint64
	UserID    string
	Symbol    string
	OrderType OrderType
	Side      SideType
	Price     float64
	StopPrice float64
	Size      int
	TimeStamp time.Time
}

func (o *Order) IsValid() bool {
	if o.OrderType == NoActionOrder || o.Side == NoActionSide {
		return false
	}
	if o.Size <= 0 {
		return false
	}
	if o.OrderType == LimitOrder && o.Price <= 0 {
		return false
	}
	return true
}

func NewOrder(id uint64, userId string, orderType OrderType, side SideType, price float64, quantity int) *Order {
	return &Order{
		ID:        id,
		UserID:    userId,
		Symbol:    "COOTX",
		OrderType: orderType,
		Side:      side,
		Price:     price,
		Size:      quantity,
		TimeStamp: time.Now(),
	}
}

func (o *Order) SetSize(size int) {
	if size >= 0 {
		o.Size = size
	}
}
