package matching

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/PxPatel/trading-system/internal/matching"
)

// Benchmark KPIs and Metrics:
// - Orders/second throughput
// - Average latency per operation
// - Memory allocations
// - Scalability with book depth

// BenchmarkOrderCreation benchmarks order creation speed
func BenchmarkOrderCreation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 100.0, 10)
	}

	// KPI: Orders created per second
	ordersPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
}

// BenchmarkOrderValidation benchmarks order validation speed
func BenchmarkOrderValidation(b *testing.B) {
	order := matching.NewOrder(1, matching.LimitOrder, matching.Buy, 100.0, 10)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		order.IsValid()
	}

	validationsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(validationsPerSec, "validations/sec")
}

// BenchmarkAddBidOrder benchmarks adding bid orders to orderbook
func BenchmarkAddBidOrder(b *testing.B) {
	ob := matching.NewOrderBook()
	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orders[i] = matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 100.0+float64(i%100)*0.01, 10)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ob.AddBidOrder(orders[i])
	}

	addsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(addsPerSec, "adds/sec")
}

// BenchmarkAddAskOrder benchmarks adding ask orders to orderbook
func BenchmarkAddAskOrder(b *testing.B) {
	ob := matching.NewOrderBook()
	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orders[i] = matching.NewOrder(uint64(i), matching.LimitOrder, matching.Sell, 101.0+float64(i%100)*0.01, 10)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ob.AddAskOrder(orders[i])
	}

	addsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(addsPerSec, "adds/sec")
}

// BenchmarkGetBestBid benchmarks retrieving best bid
func BenchmarkGetBestBid(b *testing.B) {
	ob := matching.NewOrderBook()

	// Pre-populate with orders at different prices
	for i := 0; i < 100; i++ {
		price := 100.0 - float64(i)*0.01
		ob.AddBidOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, price, 10))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ob.GetBestBid()
	}

	lookupsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(lookupsPerSec, "lookups/sec")
}

// BenchmarkGetBestAsk benchmarks retrieving best ask
func BenchmarkGetBestAsk(b *testing.B) {
	ob := matching.NewOrderBook()

	// Pre-populate with orders at different prices
	for i := 0; i < 100; i++ {
		price := 101.0 + float64(i)*0.01
		ob.AddAskOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Sell, price, 10))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ob.GetBestAsk()
	}

	lookupsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(lookupsPerSec, "lookups/sec")
}

// BenchmarkSearchById benchmarks searching for orders by ID
func BenchmarkSearchById(b *testing.B) {
	ob := matching.NewOrderBook()

	// Pre-populate orderbook
	for i := 0; i < 1000; i++ {
		price := 100.0 + float64(i%100)*0.01
		ob.AddBidOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, price, 10))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ob.SearchById(uint64(i % 1000))
	}

	searchesPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(searchesPerSec, "searches/sec")
}

// BenchmarkDeleteBidOrder benchmarks deleting bid orders
func BenchmarkDeleteBidOrder(b *testing.B) {
	// Pre-create orders
	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orders[i] = matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 100.0, 10)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ob := matching.NewOrderBook()
		ob.AddBidOrder(orders[i])
		ob.DeleteBidOrder(uint64(i))
	}

	deletesPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(deletesPerSec, "deletes/sec")
}

// BenchmarkMarketOrderExecution benchmarks market order execution
func BenchmarkMarketOrderExecution(b *testing.B) {
	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orders[i] = matching.NewOrder(uint64(i+10000), matching.MarketOrder, matching.Buy, 0.0, 10)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := matching.NewEngine()
		// Add liquidity
		for j := 0; j < 10; j++ {
			engine.PlaceOrder(matching.NewOrder(uint64(j), matching.LimitOrder, matching.Sell, 101.0+float64(j)*0.01, 10))
		}
		// Execute market order
		engine.PlaceOrder(orders[i])
	}

	executionsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(executionsPerSec, "executions/sec")
}

// BenchmarkLimitOrderExecution benchmarks limit order execution
func BenchmarkLimitOrderExecution(b *testing.B) {
	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orders[i] = matching.NewOrder(uint64(i+10000), matching.LimitOrder, matching.Buy, 101.0, 10)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := matching.NewEngine()
		// Add liquidity
		engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 10))
		// Execute limit order
		engine.PlaceOrder(orders[i])
	}

	executionsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(executionsPerSec, "executions/sec")
}

// BenchmarkCancelOrder benchmarks order cancellation
func BenchmarkCancelOrder(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := matching.NewEngine()
		order := matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 100.0, 10)
		engine.PlaceOrder(order)
		engine.CancelOrder(uint64(i))
	}

	cancelsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(cancelsPerSec, "cancels/sec")
}

// BenchmarkOrderBookDepth_10 benchmarks with 10 price levels
func BenchmarkOrderBookDepth_10(b *testing.B) {
	benchmarkOrderBookDepth(b, 10)
}

// BenchmarkOrderBookDepth_100 benchmarks with 100 price levels
func BenchmarkOrderBookDepth_100(b *testing.B) {
	benchmarkOrderBookDepth(b, 100)
}

// BenchmarkOrderBookDepth_1000 benchmarks with 1000 price levels
func BenchmarkOrderBookDepth_1000(b *testing.B) {
	benchmarkOrderBookDepth(b, 1000)
}

// BenchmarkOrderBookDepth_10000 benchmarks with 10000 price levels
func BenchmarkOrderBookDepth_10000(b *testing.B) {
	benchmarkOrderBookDepth(b, 10000)
}

// benchmarkOrderBookDepth is a helper for depth benchmarks
func benchmarkOrderBookDepth(b *testing.B, depth int) {
	engine := matching.NewEngine()

	// Pre-populate orderbook with depth price levels
	for i := 0; i < depth; i++ {
		bidPrice := 100.0 - float64(i)*0.01
		askPrice := 101.0 + float64(i)*0.01
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, bidPrice, 10))
		engine.PlaceOrder(matching.NewOrder(uint64(i+depth), matching.LimitOrder, matching.Sell, askPrice, 10))
	}

	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		// Alternate between market buys and sells
		if i%2 == 0 {
			orders[i] = matching.NewOrder(uint64(i+depth*2), matching.MarketOrder, matching.Buy, 0.0, 5)
		} else {
			orders[i] = matching.NewOrder(uint64(i+depth*2), matching.MarketOrder, matching.Sell, 0.0, 5)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.PlaceOrder(orders[i])
	}

	ordersPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
	avgLatency := b.Elapsed().Nanoseconds() / int64(b.N)
	b.ReportMetric(float64(avgLatency)/1000.0, "µs/op")
}

// BenchmarkHighFrequencyTrading simulates HFT scenario
func BenchmarkHighFrequencyTrading(b *testing.B) {
	engine := matching.NewEngine()

	// Initialize book with liquidity
	for i := 0; i < 50; i++ {
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 99.0+float64(i)*0.01, 100))
		engine.PlaceOrder(matching.NewOrder(uint64(i+50), matching.LimitOrder, matching.Sell, 101.0+float64(i)*0.01, 100))
	}

	rand.Seed(time.Now().UnixNano())
	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orderType := matching.LimitOrder
		side := matching.Buy
		price := 100.0

		if rand.Float64() > 0.5 {
			side = matching.Sell
			price = 101.0
		}

		orders[i] = matching.NewOrder(uint64(i+1000), orderType, side, price, 10)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.PlaceOrder(orders[i])
	}

	ordersPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
	avgLatency := b.Elapsed().Nanoseconds() / int64(b.N)
	b.ReportMetric(float64(avgLatency)/1000.0, "µs/op")
}

// BenchmarkMixedOperations benchmarks realistic mix of operations
func BenchmarkMixedOperations(b *testing.B) {
	engine := matching.NewEngine()

	// Initialize with some liquidity
	for i := 0; i < 20; i++ {
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 99.0+float64(i)*0.01, 50))
		engine.PlaceOrder(matching.NewOrder(uint64(i+20), matching.LimitOrder, matching.Sell, 101.0+float64(i)*0.01, 50))
	}

	rand.Seed(time.Now().UnixNano())
	operations := make([]func(), b.N)

	for i := 0; i < b.N; i++ {
		r := rand.Float64()
		switch {
		case r < 0.4: // 40% limit orders
			side := matching.Buy
			price := 99.5
			if rand.Float64() > 0.5 {
				side = matching.Sell
				price = 101.5
			}
			order := matching.NewOrder(uint64(i+1000), matching.LimitOrder, side, price, 10)
			operations[i] = func() { engine.PlaceOrder(order) }

		case r < 0.7: // 30% market orders
			side := matching.Buy
			if rand.Float64() > 0.5 {
				side = matching.Sell
			}
			order := matching.NewOrder(uint64(i+1000), matching.MarketOrder, side, 0.0, 10)
			operations[i] = func() { engine.PlaceOrder(order) }

		default: // 30% cancellations
			id := uint64(rand.Intn(1000) + 1000)
			operations[i] = func() { engine.CancelOrder(id) }
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		operations[i]()
	}

	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
	avgLatency := b.Elapsed().Nanoseconds() / int64(b.N)
	b.ReportMetric(float64(avgLatency)/1000.0, "µs/op")
}

// BenchmarkWorstCaseSearch benchmarks search in worst case (order at end)
func BenchmarkWorstCaseSearch(b *testing.B) {
	ob := matching.NewOrderBook()

	// Add many orders at same price
	for i := 0; i < 1000; i++ {
		ob.AddBidOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 100.0, 10))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Search for last order (worst case)
		ob.SearchById(999)
	}

	searchesPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(searchesPerSec, "searches/sec")
}

// BenchmarkBestCaseSearch benchmarks search in best case (order at start)
func BenchmarkBestCaseSearch(b *testing.B) {
	ob := matching.NewOrderBook()

	// Add many orders at different prices
	for i := 0; i < 1000; i++ {
		price := 100.0 + float64(i)*0.01
		ob.AddBidOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, price, 10))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Search for first order in first price level (best case)
		ob.SearchById(0)
	}

	searchesPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(searchesPerSec, "searches/sec")
}

// BenchmarkFullExecution benchmarks complete order execution cycle
func BenchmarkFullExecution(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := matching.NewEngine()

		// Add liquidity
		engine.PlaceOrder(matching.NewOrder(1, matching.LimitOrder, matching.Sell, 101.0, 100))

		// Execute market order
		engine.PlaceOrder(matching.NewOrder(2, matching.MarketOrder, matching.Buy, 0.0, 50))

		// Cancel remaining
		engine.CancelOrder(1)
	}

	cyclesPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(cyclesPerSec, "cycles/sec")
}

// BenchmarkLargeOrderPartialFill benchmarks large orders with many partial fills
func BenchmarkLargeOrderPartialFill(b *testing.B) {
	engine := matching.NewEngine()

	// Add many small orders
	for i := 0; i < 100; i++ {
		price := 101.0 + float64(i)*0.01
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Sell, price, 10))
	}

	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orders[i] = matching.NewOrder(uint64(i+1000), matching.MarketOrder, matching.Buy, 0.0, 500)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.PlaceOrder(orders[i])
		// Replenish liquidity
		for j := 0; j < 100; j++ {
			price := 101.0 + float64(j)*0.01
			engine.PlaceOrder(matching.NewOrder(uint64(i*100+j+10000), matching.LimitOrder, matching.Sell, price, 10))
		}
	}

	ordersPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
}

// BenchmarkPriceTimePriority benchmarks FIFO execution at same price
func BenchmarkPriceTimePriority(b *testing.B) {
	engine := matching.NewEngine()

	// Add many orders at same price
	for i := 0; i < 100; i++ {
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Sell, 101.0, 10))
	}

	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		orders[i] = matching.NewOrder(uint64(i+1000), matching.MarketOrder, matching.Buy, 0.0, 10)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.PlaceOrder(orders[i])
		// Add replacement order
		engine.PlaceOrder(matching.NewOrder(uint64(i+10000), matching.LimitOrder, matching.Sell, 101.0, 10))
	}

	ordersPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(ordersPerSec, "orders/sec")
}

// BenchmarkMemoryUsage benchmarks memory usage under load
func BenchmarkMemoryUsage(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		engine := matching.NewEngine()

		// Simulate realistic orderbook
		for j := 0; j < 1000; j++ {
			bidPrice := 100.0 - float64(j)*0.01
			askPrice := 101.0 + float64(j)*0.01
			engine.PlaceOrder(matching.NewOrder(uint64(j*2), matching.LimitOrder, matching.Buy, bidPrice, 100))
			engine.PlaceOrder(matching.NewOrder(uint64(j*2+1), matching.LimitOrder, matching.Sell, askPrice, 100))
		}
	}

	// Memory allocations are automatically reported by b.ReportAllocs()
}

// BenchmarkThroughputStressTest stress tests maximum throughput
func BenchmarkThroughputStressTest(b *testing.B) {
	engine := matching.NewEngine()

	// Pre-populate with deep liquidity
	for i := 0; i < 500; i++ {
		engine.PlaceOrder(matching.NewOrder(uint64(i), matching.LimitOrder, matching.Buy, 95.0+float64(i)*0.01, 1000))
		engine.PlaceOrder(matching.NewOrder(uint64(i+500), matching.LimitOrder, matching.Sell, 105.0+float64(i)*0.01, 1000))
	}

	rand.Seed(time.Now().UnixNano())
	orders := make([]*matching.Order, b.N)
	for i := 0; i < b.N; i++ {
		if rand.Float64() > 0.5 {
			orders[i] = matching.NewOrder(uint64(i+10000), matching.LimitOrder, matching.Buy, 100.0, 10)
		} else {
			orders[i] = matching.NewOrder(uint64(i+10000), matching.LimitOrder, matching.Sell, 101.0, 10)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.PlaceOrder(orders[i])
	}

	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "orders/sec")
	avgLatency := b.Elapsed().Nanoseconds() / int64(b.N)
	b.ReportMetric(float64(avgLatency)/1000.0, "µs/op")

	// Print summary
	b.Logf("\n=== Throughput Stress Test Summary ===")
	b.Logf("Total Orders: %d", b.N)
	b.Logf("Throughput: %.2f orders/sec", throughput)
	b.Logf("Avg Latency: %.2f µs/op", float64(avgLatency)/1000.0)
}

// PrintBenchmarkSummary prints a summary of key performance indicators
func PrintBenchmarkSummary(b *testing.B) {
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         MATCHING ENGINE PERFORMANCE SUMMARY                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("\nKey Performance Indicators (KPIs):")
	fmt.Println("  • Order Throughput: Orders processed per second")
	fmt.Println("  • Latency: Average time per operation (microseconds)")
	fmt.Println("  • Memory: Allocations per operation")
	fmt.Println("  • Scalability: Performance across different book depths")
	fmt.Println("\nRun benchmarks with:")
	fmt.Println("  go test -bench=. -benchmem -benchtime=10s")
	fmt.Println("\nFor detailed profiling:")
	fmt.Println("  go test -bench=BenchmarkThroughputStressTest -cpuprofile=cpu.prof")
	fmt.Println("  go test -bench=BenchmarkMemoryUsage -memprofile=mem.prof")
	fmt.Println("  go tool pprof cpu.prof")
	fmt.Println("  go tool pprof mem.prof")
}
