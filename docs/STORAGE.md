# Storage Layer Documentation

## Overview

The matching engine uses a **pluggable storage architecture** that supports multiple deployment modes from simple in-memory operation to fully distributed systems with Redis caching and PostgreSQL persistence.

The storage layer is built on two core interfaces:
- **OrderStore**: Manages order lifecycle (create, read, update, delete, query)
- **TradeStore**: Manages trade persistence and retrieval

## Architecture

### Interface-Based Design

All storage operations go through well-defined interfaces in `internal/storage/interfaces.go`:

```go
type OrderStore interface {
    Save(order *Order) error
    Get(orderID uint64) (*Order, error)
    Remove(orderID uint64) error
    Update(order *Order) error
    GetAll() []*Order
    GetByUser(userID string) []*Order
    GetBySide(side SideType) []*Order
    Close() error
}

type TradeStore interface {
    Save(trade *Trade) error
    SaveBatch(trades []*Trade) error
    GetRecent(limit int) ([]*Trade, error)
    Close() error
}
```

This design allows the matching engine to work with any storage backend without code changes.

### Composite Pattern for Layered Caching

The `CompositeOrderStore` and `CompositeTradeStore` combine multiple storage implementations:

**Write Path**: Data is written to **ALL** layers (write-through)
**Read Path**: Data is read from the **FIRST** layer that succeeds (fast path optimization)

Example configuration:
```go
store := NewCompositeTradeStore(
    memoryStore,    // L1: 100ns latency
    redisStore,     // L2: 1ms latency
    postgresStore,  // L3: 5ms latency
)
```

Read attempts: memory → redis → postgres (first hit wins)
Write broadcasts: memory **AND** redis **AND** postgres

---

## Available Store Implementations

### 1. InMemoryOrderStore

**Location**: `internal/storage/memory_order_store.go`

**Use Case**: Single-node deployment, development, testing

**Characteristics**:
- Thread-safe with `sync.RWMutex`
- O(1) save, get, remove operations
- O(n) query operations (GetAll, GetByUser, GetBySide)
- No persistence - all data lost on restart
- No capacity limit (unbounded growth)

**When to Use**:
- Development and testing
- Single-node production with < 100K orders
- When you don't need data durability

---

### 2. InMemoryTradeStore

**Location**: `internal/storage/memory_trade_store.go`

**Use Case**: Recent trade caching

**Characteristics**:
- Circular buffer with configurable size (default: 1000 trades)
- Thread-safe with `sync.RWMutex`
- O(1) append, O(1) recent retrieval
- Automatically trims to max size
- No persistence

**Configuration**:
```bash
TRADE_HISTORY_SIZE=1000  # Number of recent trades to cache
```

**When to Use**:
- As L1 cache in composite trade store
- When you only need recent trades (API queries)
- Combined with FileTradeStore or PostgresTradeStore for durability

---

### 3. FileTradeStore

**Location**: `internal/storage/file_trade_store.go`

**Use Case**: Audit trail, compliance, disaster recovery

**Characteristics**:
- Append-only NDJSON format (newline-delimited JSON)
- Async writes (non-blocking, uses goroutines)
- Write-only (no read support in store itself)
- Sequential I/O optimized (~10-50μs per trade)
- Durable and tamper-evident

**Format**:
```json
{"trade_id":1,"buy_order_id":42,"sell_order_id":99,"price":100.5,"quantity":10,"timestamp":"2025-01-15T10:30:45.123Z"}
{"trade_id":2,"buy_order_id":43,"sell_order_id":100,"price":101.0,"quantity":5,"timestamp":"2025-01-15T10:31:12.456Z"}
```

**Configuration**:
```bash
TRADE_LOG_PATH=trades.log
```

**When to Use**:
- Always include in production for audit trail
- Compliance requirements (immutable trade record)
- Disaster recovery (can replay from log)

**Limitations**:
- No built-in reader (use external tools or add custom implementation)
- Single file (no rotation/compression built-in)

---

### 4. PostgresOrderStore

**Location**: `internal/storage/postgres_order_store.go`

**Use Case**: Persistent, queryable order storage

**Characteristics**:
- Connection pooling with pgx/v5
- Prepared statements for performance
- ACID transactions
- Automatic schema migrations
- Context-aware with timeouts (5s default)

**Schema**:
```sql
CREATE TABLE orders (
    order_id BIGINT PRIMARY KEY,
    user_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    order_type INTEGER,
    side INTEGER,
    price NUMERIC(18,8),
    size INTEGER,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

-- Indexes for fast queries
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_symbol ON orders(symbol);
CREATE INDEX idx_orders_side ON orders(side);
```

**Configuration**:
```bash
DATABASE_ENABLED=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=matching_engine
DATABASE_USER=postgres
DATABASE_PASSWORD=yourpassword
DATABASE_MAX_CONNECTIONS=20
DATABASE_CONN_MAX_LIFETIME=5m
DATABASE_SSL_MODE=disable  # or require, verify-full
```

**When to Use**:
- Production systems requiring data durability
- When you need to query historical orders
- Multi-instance deployments (shared state)
- When orders must survive restarts

---

### 5. PostgresTradeStore

**Location**: `internal/storage/postgres_trade_store.go`

**Use Case**: Historical trade queries, analytics

**Characteristics**:
- Batch insert support (efficient for high throughput)
- Indexed by timestamp for time-range queries
- Auto-incrementing trade IDs
- Connection pooling

**Schema**:
```sql
CREATE TABLE trades (
    trade_id BIGSERIAL PRIMARY KEY,
    buy_order_id BIGINT NOT NULL,
    sell_order_id BIGINT NOT NULL,
    price NUMERIC(18,8) NOT NULL,
    quantity INTEGER NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_trades_timestamp ON trades(timestamp DESC);
```

**When to Use**:
- Historical trade analysis
- Reporting and analytics
- When FileTradeStore is insufficient (need queries)

---

### 6. RedisOrderStore

**Location**: `internal/storage/redis_order_store.go`

**Use Case**: Distributed caching across multiple API nodes

**Characteristics**:
- TTL-based expiration (default: 24 hours)
- Pipeline operations for efficiency
- Automatic index management (user, side)
- JSON serialization
- Connection pooling with go-redis/v9

**Data Structures**:
```
Keys:
- order:{orderID}          → JSON (expires in 24h)
- user_orders:{userID}     → SET of order IDs
- side_orders:{side}       → SET of order IDs
```

**Configuration**:
```bash
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
```

**When to Use**:
- Multi-node API deployments (shared cache)
- When you want to reduce database load
- As L2 cache between memory and Postgres

**Limitations**:
- Orders expire after 24 hours (not for long-term storage)
- Eventual consistency in distributed setups

---

### 7. RedisTradeStore

**Location**: `internal/storage/redis_trade_store.go`

**Use Case**: Distributed recent trade caching

**Characteristics**:
- Sorted sets (ZSET) sorted by timestamp
- Automatic trimming (keeps last 10,000 trades)
- O(log n) insertion, O(1) recent retrieval
- JSON serialization

**Data Structure**:
```
Key: trades:recent
Type: ZSET (sorted set)
Score: Timestamp (nanoseconds)
Member: JSON-encoded trade
Size: Last 10,000 trades
```

**When to Use**:
- Multi-node deployments needing shared recent trades
- WebSocket servers broadcasting trade feeds
- Reducing load on PostgreSQL for recent trade queries

---

## Deployment Modes

### Mode 1: Development / In-Memory Only

**Configuration**:
```bash
DATABASE_ENABLED=false
REDIS_ENABLED=false
```

**Storage Stack**:
- Orders: `InMemoryOrderStore`
- Trades: `CompositeTradeStore(InMemoryTradeStore, FileTradeStore)`

**Characteristics**:
- ✅ Fastest (no network I/O)
- ✅ Zero infrastructure dependencies
- ❌ No data durability (orders lost on restart)
- ❌ Single node only

**Use Cases**: Local development, testing, demos

---

### Mode 2: Single-Node Production

**Configuration**:
```bash
DATABASE_ENABLED=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
REDIS_ENABLED=false
```

**Storage Stack**:
- Orders: `CompositeOrderStore(InMemoryOrderStore, PostgresOrderStore)`
- Trades: `CompositeTradeStore(InMemoryTradeStore, PostgresTradeStore, FileTradeStore)`

**Characteristics**:
- ✅ Data persists across restarts
- ✅ Historical queries possible
- ✅ ACID guarantees
- ❌ Single point of failure
- ❌ Cannot scale horizontally

**Use Cases**: Small production systems, < 10K orders/sec

---

### Mode 3: Distributed Production

**Configuration**:
```bash
DATABASE_ENABLED=true
DATABASE_HOST=postgres.internal
REDIS_ENABLED=true
REDIS_HOST=redis.internal
```

**Storage Stack**:
- Orders: `CompositeOrderStore(InMemoryOrderStore, RedisOrderStore, PostgresOrderStore)`
- Trades: `CompositeTradeStore(InMemoryTradeStore, RedisTradeStore, PostgresTradeStore, FileTradeStore)`

**Characteristics**:
- ✅ Horizontal scalability (multiple API nodes)
- ✅ Shared cache reduces DB load
- ✅ Sub-millisecond cache hits
- ✅ Full data durability
- ⚠️  Higher complexity

**Use Cases**: Production at scale, multi-region, high availability

**Read Path**:
```
API Node 1/2/3 → Memory (L1) → Redis (L2) → Postgres (L3)
                     100ns         1ms          5ms
```

**Write Path**:
```
Matching Engine → Memory + Redis + Postgres + File (parallel)
```

---

## Configuration Guide

### Enabling PostgreSQL

1. **Install PostgreSQL**:
```bash
# macOS
brew install postgresql
brew services start postgresql

# Ubuntu
sudo apt-get install postgresql
sudo systemctl start postgresql

# Docker
docker run -d --name postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15-alpine
```

2. **Create Database**:
```bash
createdb matching_engine
```

3. **Configure .env**:
```bash
DATABASE_ENABLED=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=matching_engine
DATABASE_USER=postgres
DATABASE_PASSWORD=yourpassword
```

4. **Start Server** (migrations run automatically):
```bash
go run cmd/api/server.go
```

### Enabling Redis

1. **Install Redis**:
```bash
# macOS
brew install redis
brew services start redis

# Ubuntu
sudo apt-get install redis-server
sudo systemctl start redis

# Docker
docker run -d --name redis \
  -p 6379:6379 \
  redis:7-alpine
```

2. **Configure .env**:
```bash
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=  # Leave empty if no auth
```

3. **Start Server**:
```bash
go run cmd/api/server.go
```

### Connection Tuning

**PostgreSQL Pool Sizing**:
```bash
# Conservative (default)
DATABASE_MAX_CONNECTIONS=20
DATABASE_MAX_IDLE_CONNECTIONS=5

# High throughput
DATABASE_MAX_CONNECTIONS=50
DATABASE_MAX_IDLE_CONNECTIONS=20
```

**Redis Pool Sizing**:
```bash
# Conservative
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=2

# High throughput
REDIS_POOL_SIZE=50
REDIS_MIN_IDLE_CONNS=10
```

---

## Creating Custom Stores

### Implementing OrderStore

Example: Custom SQLite store

```go
package storage

import (
    "database/sql"
    "github.com/PxPatel/trading-system/internal/types"
)

type SQLiteOrderStore struct {
    db *sql.DB
}

func NewSQLiteOrderStore(path string) (*SQLiteOrderStore, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }
    return &SQLiteOrderStore{db: db}, nil
}

func (s *SQLiteOrderStore) Save(order *types.Order) error {
    _, err := s.db.Exec(
        "INSERT OR REPLACE INTO orders VALUES (?, ?, ?, ?)",
        order.ID, order.UserID, order.Price, order.Size,
    )
    return err
}

// Implement remaining interface methods...
```

### Registering Custom Stores

In `cmd/api/server.go`, modify `buildStorageLayers()`:

```go
func buildStorageLayers(cfg *config.Config) (storage.OrderStore, storage.TradeStore) {
    memOrderStore := storage.NewInMemoryOrderStore()

    // Add custom store
    sqliteStore, _ := storage.NewSQLiteOrderStore("orders.db")

    orderStore := storage.NewCompositeOrderStore(memOrderStore, sqliteStore)
    // ...
}
```

---

## Performance Considerations

### Cache Hit Rates

**Optimal Layering** (for 80% cache hit rate):
- L1 (Memory): Recent 1000-10000 items
- L2 (Redis): Recent 100K items or 24 hours
- L3 (Postgres): Full history

**Monitoring**:
- Track cache hits vs misses
- Adjust `TRADE_HISTORY_SIZE` based on access patterns
- Tune Redis TTL based on query age distribution

### Write Amplification

**CompositeTradeStore** writes to ALL layers:
```
Single trade → 4 writes (Memory + Redis + Postgres + File)
```

**Optimization**:
- Use batch writes where possible (`SaveBatch`)
- FileTradeStore is async (non-blocking)
- Redis/Postgres use connection pooling

**Acceptable for**:
- Trade volume < 10K/sec
- Network latency < 5ms

### Connection Pooling

**Too Few Connections**: Requests queue, high latency
**Too Many Connections**: DB/Redis resource exhaustion

**Rule of Thumb**:
```
Max Connections = (CPU cores * 2) + disk spindles
```

For PostgreSQL on 4-core machine with SSD:
```bash
DATABASE_MAX_CONNECTIONS=10  # (4*2 + 1 disk)
```

---

## Troubleshooting

### Issue: "Failed to connect to PostgreSQL"

**Check**:
1. Is PostgreSQL running? `pg_isready`
2. Can you connect manually? `psql -h localhost -U postgres`
3. Check credentials in `.env`
4. Check firewall rules

**Solution**:
```bash
# Test connection
psql -h localhost -U postgres -d matching_engine

# Check logs
tail -f /usr/local/var/log/postgresql.log  # macOS
sudo journalctl -u postgresql  # Linux
```

### Issue: "Failed to connect to Redis"

**Check**:
1. Is Redis running? `redis-cli ping`
2. Check host/port in `.env`
3. Check password if AUTH enabled

**Solution**:
```bash
# Test connection
redis-cli -h localhost -p 6379 ping

# Check logs
redis-cli INFO server
```

### Issue: High Memory Usage

**Cause**: Unbounded order tracking in InMemoryOrderStore

**Solutions**:
1. Enable PostgreSQL: `DATABASE_ENABLED=true`
2. Remove in-memory layer from orderStore (use Redis + Postgres only)
3. Implement order cleanup (periodic deletion of filled orders)

### Issue: Slow Trade Queries

**Symptoms**: `/api/v1/trades` endpoint > 100ms

**Solutions**:
1. Increase `TRADE_HISTORY_SIZE` for memory cache
2. Enable Redis: `REDIS_ENABLED=true`
3. Add database indexes:
```sql
CREATE INDEX idx_trades_timestamp_desc ON trades(timestamp DESC);
```

---

## Migration Guide

### From In-Memory to PostgreSQL

**Steps**:
1. Install PostgreSQL
2. Update `.env`: `DATABASE_ENABLED=true`
3. Restart server (migrations run automatically)
4. Existing orders lost (expected - they were in-memory only)
5. New orders persist to database

**No code changes required** - all configuration-driven.

### Adding Redis to Existing Deployment

**Steps**:
1. Install Redis
2. Update `.env`: `REDIS_ENABLED=true`
3. Restart server
4. Verify logs show "Redis cache connected successfully"

**Graceful Degradation**: If Redis fails to connect, server continues with Postgres only.

---

## Best Practices

✅ **DO**:
- Always enable FileTradeStore for audit trail
- Use PostgreSQL for production data durability
- Use Redis for multi-node deployments
- Monitor connection pool exhaustion
- Set reasonable TTLs in Redis (24h for orders)
- Use batch operations where possible

❌ **DON'T**:
- Rely solely on in-memory storage in production
- Use Redis as primary data store (use Postgres)
- Disable FileTradeStore (compliance risk)
- Set unlimited connection pools
- Ignore connection failures (check logs)

---

## Future Enhancements

Planned features (not yet implemented):

1. **Trade Log Reader**: Query FileTradeStore directly
2. **MongoDB Support**: Alternative to PostgreSQL
3. **S3 Storage**: Archive old trades to object storage
4. **Compression**: Compress old trades in database
5. **Sharding**: Partition by symbol for scaling
6. **Read Replicas**: Separate read/write databases
7. **Circuit Breaker**: Disable failing backends automatically
8. **Metrics**: Prometheus metrics for cache hit rates

---

## Summary

The storage layer provides:
- **Flexibility**: Swap backends without code changes
- **Performance**: Layered caching (memory → redis → postgres)
- **Reliability**: Graceful degradation when backends fail
- **Scalability**: Horizontal scaling with shared Redis/Postgres
- **Simplicity**: Configuration-driven deployment modes

Choose your deployment mode based on requirements:
- **Development**: In-memory only
- **Single Node**: Memory + Postgres
- **Distributed**: Memory + Redis + Postgres

All modes share the same codebase with zero code changes - just update `.env` configuration.
