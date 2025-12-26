package performance

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PxPatel/trading-system/internal/api/models"
	"github.com/PxPatel/trading-system/internal/api/tests/testutils"
	"github.com/stretchr/testify/require"
)

// BenchmarkOrderSubmissionThroughput measures orders per second
func BenchmarkOrderSubmissionThroughput(b *testing.B) {
	ts := testutils.NewTestServer(b)
	defer ts.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		order := testutils.NewLimitBuyOrder("user", 100.0+float64(i%100)*0.01, 10)
		resp := ts.Post("/api/v1/orders", order)
		require.Equal(b, 200, resp.StatusCode)
		resp.Body.Close()
	}

	ordersPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
}

// BenchmarkMarketOrderExecution measures market order matching speed
func BenchmarkMarketOrderExecution(b *testing.B) {
	ts := testutils.NewTestServer(b)
	defer ts.Close()

	// Pre-populate orderbook with liquidity
	for i := 0; i < 100; i++ {
		price := 100.0 + float64(i)*0.01
		ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", price, 10))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp := ts.Post("/api/v1/orders", testutils.NewMarketBuyOrder("bob", 5))
		require.Equal(b, 200, resp.StatusCode)
		resp.Body.Close()
	}

	executionsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(executionsPerSec, "executions/sec")
}

// BenchmarkOrderBookSnapshot measures orderbook retrieval speed
func BenchmarkOrderBookSnapshot(b *testing.B) {
	ts := testutils.NewTestServer(b)
	defer ts.Close()

	// Populate orderbook with 50 levels each side
	for i := 0; i < 50; i++ {
		bidPrice := 99.0 - float64(i)*0.01
		askPrice := 101.0 + float64(i)*0.01
		ts.Post("/api/v1/orders", testutils.NewLimitBuyOrder("alice", bidPrice, 10))
		ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("bob", askPrice, 10))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp := ts.Get("/api/v1/orderbook?depth=10")
		require.Equal(b, 200, resp.StatusCode)
		resp.Body.Close()
	}

	snapshotsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(snapshotsPerSec, "snapshots/sec")
}

// BenchmarkBatchOrderSubmission measures batch order throughput
func BenchmarkBatchOrderSubmission(b *testing.B) {
	ts := testutils.NewTestServer(b)
	defer ts.Close()

	// Create batch of 10 orders
	orders := make([]models.SubmitOrderRequest, 10)
	for i := 0; i < 10; i++ {
		orders[i] = testutils.NewLimitBuyOrder(fmt.Sprintf("user%d", i), 100.0, 10)
	}
	batch := testutils.NewBatchRequest(orders...)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp := ts.Post("/api/v1/orders/batch", batch)
		require.Equal(b, 200, resp.StatusCode)
		resp.Body.Close()
	}

	ordersPerSec := float64(b.N*10) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
}

// BenchmarkConcurrentOrderSubmission measures concurrent request handling
func BenchmarkConcurrentOrderSubmission(b *testing.B) {
	ts := testutils.NewTestServer(b)
	defer ts.Close()

	concurrency := 10
	b.SetParallelism(concurrency)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			order := testutils.NewLimitBuyOrder("user", 100.0+float64(i%100)*0.01, 10)
			resp := ts.Post("/api/v1/orders", order)
			require.Equal(b, 200, resp.StatusCode)
			resp.Body.Close()
			i++
		}
	})

	ordersPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
}

// TestHighFrequencyTradingSimulation simulates HFT scenario
func TestHighFrequencyTradingSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HFT simulation in short mode")
	}

	ts := testutils.NewTestServer(t)
	defer ts.Close()

	duration := 5 * time.Second
	var orderCount atomic.Uint64
	var tradeCount atomic.Uint64

	// Liquidity provider: continuously adds limit orders
	liquidityProvider := func() {
		for start := time.Now(); time.Since(start) < duration; {
			price := 100.0 + float64(time.Now().UnixNano()%100)*0.01
			ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("lp", price, 10))
			orderCount.Add(1)
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Market maker: places both sides
	marketMaker := func() {
		for start := time.Now(); time.Since(start) < duration; {
			spread := 0.05
			mid := 100.0
			ts.Post("/api/v1/orders", testutils.NewLimitBuyOrder("mm", mid-spread, 5))
			ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("mm", mid+spread, 5))
			orderCount.Add(2)
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Aggressive trader: sends market orders
	aggressiveTrader := func() {
		for start := time.Now(); time.Since(start) < duration; {
			resp := ts.Post("/api/v1/orders", testutils.NewMarketBuyOrder("trader", 5))
			if resp.StatusCode == 200 {
				var result models.SubmitOrderResponse
				testutils.DecodeJSON(t, resp, &result)
				tradeCount.Add(uint64(len(result.Trades)))
			}
			resp.Body.Close()
			orderCount.Add(1)
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Run simulation with multiple participants
	var wg sync.WaitGroup
	wg.Add(5)

	go func() { defer wg.Done(); liquidityProvider() }()
	go func() { defer wg.Done(); marketMaker() }()
	go func() { defer wg.Done(); aggressiveTrader() }()
	go func() { defer wg.Done(); aggressiveTrader() }()
	go func() { defer wg.Done(); aggressiveTrader() }()

	wg.Wait()

	// Report metrics
	totalOrders := orderCount.Load()
	totalTrades := tradeCount.Load()
	ordersPerSec := float64(totalOrders) / duration.Seconds()
	tradesPerSec := float64(totalTrades) / duration.Seconds()

	t.Logf("HFT Simulation Results (%v):", duration)
	t.Logf("  Total Orders: %d", totalOrders)
	t.Logf("  Total Trades: %d", totalTrades)
	t.Logf("  Orders/sec: %.2f", ordersPerSec)
	t.Logf("  Trades/sec: %.2f", tradesPerSec)
	t.Logf("  Match Rate: %.2f%%", float64(totalTrades)/float64(totalOrders)*100)

	require.Greater(t, totalOrders, uint64(0), "Should process orders")
}

// TestLoadStressTest tests system under heavy load
func TestLoadStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Test parameters
	numWorkers := 20
	ordersPerWorker := 100

	var wg sync.WaitGroup
	var successCount atomic.Uint64
	var errorCount atomic.Uint64

	start := time.Now()

	// Spawn workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < ordersPerWorker; i++ {
				order := testutils.NewLimitBuyOrder(
					fmt.Sprintf("worker%d", workerID),
					100.0+float64(i%50)*0.01,
					5,
				)

				resp := ts.Post("/api/v1/orders", order)
				if resp.StatusCode == 200 {
					successCount.Add(1)
				} else {
					errorCount.Add(1)
				}
				resp.Body.Close()
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalOrders := numWorkers * ordersPerWorker
	throughput := float64(totalOrders) / elapsed.Seconds()
	errorRate := float64(errorCount.Load()) / float64(totalOrders) * 100

	t.Logf("Load Test Results:")
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Orders per worker: %d", ordersPerWorker)
	t.Logf("  Total orders: %d", totalOrders)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f orders/sec", throughput)
	t.Logf("  Success: %d", successCount.Load())
	t.Logf("  Errors: %d", errorCount.Load())
	t.Logf("  Error rate: %.2f%%", errorRate)

	require.Equal(t, uint64(totalOrders), successCount.Load(), "All orders should succeed")
	require.Zero(t, errorCount.Load(), "No errors expected")
}

// TestLatencyMeasurement measures end-to-end latency
func TestLatencyMeasurement(t *testing.T) {
	ts := testutils.NewTestServer(t)
	defer ts.Close()

	// Pre-populate with liquidity
	for i := 0; i < 50; i++ {
		ts.Post("/api/v1/orders", testutils.NewLimitSellOrder("alice", 100.0+float64(i)*0.01, 10))
	}

	numRequests := 1000
	latencies := make([]time.Duration, numRequests)

	// Measure latencies
	for i := 0; i < numRequests; i++ {
		start := time.Now()
		resp := ts.Post("/api/v1/orders", testutils.NewMarketBuyOrder("bob", 5))
		latencies[i] = time.Since(start)
		require.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}

	// Calculate statistics
	var total time.Duration
	min := latencies[0]
	max := latencies[0]

	for _, lat := range latencies {
		total += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	avg := total / time.Duration(numRequests)

	// Calculate p95 and p99
	sorted := make([]time.Duration, numRequests)
	copy(sorted, latencies)
	for i := 0; i < numRequests; i++ {
		for j := i + 1; j < numRequests; j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p95 := sorted[int(float64(numRequests)*0.95)]
	p99 := sorted[int(float64(numRequests)*0.99)]

	t.Logf("Latency Statistics (%d requests):", numRequests)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Avg: %v", avg)
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)

	// Assert reasonable latency (adjust based on your requirements)
	require.Less(t, avg, 50*time.Millisecond, "Average latency should be < 50ms")
	require.Less(t, p99, 200*time.Millisecond, "P99 latency should be < 200ms")
}
