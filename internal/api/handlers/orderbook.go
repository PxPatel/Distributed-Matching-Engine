package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/PxPatel/trading-system/internal/api/logger"
	"github.com/PxPatel/trading-system/internal/api/models"
	"github.com/PxPatel/trading-system/internal/matching"
)

// aggregatePriceLevels aggregates orders by tick size
func aggregatePriceLevels(prices []float64, getOrders func(float64) []*matching.Order, tickSize float64, maxDepth int) []models.PriceLevel {
	if len(prices) == 0 {
		return []models.PriceLevel{}
	}

	// If no tick size aggregation, return raw levels
	if tickSize <= 0 {
		levels := make([]models.PriceLevel, 0)
		for i, price := range prices {
			if i >= maxDepth {
				break
			}

			orders := getOrders(price)
			totalQty := 0
			for _, order := range orders {
				totalQty += order.Size
			}

			levels = append(levels, models.PriceLevel{
				Price:      price,
				Quantity:   totalQty,
				OrderCount: len(orders),
			})
		}
		return levels
	}

	// Aggregate by tick size
	aggregated := make(map[float64]*models.PriceLevel)
	for _, price := range prices {
		// Round to nearest tick
		tickPrice := math.Round(price/tickSize) * tickSize

		if aggregated[tickPrice] == nil {
			aggregated[tickPrice] = &models.PriceLevel{
				Price:      tickPrice,
				Quantity:   0,
				OrderCount: 0,
			}
		}

		orders := getOrders(price)
		for _, order := range orders {
			aggregated[tickPrice].Quantity += order.Size
			aggregated[tickPrice].OrderCount++
		}
	}

	// Convert to sorted slice
	sortedPrices := make([]float64, 0, len(aggregated))
	for price := range aggregated {
		sortedPrices = append(sortedPrices, price)
	}

	// Sort based on side (bids descending, asks ascending)
	for i := 0; i < len(sortedPrices); i++ {
		for j := i + 1; j < len(sortedPrices); j++ {
			if sortedPrices[i] < sortedPrices[j] {
				sortedPrices[i], sortedPrices[j] = sortedPrices[j], sortedPrices[i]
			}
		}
	}

	// Build result
	levels := make([]models.PriceLevel, 0)
	for i, price := range sortedPrices {
		if i >= maxDepth {
			break
		}
		levels = append(levels, *aggregated[price])
	}

	return levels
}

// GetOrderBookHandler handles full order book snapshot requests
func (eh *EngineHolder) GetOrderBookHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	depthStr := r.URL.Query().Get("depth")
	aggregateStr := r.URL.Query().Get("aggregate")

	// Default depth: 10, max: 10
	depth := 10
	if depthStr != "" {
		parsedDepth, err := strconv.Atoi(depthStr)
		if err == nil && parsedDepth > 0 {
			depth = parsedDepth
			if depth > 10 {
				depth = 10
			}
		}
	}

	// Parse tick size for aggregation
	tickSize := 0.0
	if aggregateStr != "" {
		parsedTick, err := strconv.ParseFloat(aggregateStr, 64)
		if err == nil && parsedTick > 0 {
			tickSize = parsedTick
		}
	}

	// Get order book from engine
	bidPrices := eh.Engine.GetOrderBook().GetAllBids()
	askPrices := eh.Engine.GetOrderBook().GetAllAsks()

	// Build bid levels (descending)
	bids := aggregatePriceLevels(bidPrices, eh.Engine.GetOrderBook().GetBidsAtPrice, tickSize, depth)

	// Build ask levels (ascending) - need to reverse sort
	asks := aggregatePriceLevels(askPrices, eh.Engine.GetOrderBook().GetAsksAtPrice, tickSize, depth)
	// Re-sort asks in ascending order
	for i := 0; i < len(asks); i++ {
		for j := i + 1; j < len(asks); j++ {
			if asks[i].Price > asks[j].Price {
				asks[i], asks[j] = asks[j], asks[i]
			}
		}
	}

	// Calculate spread and mid price
	var spread, midPrice float64
	if len(bids) > 0 && len(asks) > 0 {
		spread = asks[0].Price - bids[0].Price
		midPrice = (bids[0].Price + asks[0].Price) / 2.0
	}

	logger.Info("Order book snapshot retrieved", map[string]interface{}{
		"bid_levels": len(bids),
		"ask_levels": len(asks),
		"tick_size":  tickSize,
	})

	// Return response
	response := models.OrderBookResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
		},
		Symbol:   "COOTX",
		Bids:     bids,
		Asks:     asks,
		Spread:   spread,
		MidPrice: midPrice,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetTopOfBookHandler handles best bid/ask requests
func (eh *EngineHolder) GetTopOfBookHandler(w http.ResponseWriter, r *http.Request) {
	// Get best bid and ask
	bestBidPrice, bestBidOrders := eh.Engine.GetOrderBook().GetBestBid()
	bestAskPrice, bestAskOrders := eh.Engine.GetOrderBook().GetBestAsk()

	var bestBid, bestAsk *models.BestQuote
	var spread, midPrice float64

	// Build best bid
	if len(bestBidOrders) > 0 {
		totalQty := 0
		for _, order := range bestBidOrders {
			totalQty += order.Size
		}
		bestBid = &models.BestQuote{
			Price:    bestBidPrice,
			Quantity: totalQty,
		}
	}

	// Build best ask
	if len(bestAskOrders) > 0 {
		totalQty := 0
		for _, order := range bestAskOrders {
			totalQty += order.Size
		}
		bestAsk = &models.BestQuote{
			Price:    bestAskPrice,
			Quantity: totalQty,
		}
	}

	// Calculate spread and mid price
	if bestBid != nil && bestAsk != nil {
		spread = bestAsk.Price - bestBid.Price
		midPrice = (bestBid.Price + bestAsk.Price) / 2.0
	}

	logger.Info("Top of book retrieved", map[string]interface{}{
		"best_bid": bestBidPrice,
		"best_ask": bestAskPrice,
	})

	// Return response
	response := models.TopOfBookResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
		},
		Symbol:   "COOTX",
		BestBid:  bestBid,
		BestAsk:  bestAsk,
		Spread:   spread,
		MidPrice: midPrice,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetOrderBook is a helper to access the order book
func (eh *EngineHolder) GetOrderBook() *matching.OrderBook {
	return eh.Engine.GetOrderBook()
}
