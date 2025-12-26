package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PxPatel/trading-system/internal/api/handlers"
	"github.com/PxPatel/trading-system/internal/api/routes"
	"github.com/PxPatel/trading-system/internal/matching"
	"github.com/stretchr/testify/require"
)

// TestServer wraps a test HTTP server with the matching engine
type TestServer struct {
	Server       *httptest.Server
	Engine       *matching.Engine
	TradeLogPath string
	t            testing.TB
}

// NewTestServer creates a new test server with a fresh engine
func NewTestServer(t testing.TB) *TestServer {
	// Create temporary trade log file
	tmpDir := t.TempDir()
	tradeLogPath := filepath.Join(tmpDir, "test_trades.log")

	// Create engine with test configuration
	engine := matching.NewEngineWithConfig(&matching.EngineConfig{
		TradeHistorySize: 100,
		TradeLogPath:     tradeLogPath,
	})

	// Create handler and server
	engineHolder := handlers.NewEngineHolder(engine)
	handler := routes.SetupRoutes(engineHolder)
	server := httptest.NewServer(handler)

	return &TestServer{
		Server:       server,
		Engine:       engine,
		TradeLogPath: tradeLogPath,
		t:            t,
	}
}

// Close cleans up the test server
func (ts *TestServer) Close() {
	ts.Server.Close()
	ts.Engine.Close()
	// Cleanup is automatic via t.TempDir()
}

// URL returns the base URL for the test server
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// Get makes a GET request to the test server
func (ts *TestServer) Get(path string) *http.Response {
	resp, err := http.Get(ts.URL() + path)
	require.NoError(ts.t, err, "GET request failed")
	return resp
}

// Post makes a POST request with JSON body
func (ts *TestServer) Post(path string, body interface{}) *http.Response {
	jsonBody, err := json.Marshal(body)
	require.NoError(ts.t, err, "Failed to marshal request body")

	resp, err := http.Post(ts.URL()+path, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(ts.t, err, "POST request failed")
	return resp
}

// Delete makes a DELETE request
func (ts *TestServer) Delete(path string) *http.Response {
	req, err := http.NewRequest("DELETE", ts.URL()+path, nil)
	require.NoError(ts.t, err, "Failed to create DELETE request")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(ts.t, err, "DELETE request failed")
	return resp
}

// DecodeJSON decodes JSON response into target
func DecodeJSON(t testing.TB, resp *http.Response, target interface{}) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	err = json.Unmarshal(body, target)
	require.NoError(t, err, "Failed to decode JSON response: %s", string(body))
}

// ReadTradeLog reads the trade log file and returns trades
func (ts *TestServer) ReadTradeLog() []matching.Trade {
	data, err := os.ReadFile(ts.TradeLogPath)
	if err != nil {
		return []matching.Trade{}
	}

	var trades []matching.Trade
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var trade matching.Trade
		if err := decoder.Decode(&trade); err == io.EOF {
			break
		} else if err != nil {
			ts.t.Fatalf("Failed to decode trade: %v", err)
		}
		trades = append(trades, trade)
	}
	return trades
}

// GetOrderBookDepth returns the current orderbook depth
func (ts *TestServer) GetOrderBookDepth() (bidLevels, askLevels int) {
	bidPrices := ts.Engine.GetOrderBook().GetAllBids()
	askPrices := ts.Engine.GetOrderBook().GetAllAsks()
	return len(bidPrices), len(askPrices)
}

// GetTrackedOrderCount returns the number of tracked orders
func (ts *TestServer) GetTrackedOrderCount() int {
	return len(ts.Engine.GetAllOrders())
}
