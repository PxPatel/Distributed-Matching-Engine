# Phase 4: Kafka Integration - Implementation Plan

## Overview

Transform from synchronous request-response architecture to event-driven architecture using Kafka as the message broker.

### Current Architecture
```
HTTP Request → API Handler → engine.PlaceOrder() → Trades → HTTP Response
```

### Target Architecture
```
HTTP Request → API Handler → Kafka (orders.incoming) → HTTP Response (202 Accepted)
                                        ↓
                                  Engine Consumer
                                        ↓
                                  Process Order
                                        ↓
                          Kafka (trades.executed)
```

## Goals

1. ✅ Decouple API from matching engine
2. ✅ Enable horizontal scaling of engine instances
3. ✅ Add message durability and replay capability
4. ✅ Improve observability via message logs
5. ✅ Maintain existing storage layer (Redis, Postgres)

## Architecture Changes

### Kafka Topics

| Topic | Partitions | Purpose | Producer | Consumer |
|-------|------------|---------|----------|----------|
| `orders.incoming` | 1 (Phase 4) | New orders from API | API Server | Engine |
| `trades.executed` | 1 | Completed trades | Engine | (Future: WebSocket, Analytics) |
| `orderbook.snapshots` | 1 | Periodic snapshots | Engine | (Future: Recovery, Analytics) |

**Note**: Single partition in Phase 4 to maintain order processing sequence. Phase 5+ can add partitioning by symbol.

### API Changes

**Before (Synchronous)**:
```go
func PlaceOrder(w http.ResponseWriter, r *http.Request) {
    order := parseRequest(r)
    trades := engine.PlaceOrder(order)  // Blocking call
    respond(w, OrderResponse{
        Order: order,
        Trades: trades,  // Immediate results
    })
}
```

**After (Asynchronous)**:
```go
func PlaceOrder(w http.ResponseWriter, r *http.Request) {
    order := parseRequest(r)
    err := kafkaProducer.PublishOrder(order)  // Non-blocking
    if err != nil {
        respondError(w, 500, "Failed to accept order")
        return
    }
    respond(w, 202, OrderResponse{
        Order: order,
        Status: "ACCEPTED",  // Not yet executed
        Message: "Order accepted for processing",
    })
}
```

### Response Changes

**New HTTP Status Codes**:
- `202 Accepted` - Order accepted, will be processed asynchronously
- `200 OK` - Query operations (GET orderbook, trades, etc.)
- `400 Bad Request` - Invalid order data
- `500 Internal Server Error` - Kafka publish failed

**New Order Statuses**:
- `ACCEPTED` - Order published to Kafka, awaiting processing
- `PENDING` - Order in engine queue (future enhancement)
- `FILLED` - Order fully executed
- `PARTIALLY_FILLED` - Order partially executed
- `CANCELLED` - Order cancelled
- `REJECTED` - Order rejected by engine

## Implementation Tasks

### Task 1: Kafka Infrastructure Setup

**1.1 Add Kafka Client Dependency**
```bash
go get github.com/IBM/sarama@latest
```

**1.2 Docker Compose for Local Development**

Create `docker-compose.kafka.yml`:
```yaml
version: '3.8'
services:
  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    ports:
      - "2181:2181"

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
```

**1.3 Configuration**

Add to `.env`:
```bash
# Kafka Configuration
KAFKA_ENABLED=false
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=matching-engine-group
KAFKA_ORDERS_TOPIC=orders.incoming
KAFKA_TRADES_TOPIC=trades.executed
KAFKA_ORDERBOOK_TOPIC=orderbook.snapshots
```

Add to `config/config.go`:
```go
type KafkaConfig struct {
    Enabled        bool
    Brokers        []string
    GroupID        string
    OrdersTopic    string
    TradesTopic    string
    OrderBookTopic string
}
```

---

### Task 2: Kafka Client Infrastructure

**2.1 Create Kafka Package**

Create `internal/messaging/kafka/` directory structure:
```
internal/messaging/
├── kafka/
│   ├── producer.go      # Kafka producer wrapper
│   ├── consumer.go      # Kafka consumer wrapper
│   ├── config.go        # Kafka-specific config
│   └── admin.go         # Topic management
└── messages/
    ├── order.go         # Order message schema
    └── trade.go         # Trade message schema
```

**2.2 Producer Interface**

`internal/messaging/kafka/producer.go`:
```go
type Producer struct {
    producer sarama.SyncProducer
    config   *ProducerConfig
}

func NewProducer(brokers []string, config *ProducerConfig) (*Producer, error)
func (p *Producer) PublishOrder(order *types.Order) error
func (p *Producer) Close() error
```

**2.3 Consumer Interface**

`internal/messaging/kafka/consumer.go`:
```go
type Consumer struct {
    consumer      sarama.ConsumerGroup
    config        *ConsumerConfig
    orderHandler  OrderHandler
}

type OrderHandler func(*types.Order) error

func NewConsumer(brokers []string, groupID string, config *ConsumerConfig) (*Consumer, error)
func (c *Consumer) Subscribe(topics []string, handler OrderHandler) error
func (c *Consumer) Close() error
```

---

### Task 3: Modify API Layer

**3.1 Update `cmd/api/server.go`**

Add Kafka producer initialization:
```go
func main() {
    // ... existing code ...

    var kafkaProducer *kafka.Producer
    if cfg.Kafka.Enabled {
        producer, err := kafka.NewProducer(cfg.Kafka.Brokers, &kafka.ProducerConfig{
            OrdersTopic: cfg.Kafka.OrdersTopic,
        })
        if err != nil {
            logger.Fatal("Failed to create Kafka producer", err)
        }
        kafkaProducer = producer
        defer kafkaProducer.Close()
    }

    // Pass producer to handlers
    handlers := api.NewHandlers(engine, orderStore, tradeStore, kafkaProducer)
}
```

**3.2 Update `internal/api/handlers/orders.go`**

Modify `PlaceOrder` handler:
```go
func (h *OrderHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req models.OrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, 400, "Invalid request body")
        return
    }

    // Validate order
    if err := req.Validate(); err != nil {
        respondError(w, 400, err.Error())
        return
    }

    // Create order
    order := toEngineOrder(req)

    // If Kafka enabled, publish asynchronously
    if h.kafkaProducer != nil {
        err := h.kafkaProducer.PublishOrder(order)
        if err != nil {
            logger.Error("Failed to publish order to Kafka", err)
            respondError(w, 500, "Failed to accept order")
            return
        }

        // Return 202 Accepted
        respondJSON(w, 202, models.OrderResponse{
            Order:   toAPIOrder(order),
            Status:  "ACCEPTED",
            Message: "Order accepted for processing",
        })
        return
    }

    // Fallback: synchronous processing (if Kafka disabled)
    trades := h.engine.PlaceOrder(order)
    respondJSON(w, 200, models.OrderResponse{
        Order:  toAPIOrder(order),
        Trades: toAPITrades(trades),
        Status: "FILLED",
    })
}
```

---

### Task 4: Modify Engine to Consume from Kafka

**4.1 Create Engine Consumer**

New file: `internal/matching/kafka_consumer.go`:
```go
type EngineConsumer struct {
    engine   *Engine
    consumer *kafka.Consumer
}

func NewEngineConsumer(engine *Engine, consumer *kafka.Consumer) *EngineConsumer {
    return &EngineConsumer{
        engine:   engine,
        consumer: consumer,
    }
}

func (ec *EngineConsumer) Start(ctx context.Context) error {
    return ec.consumer.Subscribe([]string{"orders.incoming"}, ec.handleOrder)
}

func (ec *EngineConsumer) handleOrder(order *types.Order) error {
    // Process order through engine
    trades := ec.engine.PlaceOrder(order)

    // Publish trades (if Kafka producer available)
    if ec.tradeProducer != nil {
        for _, trade := range trades {
            _ = ec.tradeProducer.PublishTrade(trade)
        }
    }

    return nil
}
```

**4.2 Create Standalone Engine Service**

New file: `cmd/engine/main.go`:
```go
package main

func main() {
    // Load config
    cfg := config.Load()

    // Create storage layers
    orderStore, tradeStore := buildStorageLayers(cfg)

    // Create engine
    engine := matching.NewEngineWithStores(orderStore, tradeStore)

    // Create Kafka consumer
    consumer, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, nil)
    if err != nil {
        log.Fatal("Failed to create Kafka consumer:", err)
    }
    defer consumer.Close()

    // Create engine consumer
    engineConsumer := matching.NewEngineConsumer(engine, consumer)

    // Start consuming
    ctx := context.Background()
    log.Println("Engine started, consuming from Kafka...")
    if err := engineConsumer.Start(ctx); err != nil {
        log.Fatal("Engine consumer failed:", err)
    }
}
```

Now you can run:
- `cmd/api/server.go` - API server (produces orders)
- `cmd/engine/main.go` - Engine consumer (processes orders)

---

### Task 5: Message Schemas

**5.1 Order Message**

`internal/messaging/messages/order.go`:
```go
type OrderMessage struct {
    ID        uint64    `json:"id"`
    UserID    string    `json:"user_id"`
    Symbol    string    `json:"symbol"`
    Side      string    `json:"side"`      // "buy" or "sell"
    Type      string    `json:"type"`      // "market" or "limit"
    Price     float64   `json:"price"`
    Size      int       `json:"size"`
    Timestamp time.Time `json:"timestamp"`
}

func (om *OrderMessage) ToEngineOrder() *types.Order
func FromEngineOrder(order *types.Order) *OrderMessage
```

**5.2 Trade Message**

`internal/messaging/messages/trade.go`:
```go
type TradeMessage struct {
    BuyOrderID  uint64    `json:"buy_order_id"`
    SellOrderID uint64    `json:"sell_order_id"`
    Price       float64   `json:"price"`
    Size        int       `json:"size"`
    Timestamp   time.Time `json:"timestamp"`
}

func FromEngineTrade(trade *types.Trade) *TradeMessage
```

---

### Task 6: Testing

**6.1 Integration Test**

`internal/api/tests/kafka_test.go`:
```go
func TestKafkaOrderFlow(t *testing.T) {
    // 1. Start local Kafka (testcontainers)
    // 2. Start API server with Kafka enabled
    // 3. Start engine consumer
    // 4. POST order to API
    // 5. Assert: 202 Accepted response
    // 6. Wait for engine to process
    // 7. Assert: Trade appears in storage
}
```

**6.2 Manual Testing Script**

Create `scripts/test_kafka.sh`:
```bash
#!/bin/bash

# Start Kafka
docker-compose -f docker-compose.kafka.yml up -d

# Wait for Kafka
sleep 10

# Start engine consumer
go run cmd/engine/main.go &
ENGINE_PID=$!

# Start API server
go run cmd/api/server.go &
API_PID=$!

# Wait for services
sleep 5

# Place order
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "alice",
    "symbol": "AAPL",
    "order_type": 1,
    "side": 0,
    "price": 150.00,
    "size": 100
  }'

# Check trades
sleep 2
curl http://localhost:8080/api/v1/trades?limit=5

# Cleanup
kill $ENGINE_PID $API_PID
docker-compose -f docker-compose.kafka.yml down
```

---

### Task 7: Documentation

**7.1 Update QUICKSTART.md**

Add new section: "Running with Kafka"

**7.2 Create KAFKA.md**

Document:
- Kafka architecture
- Topic schemas
- Consumer group management
- Monitoring with Kafka tools

---

## Implementation Order

1. **Day 1: Infrastructure** (Tasks 1-2)
   - Add dependencies
   - Create Kafka package
   - Docker Compose setup

2. **Day 2: API Changes** (Task 3)
   - Modify API handlers
   - Add producer integration
   - Update response models

3. **Day 3: Engine Consumer** (Task 4)
   - Create engine consumer
   - Create standalone engine service
   - Test order flow

4. **Day 4: Testing & Polish** (Tasks 5-6)
   - Message schemas
   - Integration tests
   - Manual testing

5. **Day 5: Documentation** (Task 7)
   - Update guides
   - Add examples
   - Production considerations

---

## Open Questions

1. **Order ID Generation**: Who generates order IDs?
   - **Option A**: API generates (before Kafka publish)
   - **Option B**: Engine generates (after Kafka consume)
   - **Recommendation**: API generates (better for tracking)

2. **Order Acknowledgment**: How does user know order was processed?
   - **Option A**: Poll GET /orders/{id} endpoint
   - **Option B**: WebSocket feed (Phase 5)
   - **Recommendation**: Start with polling, add WebSocket in Phase 5

3. **Failed Orders**: What if engine rejects an order?
   - **Option A**: Publish to `orders.rejected` topic
   - **Option B**: Update order status in database
   - **Recommendation**: Both (publish + database)

4. **Kafka Availability**: What if Kafka is down?
   - **Option A**: Fail fast (return 503)
   - **Option B**: Fallback to sync processing
   - **Recommendation**: Fallback for development, fail fast for production

---

## Phase 5 Preview (Future)

After Phase 4 is complete:
- **WebSocket feeds** for real-time order/trade updates
- **Multi-partition** Kafka topics (partition by symbol)
- **Multiple engine instances** (horizontal scaling)
- **Dead letter queue** for failed orders
- **Kafka Streams** for analytics

---

## Success Criteria

Phase 4 is complete when:

- ✅ API publishes orders to Kafka
- ✅ API returns 202 Accepted (not 200)
- ✅ Engine consumes from Kafka and processes orders
- ✅ Trades are published to Kafka
- ✅ Storage layers still work (Redis, Postgres)
- ✅ Integration tests pass
- ✅ Documentation updated
- ✅ Can run API and Engine as separate processes
