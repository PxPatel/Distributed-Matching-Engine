package matching

import "math"

/*
Data structure to hold best bid and best ask at the top
Sorted
Efficiently find and remove order
Heap would be great, but not efficient find
Red-Black tree would not necessary have smallest at the top, but not hard to find it in log(n) time (all the way to the left)

Red-Black tree approach
Order books have multiple price levels.
Price level only matters if there is atleast an order there
Each node of the tree can be a price level. Each of it can store an array of pointers to orders
New orders at a new price would create node and add to array
New orders at a existing price node would add on to the array
Can use a set instead of an array for hashing. Maybe a sorted set. Might be too much overhead

What are alternatives. Maybe combination of structures. min/max heap to keep the best price of one side. Store a pointer inside to the order array
Only the best bid and ask matter. So heap works well
Can also use hash for quick check on if the price level node exists AND for efficient find

Conclusion: Min/Max heap + hash + array of *Order in memory
*/

type OrderBook struct {
	bids map[float64][]*Order // bid side
	asks map[float64][]*Order // ask side
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		bids: make(map[float64][]*Order),
		asks: make(map[float64][]*Order),
	}
}

type PriceLevelOrders struct {
	Bids []*Order
	Asks []*Order
}

func (orderBook *OrderBook) GetOrdersByPrice(priceLevel float64) *PriceLevelOrders {
	return &PriceLevelOrders{
		Bids: orderBook.bids[priceLevel],
		Asks: orderBook.asks[priceLevel],
	}
}

// Search, given orderId
func (orderBook *OrderBook) SearchById(orderId uint64) *Order {
	// Check bids
	foundOrder, ok, _, _ := findOrderInBookSide(orderBook.bids, orderId)
	if ok {
		return foundOrder
	}

	// Check asks
	foundOrder, ok, _, _ = findOrderInBookSide(orderBook.asks, orderId)
	if ok {
		return foundOrder
	}
	return nil
}

func findOrderInBookSide(bookSide map[float64][]*Order, orderId uint64) (*Order, bool, float64, int) {
	for priceLevel, orderBlock := range bookSide {
		for i, order := range orderBlock {
			if order.ID == orderId {
				return order, true, priceLevel, i
			}
		}
	}
	return nil, false, 0.0, 0
}

func (orderBook *OrderBook) GetBestBid() (float64, []*Order) {
	bids := orderBook.bids

	if len(bids) == 0 {
		return 0.0, nil
	}

	max := 0.0
	for priceLevel := range bids {
		if priceLevel > max {
			max = priceLevel
		}
	}

	return max, bids[max]
}

func (orderBook *OrderBook) GetBestAsk() (float64, []*Order) {
	asks := orderBook.asks

	if len(asks) == 0 {
		return 0.0, nil
	}

	min := math.MaxFloat64
	for priceLevel := range asks {
		if priceLevel < min {
			min = priceLevel
		}
	}

	return min, asks[min]
}

func (orderBook *OrderBook) DeleteBidBlock(priceLevel float64) bool {
	return DeletePriceBlock(orderBook.bids, priceLevel)
}

func (orderBook *OrderBook) DeleteAskBlock(priceLevel float64) bool {
	return DeletePriceBlock(orderBook.asks, priceLevel)
}

func DeletePriceBlock(bookSide map[float64][]*Order, priceLevel float64) bool {
	_, ok := bookSide[priceLevel]
	if ok {
		delete(bookSide, priceLevel)
		return true
	}
	return false
}

func (orderBook *OrderBook) DeleteOrderById(orderId uint64) bool {
	return orderBook.DeleteBidOrder(orderId) || orderBook.DeleteAskOrder(orderId)
}

func (orderBook *OrderBook) DeleteBidOrder(orderId uint64) bool {
	_, ok, priceLevel, index := findOrderInBookSide(orderBook.bids, orderId)
	if ok {
		block := orderBook.bids[priceLevel]
		block = append(block[:index], block[index+1:]...)
		orderBook.bids[priceLevel] = block

		// Clean up empty price level
		if len(block) == 0 {
			delete(orderBook.bids, priceLevel)
		}
		return true
	}

	return false
}

func (orderBook *OrderBook) DeleteAskOrder(orderId uint64) bool {
	_, ok, priceLevel, index := findOrderInBookSide(orderBook.asks, orderId)
	if ok {
		block := orderBook.asks[priceLevel]
		block = append(block[:index], block[index+1:]...)
		orderBook.asks[priceLevel] = block

		// Clean up empty price level
		if len(block) == 0 {
			delete(orderBook.asks, priceLevel)
		}
		return true
	}

	return false
}

func (orderBook *OrderBook) AddBidOrder(newOrder *Order) bool {
	block := orderBook.bids[newOrder.Price]
	block = append(block, newOrder)
	orderBook.bids[newOrder.Price] = block
	return true
}

func (orderBook *OrderBook) AddAskOrder(newOrder *Order) bool {
	block := orderBook.asks[newOrder.Price]
	block = append(block, newOrder)
	orderBook.asks[newOrder.Price] = block
	return true
}
