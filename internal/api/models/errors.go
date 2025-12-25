package models

import "net/http"

// ErrorCode represents standard error codes
type ErrorCode string

const (
	ErrInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrInvalidOrderType ErrorCode = "INVALID_ORDER_TYPE"
	ErrInvalidSide      ErrorCode = "INVALID_SIDE"
	ErrInvalidPrice     ErrorCode = "INVALID_PRICE"
	ErrInvalidQuantity  ErrorCode = "INVALID_QUANTITY"
	ErrMissingPrice     ErrorCode = "MISSING_PRICE"
	ErrOrderNotFound    ErrorCode = "ORDER_NOT_FOUND"
	ErrInternalError    ErrorCode = "INTERNAL_ERROR"
)

// APIError represents a structured error response
type APIError struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HTTPError wraps an APIError with an HTTP status code
type HTTPError struct {
	StatusCode int
	Error      APIError
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, code ErrorCode, message string, details map[string]interface{}) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Common error constructors

func ErrBadRequest(message string, details map[string]interface{}) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, ErrInvalidRequest, message, details)
}

func ErrInvalidOrderTypeError(providedType string) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, ErrInvalidOrderType,
		"Invalid order type, must be 'market' or 'limit'",
		map[string]interface{}{"provided_value": providedType})
}

func ErrInvalidSideError(providedSide string) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, ErrInvalidSide,
		"Invalid side, must be 'buy' or 'sell'",
		map[string]interface{}{"provided_value": providedSide})
}

func ErrInvalidPriceError(price float64) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, ErrInvalidPrice,
		"Price must be greater than 0 for limit orders",
		map[string]interface{}{"field": "price", "provided_value": price})
}

func ErrInvalidQuantityError(quantity int) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, ErrInvalidQuantity,
		"Quantity must be positive",
		map[string]interface{}{"field": "quantity", "provided_value": quantity})
}

func ErrMissingPriceError() *HTTPError {
	return NewHTTPError(http.StatusUnprocessableEntity, ErrMissingPrice,
		"Price is required for limit orders", nil)
}

func ErrOrderNotFoundError(orderID uint64) *HTTPError {
	return NewHTTPError(http.StatusNotFound, ErrOrderNotFound,
		"Order not found",
		map[string]interface{}{"order_id": orderID})
}

func ErrInternal(message string) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, ErrInternalError, message, nil)
}
