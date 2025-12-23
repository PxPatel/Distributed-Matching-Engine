# Matching Engine Test Suite

Comprehensive test suite for the distributed matching engine, including unit tests, integration tests, and performance benchmarks.

## Overview

This test suite provides:
- **High test coverage** of all matching engine components
- **Edge case testing** for robustness validation
- **Performance benchmarks** with key performance indicators (KPIs)
- **Maintainable structure** following Go testing best practices

## Test Files

### 1. `order_test.go`
Tests for the Order type and related functionality.

**Coverage:**
- Order type constants (NoActionOrder, MarketOrder, LimitOrder, etc.)
- Side type constants (Buy, Sell)
- Order creation via `NewOrder()`
- Order validation via `IsValid()`
- Order size modification via `SetSize()`
- Edge cases: zero IDs, extreme prices, concurrent creation
- Timestamp accuracy

**Key Test Cases:**
- Valid and invalid order configurations
- Order field accessibility and modification
- Concurrent order creation (1000 orders)
- Edge case price values (very small, very large)

### 2. `orderbook_test.go`
Tests for the OrderBook data structure and operations.

**Coverage:**
- OrderBook creation
- Adding orders (bids and asks)
- Retrieving best bid/ask
- Searching orders by ID
- Deleting orders (individual and entire price levels)
- Price level management
- Price-time priority (FIFO at each price level)

**Key Test Cases:**
- Empty orderbook operations
- Multiple orders at same price level
- Large orderbook (1000+ orders)
- Bid-ask spread calculation
- Crossed books (bid ≥ ask)
- Edge case price values
- Concurrent access scenarios

### 3. `engine_test.go`
Tests for the matching engine logic and trade execution.

**Coverage:**
- Market order execution (buy and sell)
- Limit order execution (immediate fill and add to book)
- Order cancellation
- Partial fills and complete fills
- Multiple trades from single order
- Price-time priority enforcement
- Trade generation and recording

**Key Test Cases:**
- Market orders with full/partial/no liquidity
- Limit orders with immediate match vs. resting
- Aggressive limit orders crossing spread
- Price improvement scenarios
- Sequential trade execution
- Large order execution (100+ fills)
- Edge cases: zero size, exact fills
- Concurrent order placement

### 4. `benchmark_test.go`
Performance benchmarks and KPI measurements.

**Benchmarks:**
- `BenchmarkOrderCreation` - Order instantiation speed
- `BenchmarkOrderValidation` - Validation throughput
- `BenchmarkAddBidOrder` / `BenchmarkAddAskOrder` - Book add performance
- `BenchmarkGetBestBid` / `BenchmarkGetBestAsk` - Best price lookup speed
- `BenchmarkSearchById` - Order search performance
- `BenchmarkMarketOrderExecution` - Market order latency
- `BenchmarkLimitOrderExecution` - Limit order latency
- `BenchmarkCancelOrder` - Cancellation speed
- `BenchmarkOrderBookDepth_*` - Scalability tests (10, 100, 1K, 10K levels)
- `BenchmarkHighFrequencyTrading` - HFT simulation
- `BenchmarkMixedOperations` - Realistic operation mix
- `BenchmarkThroughputStressTest` - Maximum throughput test

**KPIs Measured:**
- Orders/second throughput
- Average latency (microseconds per operation)
- Memory allocations per operation
- Scalability across different book depths

## Running Tests

### Run All Tests
```bash
cd internal/matching/tests
go test -v
```

### Run Specific Test File
```bash
go test -v -run TestOrder          # Order tests
go test -v -run TestOrderBook      # OrderBook tests
go test -v -run TestEngine         # Engine tests
```

### Run Specific Test Case
```bash
go test -v -run TestPlaceMarketOrderBuy
go test -v -run TestPriceTimePriority
```

### Generate Coverage Report
```bash
# Generate coverage profile
go test -coverprofile=coverage.out

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Run All Tests with Coverage
```bash
go test -v -cover
```

## Running Benchmarks

### Run All Benchmarks
```bash
go test -bench=. -benchmem
```

### Run Specific Benchmark
```bash
go test -bench=BenchmarkMarketOrderExecution -benchmem
go test -bench=BenchmarkOrderBookDepth -benchmem
```

### Run Benchmarks with Extended Time
```bash
# Run for 10 seconds per benchmark for more accurate results
go test -bench=. -benchmem -benchtime=10s
```

### Run Benchmarks Multiple Times
```bash
# Run each benchmark 5 times to get average
go test -bench=. -benchmem -count=5
```

### Compare Benchmark Results
```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks and save results
go test -bench=. -benchmem -count=10 > before.txt

# Make changes, then run again
go test -bench=. -benchmem -count=10 > after.txt

# Compare
benchstat before.txt after.txt
```

## Performance Profiling

### CPU Profile
```bash
# Generate CPU profile
go test -bench=BenchmarkThroughputStressTest -cpuprofile=cpu.prof

# Analyze with pprof
go tool pprof cpu.prof

# Commands in pprof:
# - top10: Show top 10 functions by CPU time
# - list <function>: Show source code with annotations
# - web: Generate visualization (requires graphviz)
```

### Memory Profile
```bash
# Generate memory profile
go test -bench=BenchmarkMemoryUsage -memprofile=mem.prof

# Analyze with pprof
go tool pprof mem.prof

# Commands in pprof:
# - top10: Show top 10 functions by memory allocation
# - list <function>: Show source code with annotations
```

### Trace Analysis
```bash
# Generate execution trace
go test -bench=BenchmarkHighFrequencyTrading -trace=trace.out

# View trace
go tool trace trace.out
```

## Expected Performance Targets

Based on benchmark results, here are typical performance targets:

| Operation | Target Throughput | Target Latency |
|-----------|------------------|----------------|
| Order Creation | > 10M ops/sec | < 100 ns/op |
| Add to OrderBook | > 1M ops/sec | < 1 µs/op |
| Best Bid/Ask Lookup | > 100K ops/sec | < 10 µs/op |
| Market Order Execution | > 100K ops/sec | < 10 µs/op |
| Limit Order Execution | > 100K ops/sec | < 10 µs/op |
| Order Cancellation | > 500K ops/sec | < 2 µs/op |

**Note:** Actual performance depends on hardware, book depth, and operation mix.

## Test Organization

### Test Naming Convention
- Test functions: `Test<Component><Functionality>`
- Benchmark functions: `Benchmark<Operation>`
- Helper functions: `<descriptiveName>` (lowercase start)

### Test Structure
Each test follows the pattern:
1. **Setup** - Create necessary objects
2. **Action** - Perform the operation being tested
3. **Assert** - Verify expected outcomes
4. **Cleanup** - (if needed)

### Table-Driven Tests
Most tests use table-driven approach for maintainability:
```go
tests := []struct {
    name     string
    input    type
    expected type
}{
    {"case1", input1, expected1},
    {"case2", input2, expected2},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

## Edge Cases Covered

### Order Tests
- Invalid order types and sides
- Zero and negative sizes
- Zero and negative prices
- Very small prices (0.0001)
- Very large prices (999999999.99)
- Large quantities (max int32)
- Concurrent order creation

### OrderBook Tests
- Empty orderbook operations
- Single order operations
- Multiple orders at same price
- Large orderbooks (1000+ orders)
- Duplicate order IDs
- Crossed books
- Price level cleanup
- Concurrent access

### Engine Tests
- No liquidity scenarios
- Insufficient liquidity
- Exact fills
- Partial fills
- Multiple partial fills
- Price-time priority
- Self-matching (allowed at engine level)
- Zero size orders
- Very large orders (1000+ fills)
- Sequential operations
- Concurrent operations

## Continuous Integration

To integrate into CI/CD pipeline:

```yaml
# Example GitHub Actions workflow
- name: Run Tests
  run: |
    cd internal/matching/tests
    go test -v -race -coverprofile=coverage.out
    go tool cover -func=coverage.out

- name: Run Benchmarks
  run: |
    cd internal/matching/tests
    go test -bench=. -benchmem -benchtime=5s
```

## Future Improvements

Potential areas for test expansion:
- [ ] Concurrent stress tests with goroutines
- [ ] Fuzz testing for order validation
- [ ] Property-based testing
- [ ] Integration tests with real network I/O
- [ ] Performance regression tests
- [ ] Load testing scenarios
- [ ] Chaos testing (random failures)

## Contributing

When adding new tests:
1. Follow existing naming conventions
2. Use table-driven tests where applicable
3. Include both positive and negative test cases
4. Add edge case coverage
5. Update this README with new test descriptions
6. Ensure tests are deterministic and reproducible

## Troubleshooting

### Tests Failing
- Ensure you're in the correct directory
- Check Go version compatibility (requires Go 1.25.1)
- Verify all dependencies are installed

### Benchmarks Inconsistent
- Close other applications to reduce noise
- Run benchmarks multiple times (`-count=10`)
- Use `-benchtime=10s` for longer, more stable runs
- Check CPU frequency scaling settings

### Coverage Issues
- Ensure test package name matches source package
- Use `go test -cover -v` for detailed output
- Check for unexported functions (they need to be in same package)

## Contact

For questions or issues with the test suite, please open an issue in the repository.
