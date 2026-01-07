# Quick Start Guide

Get up and running with the Distributed Matching Engine in under 5 minutes.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Quick Start Scenarios](#quick-start-scenarios)
4. [Configuration](#configuration)
5. [API Usage Examples](#api-usage-examples)
6. [Next Steps](#next-steps)

---

## Prerequisites

- **Go 1.21+** installed ([download](https://go.dev/dl/))
- **Git** for cloning the repository
- **Optional**: PostgreSQL 15+ and/or Redis 7+ (for persistent/distributed storage)

Check your Go version:
```bash
go version  # Should show 1.21 or higher
```

---

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/PxPatel/trading-system.git
cd trading-system
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Server

```bash
go build -o server cmd/api/server.go
```

This creates an executable named `server` in your current directory.

---

## Quick Start Scenarios

Choose the scenario that best fits your needs:

### Scenario 1: Local Development (In-Memory Only)

**Best for**: Learning, testing, development

**Setup Time**: 30 seconds

**Steps**:

1. No additional setup needed - uses in-memory storage by default

2. Start the server:
```bash
./server
```

3. Server is now running on `http://localhost:8080`

**Configuration**: Uses defaults from `.env.example` (all storage in memory, trades logged to file)

**Trade-offs**:
- ✅ Zero dependencies
- ✅ Fastest startup
- ❌ Orders lost on restart (trades preserved in `trades.log`)

---

### Scenario 2: Production Single-Node (PostgreSQL)

**Best for**: Small-to-medium production deployments, single server

**Setup Time**: 3 minutes

**Steps**:

1. **Install and start PostgreSQL**:
```bash
# macOS
brew install postgresql@15
brew services start postgresql@15

# Ubuntu/Debian
sudo apt-get install postgresql-15
sudo systemctl start postgresql

# Docker
docker run -d --name postgres \
  -e POSTGRES_PASSWORD=yourpassword \
  -p 5432:5432 \
  postgres:15-alpine
```

2. **Create the database**:
```bash
createdb matching_engine

# Or using psql:
psql -U postgres -c "CREATE DATABASE matching_engine;"
```

3. **Create `.env` file**:
```bash
cp .env.example .env
```

Edit `.env` to enable PostgreSQL:
```bash
# Memory Storage (L1 cache)
MEMORY_ENABLED=true
MEMORY_MAX_ORDERS=100000
MEMORY_MAX_TRADES=1000

# Database Configuration (L2 persistent)
DATABASE_ENABLED=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=matching_engine
DATABASE_USER=postgres
DATABASE_PASSWORD=yourpassword
```

4. **Start the server**:
```bash
./server
```

**Configuration**: Memory (fast cache) + PostgreSQL (persistent storage)

**Trade-offs**:
- ✅ Data survives restarts
- ✅ Historical queries possible
- ✅ ACID guarantees
- ⚠️  Slightly slower than pure in-memory (sub-millisecond overhead)

---

### Scenario 3: Distributed Production (PostgreSQL + Redis)

**Best for**: High-availability, multi-node deployments, horizontal scaling

**Setup Time**: 5 minutes

**Steps**:

1. **Follow Scenario 2 to set up PostgreSQL**

2. **Install and start Redis**:
```bash
# macOS
brew install redis
brew services start redis

# Ubuntu/Debian
sudo apt-get install redis-server
sudo systemctl start redis

# Docker
docker run -d --name redis \
  -p 6379:6379 \
  redis:7-alpine
```

3. **Test Redis connection**:
```bash
redis-cli ping  # Should return "PONG"
```

4. **Update `.env` to enable Redis**:
```bash
# Memory Storage (L1 cache)
MEMORY_ENABLED=true
MEMORY_MAX_ORDERS=100000
MEMORY_MAX_TRADES=1000

# Redis Configuration (L2 distributed cache)
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_ORDER_TTL=24h
REDIS_MAX_ORDERS=50000
REDIS_MAX_TRADES=10000

# Database Configuration (L3 persistent)
DATABASE_ENABLED=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=matching_engine
DATABASE_USER=postgres
DATABASE_PASSWORD=yourpassword
```

5. **Start multiple server instances** (optional - for horizontal scaling):
```bash
# Terminal 1
PORT=8080 ./server

# Terminal 2
PORT=8081 ./server

# Terminal 3
PORT=8082 ./server
```

All instances share the same Redis cache and PostgreSQL database.

**Configuration**: Memory (L1) → Redis (L2) → PostgreSQL (L3) + File audit log

**Trade-offs**:
- ✅ Horizontal scaling across multiple nodes
- ✅ Shared cache reduces DB load
- ✅ Sub-millisecond cache hits
- ✅ Full data durability
- ⚠️  More complex infrastructure

---

## Configuration

### Environment Variables

The server is configured via environment variables (`.env` file or system environment).

**Core Settings**:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `INFO` | Logging level (DEBUG, INFO, WARN, ERROR) |

**Memory Storage**:

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY_ENABLED` | `true` | Enable in-memory caching layer |
| `MEMORY_MAX_ORDERS` | `100000` | Max inactive orders in memory (FIFO eviction) |
| `MEMORY_MAX_TRADES` | `1000` | Max recent trades in memory |

**PostgreSQL**:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_ENABLED` | `false` | Enable PostgreSQL persistence |
| `DATABASE_HOST` | `localhost` | PostgreSQL host |
| `DATABASE_PORT` | `5432` | PostgreSQL port |
| `DATABASE_NAME` | `matching_engine` | Database name |
| `DATABASE_USER` | `postgres` | Database user |
| `DATABASE_PASSWORD` | *(empty)* | Database password |
| `DATABASE_MAX_CONNECTIONS` | `20` | Connection pool size |

**Redis**:

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ENABLED` | `false` | Enable Redis distributed cache |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | *(empty)* | Redis password (if auth enabled) |
| `REDIS_TLS_ENABLED` | `false` | Enable TLS/SSL encryption (required for Upstash, AWS ElastiCache with encryption) |
| `REDIS_ORDER_TTL` | `24h` | Time-to-live for orders in cache |
| `REDIS_MAX_ORDERS` | `50000` | Max orders in Redis (FIFO eviction) |
| `REDIS_MAX_TRADES` | `10000` | Max trades in Redis (FIFO eviction) |

**Files**:

| Variable | Default | Description |
|----------|---------|-------------|
| `TRADE_LOG_PATH` | `trades.log` | Path to append-only trade log file |

### Example Configurations

**Minimal (Development)**:
```bash
# .env
PORT=8080
LOG_LEVEL=DEBUG
MEMORY_ENABLED=true
```

**Production (Single Node)**:
```bash
# .env
PORT=8080
LOG_LEVEL=INFO

MEMORY_ENABLED=true
MEMORY_MAX_ORDERS=100000
MEMORY_MAX_TRADES=1000

DATABASE_ENABLED=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=matching_engine
DATABASE_USER=postgres
DATABASE_PASSWORD=secure_password_here

TRADE_LOG_PATH=/var/log/matching_engine/trades.log
```

**Production (Distributed)**:
```bash
# .env
PORT=8080
LOG_LEVEL=INFO

MEMORY_ENABLED=true
MEMORY_MAX_ORDERS=100000
MEMORY_MAX_TRADES=1000

REDIS_ENABLED=true
REDIS_HOST=redis.internal.example.com
REDIS_PORT=6379
REDIS_PASSWORD=redis_auth_token
REDIS_ORDER_TTL=24h
REDIS_MAX_ORDERS=50000
REDIS_MAX_TRADES=10000

DATABASE_ENABLED=true
DATABASE_HOST=postgres.internal.example.com
DATABASE_PORT=5432
DATABASE_NAME=matching_engine
DATABASE_USER=matching_engine_user
DATABASE_PASSWORD=db_password_here
DATABASE_MAX_CONNECTIONS=50

TRADE_LOG_PATH=/var/log/matching_engine/trades.log
```

**Production with Cloud Redis (Upstash, AWS ElastiCache with TLS)**:
```bash
# .env
PORT=8080
LOG_LEVEL=INFO

MEMORY_ENABLED=true
MEMORY_MAX_ORDERS=100000
MEMORY_MAX_TRADES=1000

# Upstash Redis configuration (TLS required)
REDIS_ENABLED=true
REDIS_HOST=intimate-walleye-7953.upstash.io
REDIS_PORT=6379
REDIS_PASSWORD=your_upstash_password_here
REDIS_TLS_ENABLED=true
REDIS_ORDER_TTL=24h
REDIS_MAX_ORDERS=50000
REDIS_MAX_TRADES=10000

DATABASE_ENABLED=true
DATABASE_HOST=postgres.internal.example.com
DATABASE_PORT=5432
DATABASE_NAME=matching_engine
DATABASE_USER=matching_engine_user
DATABASE_PASSWORD=db_password_here
DATABASE_MAX_CONNECTIONS=50

TRADE_LOG_PATH=/var/log/matching_engine/trades.log
```

---

## API Usage Examples

The matching engine exposes a RESTful API on port 8080 (default).

### Health Check

```bash
curl http://localhost:8080/health
```

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:45.123Z"
}
```

---

### Place a Limit Buy Order

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "symbol": "AAPL",
    "order_type": 1,
    "side": 0,
    "price": 150.00,
    "size": 100
  }'
```

**Fields**:
- `order_type`: `1` = Limit Order, `0` = Market Order
- `side`: `0` = Buy, `1` = Sell
- `price`: Limit price (ignored for market orders)
- `size`: Number of shares

**Response**:
```json
{
  "order": {
    "id": 42,
    "user_id": "user123",
    "symbol": "AAPL",
    "order_type": 1,
    "side": 0,
    "price": 150.00,
    "size": 100,
    "timestamp": "2025-01-15T10:31:00.456Z"
  },
  "trades": []
}
```

---

### Place a Market Sell Order (Matches Immediately)

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user456",
    "symbol": "AAPL",
    "order_type": 0,
    "side": 1,
    "size": 50
  }'
```

**Response** (if matched with order #42):
```json
{
  "order": {
    "id": 43,
    "user_id": "user456",
    "symbol": "AAPL",
    "order_type": 0,
    "side": 1,
    "price": 0,
    "size": 50,
    "timestamp": "2025-01-15T10:32:15.789Z"
  },
  "trades": [
    {
      "buy_order_id": 42,
      "sell_order_id": 43,
      "price": 150.00,
      "size": 50,
      "timestamp": "2025-01-15T10:32:15.790Z"
    }
  ]
}
```

The market order matched with the best available bid (order #42 at $150.00).

---

### Get Order Book

```bash
curl http://localhost:8080/api/v1/orderbook?symbol=AAPL
```

**Response**:
```json
{
  "symbol": "AAPL",
  "bids": [
    {"price": 150.00, "size": 50, "orders": 1},
    {"price": 149.50, "size": 200, "orders": 3}
  ],
  "asks": [
    {"price": 150.50, "size": 100, "orders": 2},
    {"price": 151.00, "size": 150, "orders": 1}
  ],
  "timestamp": "2025-01-15T10:35:00.123Z"
}
```

---

### Get Recent Trades

```bash
curl http://localhost:8080/api/v1/trades?limit=10
```

**Response**:
```json
{
  "trades": [
    {
      "buy_order_id": 42,
      "sell_order_id": 43,
      "price": 150.00,
      "size": 50,
      "timestamp": "2025-01-15T10:32:15.790Z"
    },
    {
      "buy_order_id": 38,
      "sell_order_id": 41,
      "price": 149.75,
      "size": 100,
      "timestamp": "2025-01-15T10:28:30.456Z"
    }
  ]
}
```

---

### Get All Orders (for a specific user)

```bash
curl http://localhost:8080/api/v1/orders?user_id=user123
```

**Response**:
```json
{
  "orders": [
    {
      "id": 42,
      "user_id": "user123",
      "symbol": "AAPL",
      "order_type": 1,
      "side": 0,
      "price": 150.00,
      "size": 50,
      "timestamp": "2025-01-15T10:31:00.456Z"
    }
  ]
}
```

---

### Cancel an Order

```bash
curl -X DELETE http://localhost:8080/api/v1/orders/42
```

**Response**:
```json
{
  "message": "Order 42 cancelled successfully"
}
```

---

## Testing the Setup

### 1. Place Two Matching Orders

**Terminal 1** - Place a buy limit order:
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "alice",
    "symbol": "TSLA",
    "order_type": 1,
    "side": 0,
    "price": 200.00,
    "size": 10
  }'
```

**Terminal 2** - Place a matching sell order:
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "bob",
    "symbol": "TSLA",
    "order_type": 1,
    "side": 1,
    "price": 200.00,
    "size": 10
  }'
```

**Expected**: The second request returns a trade showing the match.

### 2. View the Trade Log

```bash
tail -f trades.log
```

You should see JSON lines like:
```json
{"buy_order_id":1,"sell_order_id":2,"price":200,"size":10,"timestamp":"2025-01-15T10:45:30.123Z"}
```

### 3. Query Recent Trades

```bash
curl http://localhost:8080/api/v1/trades?limit=1
```

Should return the trade you just created.

---

## Next Steps

### Learn More

- **[Storage Architecture](./STORAGE.md)**: Deep dive into storage layers, caching, and deployment modes
- **[API Documentation](./API.md)**: Complete REST API reference
- **[Configuration Guide](./CONFIG.md)**: Advanced configuration options

### Production Deployment

1. **Set up proper secrets management** (don't hardcode passwords in `.env`)
2. **Configure TLS/HTTPS** (use a reverse proxy like Nginx or Caddy)
3. **Enable monitoring** (add Prometheus metrics, log aggregation)
4. **Set up backups** (PostgreSQL dumps, trade log archiving)
5. **Tune connection pools** based on load (see [STORAGE.md](./STORAGE.md))

### Performance Tuning

- Increase `MEMORY_MAX_ORDERS` if you have many active orders
- Increase `REDIS_MAX_ORDERS` for distributed caching across nodes
- Tune `DATABASE_MAX_CONNECTIONS` based on your server CPU count
- Monitor cache hit rates and adjust limits accordingly

### Troubleshooting

**Server won't start?**
- Check logs for database connection errors
- Verify PostgreSQL/Redis are running: `pg_isready`, `redis-cli ping`
- Ensure ports are not in use: `lsof -i :8080`

**High memory usage?**
- Decrease `MEMORY_MAX_ORDERS` and `MEMORY_MAX_TRADES`
- Set `MEMORY_ENABLED=false` to disable in-memory caching
- Enable `DATABASE_ENABLED=true` to offload to PostgreSQL

**Slow API responses?**
- Enable Redis: `REDIS_ENABLED=true`
- Increase `MEMORY_MAX_TRADES` for better cache hit rate
- Check database indexes (automatically created on startup)

---

## Support

- **Issues**: [GitHub Issues](https://github.com/PxPatel/trading-system/issues)
- **Discussions**: [GitHub Discussions](https://github.com/PxPatel/trading-system/discussions)
- **Documentation**: See `docs/` directory

---

**Congratulations!** You now have a working matching engine. Start placing orders and building your trading application! 🚀
