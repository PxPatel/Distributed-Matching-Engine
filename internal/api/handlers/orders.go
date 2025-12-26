package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PxPatel/trading-system/internal/api/logger"
	"github.com/PxPatel/trading-system/internal/api/models"
	"github.com/PxPatel/trading-system/internal/matching"
)

// EngineHolder wraps the matching engine for dependency injection
type EngineHolder struct {
	Engine *matching.Engine
}

// NewEngineHolder creates a new engine holder
func NewEngineHolder(engine *matching.Engine) *EngineHolder {
	return &EngineHolder{Engine: engine}
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, httpErr *models.HTTPError) {
	logger.Warn("Request failed", map[string]interface{}{
		"error_code": httpErr.Error.Code,
		"status":     httpErr.StatusCode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpErr.StatusCode)

	response := models.BaseResponse{
		Success:   false,
		Timestamp: time.Now().UTC(),
		Message:   httpErr.Error.Message,
		Error:     &httpErr.Error,
	}

	json.NewEncoder(w).Encode(response)
}

// convertOrderType converts string to OrderType
func convertOrderType(orderType string) matching.OrderType {
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "market":
		return matching.MarketOrder
	case "limit":
		return matching.LimitOrder
	default:
		return matching.NoActionOrder
	}
}

// convertSide converts string to SideType
func convertSide(side string) matching.SideType {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy":
		return matching.Buy
	case "sell":
		return matching.Sell
	default:
		return matching.NoActionSide
	}
}

// convertTradesToDTO converts matching trades to DTO trades
func convertTradesToDTO(trades []*matching.Trade) []models.TradeDTO {
	dtos := make([]models.TradeDTO, len(trades))
	for i, trade := range trades {
		dtos[i] = models.TradeDTO{
			BuyOrderID:  trade.BuyOrderID,
			SellOrderID: trade.SellOrderID,
			Price:       trade.Price,
			Quantity:    trade.Size,
			Timestamp:   trade.Timestamp,
		}
	}
	return dtos
}

// SubmitOrderHandler handles single order submission
func (eh *EngineHolder) SubmitOrderHandler(w http.ResponseWriter, r *http.Request) {
	var req models.SubmitOrderRequest

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, models.ErrBadRequest("Invalid JSON format", map[string]interface{}{"error": err.Error()}))
		return
	}

	// Validate request
	if httpErr := req.Validate(); httpErr != nil {
		writeErrorResponse(w, httpErr)
		return
	}

	// Generate order ID
	orderID := eh.Engine.GenerateOrderID()

	// Convert to matching order
	order := matching.NewOrder(
		orderID,
		req.UserID,
		convertOrderType(req.OrderType),
		convertSide(req.Side),
		req.Price,
		req.Quantity,
	)

	// Submit order to engine
	trades := eh.Engine.PlaceOrder(order)

	logger.Info("Order submitted successfully", map[string]interface{}{
		"order_id": orderID,
		"user_id":  req.UserID,
		"type":     req.OrderType,
		"side":     req.Side,
		"trades":   len(trades),
	})

	// Return response
	response := models.SubmitOrderResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
			Message:   "Order submitted successfully",
		},
		OrderID: orderID,
		Trades:  convertTradesToDTO(trades),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// BatchOrderHandler handles batch order submission
func (eh *EngineHolder) BatchOrderHandler(w http.ResponseWriter, r *http.Request) {
	var req models.BatchOrderRequest

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, models.ErrBadRequest("Invalid JSON format", map[string]interface{}{"error": err.Error()}))
		return
	}

	// Validate batch request
	if httpErr := req.Validate(); httpErr != nil {
		writeErrorResponse(w, httpErr)
		return
	}

	// Process each order
	results := make([]models.BatchOrderResult, len(req.Orders))
	successful := 0
	failed := 0

	for i, orderReq := range req.Orders {
		result := models.BatchOrderResult{
			Index: i,
		}

		// Validate individual order
		if httpErr := orderReq.Validate(); httpErr != nil {
			result.Success = false
			result.Error = &httpErr.Error
			failed++
		} else {
			// Generate order ID
			orderID := eh.Engine.GenerateOrderID()

			// Convert to matching order
			order := matching.NewOrder(
				orderID,
				orderReq.UserID,
				convertOrderType(orderReq.OrderType),
				convertSide(orderReq.Side),
				orderReq.Price,
				orderReq.Quantity,
			)

			// Submit order to engine
			trades := eh.Engine.PlaceOrder(order)

			result.Success = true
			result.OrderID = orderID
			result.Trades = convertTradesToDTO(trades)
			successful++
		}

		results[i] = result
	}

	logger.Info("Batch order processed", map[string]interface{}{
		"total":      len(req.Orders),
		"successful": successful,
		"failed":     failed,
	})

	// Return response
	response := models.BatchOrderResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
		},
		Results: results,
		Summary: models.BatchOrderSummary{
			Total:      len(req.Orders),
			Successful: successful,
			Failed:     failed,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CancelOrderHandler handles order cancellation
func (eh *EngineHolder) CancelOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Extract order ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		writeErrorResponse(w, models.ErrBadRequest("Invalid order ID", nil))
		return
	}

	orderIDStr := pathParts[len(pathParts)-1]
	orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
	if err != nil {
		writeErrorResponse(w, models.ErrBadRequest("Invalid order ID format", map[string]interface{}{"provided_value": orderIDStr}))
		return
	}

	// Cancel order
	cancelled := eh.Engine.CancelOrder(orderID)

	if !cancelled {
		writeErrorResponse(w, models.ErrOrderNotFoundError(orderID))
		return
	}

	logger.Info("Order cancelled", map[string]interface{}{
		"order_id": orderID,
	})

	// Return response
	response := models.CancelOrderResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
			Message:   "Order cancelled successfully",
		},
		OrderID: orderID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetOrderHandler handles retrieving a single order
func (eh *EngineHolder) GetOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Extract order ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		writeErrorResponse(w, models.ErrBadRequest("Invalid order ID", nil))
		return
	}

	orderIDStr := pathParts[len(pathParts)-1]
	orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
	if err != nil {
		writeErrorResponse(w, models.ErrBadRequest("Invalid order ID format", map[string]interface{}{"provided_value": orderIDStr}))
		return
	}

	// Get order from engine
	order := eh.Engine.GetOrder(orderID)

	if order == nil {
		writeErrorResponse(w, models.ErrOrderNotFoundError(orderID))
		return
	}

	// Convert to DTO
	orderDTO := convertOrderToDTO(order)

	// Return response
	response := models.GetOrderResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
		},
		Order: orderDTO,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetAllOrdersHandler handles retrieving all open orders
func (eh *EngineHolder) GetAllOrdersHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	userID := r.URL.Query().Get("user_id")
	sideStr := r.URL.Query().Get("side")
	limitStr := r.URL.Query().Get("limit")

	// Default limit
	limit := 100
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	// Get orders from engine
	var orders []*matching.Order

	if userID != "" {
		orders = eh.Engine.GetOrdersByUser(userID)
	} else if sideStr != "" {
		side := convertSide(sideStr)
		if side != matching.NoActionSide {
			orders = eh.Engine.GetOrdersBySide(side)
		} else {
			orders = eh.Engine.GetAllOrders()
		}
	} else {
		orders = eh.Engine.GetAllOrders()
	}

	// Apply limit
	if len(orders) > limit {
		orders = orders[:limit]
	}

	// Convert to DTOs
	orderDTOs := make([]models.OrderDTO, len(orders))
	for i, order := range orders {
		orderDTOs[i] = *convertOrderToDTO(order)
	}

	logger.Info("Retrieved orders", map[string]interface{}{
		"count": len(orderDTOs),
	})

	// Return response
	response := models.GetOrdersResponse{
		BaseResponse: models.BaseResponse{
			Success:   true,
			Timestamp: time.Now().UTC(),
		},
		Orders: orderDTOs,
		Count:  len(orderDTOs),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// convertOrderToDTO converts a matching order to DTO
func convertOrderToDTO(order *matching.Order) *models.OrderDTO {
	var orderType, side string

	switch order.OrderType {
	case matching.MarketOrder:
		orderType = "market"
	case matching.LimitOrder:
		orderType = "limit"
	default:
		orderType = "unknown"
	}

	switch order.Side {
	case matching.Buy:
		side = "buy"
	case matching.Sell:
		side = "sell"
	default:
		side = "unknown"
	}

	return &models.OrderDTO{
		OrderID:   order.ID,
		UserID:    order.UserID,
		Symbol:    order.Symbol,
		OrderType: orderType,
		Side:      side,
		Price:     order.Price,
		Quantity:  order.Size,
		Status:    "open",
		Timestamp: order.TimeStamp,
	}
}
