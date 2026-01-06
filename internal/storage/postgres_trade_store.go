package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PxPatel/trading-system/internal/storage/migrations"
	"github.com/PxPatel/trading-system/internal/types"
)

// PostgresTradeStore implements TradeStore using PostgreSQL
type PostgresTradeStore struct {
	pool *pgxpool.Pool
}

// NewPostgresTradeStore creates a new PostgreSQL-backed trade store
func NewPostgresTradeStore(cfg PostgresConfig) (*PostgresTradeStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := migrations.RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return &PostgresTradeStore{pool: pool}, nil
}

func (s *PostgresTradeStore) Save(trade *types.Trade) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO trades (buy_order_id, sell_order_id, price, quantity, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.pool.Exec(ctx, query,
		trade.BuyOrderID, trade.SellOrderID, trade.Price, trade.Size, trade.Timestamp,
	)

	return err
}

func (s *PostgresTradeStore) SaveBatch(trades []*types.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use pgx batch for efficient batch inserts
	batch := &pgx.Batch{}
	query := `
		INSERT INTO trades (buy_order_id, sell_order_id, price, quantity, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, trade := range trades {
		batch.Queue(query, trade.BuyOrderID, trade.SellOrderID, trade.Price, trade.Size, trade.Timestamp)
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	// Execute all batched queries
	for i := 0; i < len(trades); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("batch insert failed at index %d: %w", i, err)
		}
	}

	return nil
}

func (s *PostgresTradeStore) GetRecent(limit int) ([]*types.Trade, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT trade_id, buy_order_id, sell_order_id, price, quantity, timestamp
		FROM trades
		ORDER BY timestamp DESC
		LIMIT $1
	`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*types.Trade
	for rows.Next() {
		var trade types.Trade
		err := rows.Scan(
			&trade.TradeID, &trade.BuyOrderID, &trade.SellOrderID,
			&trade.Price, &trade.Size, &trade.Timestamp,
		)
		if err != nil {
			continue
		}
		trades = append(trades, &trade)
	}

	return trades, nil
}

func (s *PostgresTradeStore) Close() error {
	s.pool.Close()
	return nil
}
