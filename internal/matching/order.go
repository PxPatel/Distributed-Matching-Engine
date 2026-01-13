package matching

import "github.com/PxPatel/trading-system/internal/types"

// Re-export types for backward compatibility
type (
	OrderType = types.OrderType
	SideType  = types.SideType
	Order     = types.Order
)

// Re-export constants
const (
	NoActionOrder   = types.NoActionOrder
	MarketOrder     = types.MarketOrder
	LimitOrder      = types.LimitOrder
	CancelOrder     = types.CancelOrder
	StopMarketOrder = types.StopMarketOrder
	StopLimitOrder  = types.StopLimitOrder

	NoActionSide = types.NoActionSide
	Buy          = types.Buy
	Sell         = types.Sell
)

// Re-export constructor
var NewOrder = types.NewOrder
