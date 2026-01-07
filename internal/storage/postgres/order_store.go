package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PxPatel/trading-system/internal/types"
)

// PostgresOrderStore implements OrderStore using PostgreSQL
type PostgresOrderStore struct {
	pool *pgxpool.Pool
}

// NewPostgresOrderStore creates a new PostgreSQL-backed order store
func NewPostgresOrderStore(cfg PostgresConfig) (*PostgresOrderStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return &PostgresOrderStore{pool: pool}, nil
}

func (s *PostgresOrderStore) Save(order *types.Order) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO orders (order_id, user_id, symbol, order_type, side, price, stop_price, size, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (order_id) DO UPDATE SET
			size = EXCLUDED.size,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.pool.Exec(ctx, query,
		order.ID, order.UserID, order.Symbol, order.OrderType, order.Side,
		order.Price, order.StopPrice, order.Size, order.TimeStamp, time.Now(),
	)

	return err
}

func (s *PostgresOrderStore) Get(orderID uint64) (*types.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT order_id, user_id, symbol, order_type, side, price, stop_price, size, created_at
		FROM orders
		WHERE order_id = $1
	`

	var order types.Order
	err := s.pool.QueryRow(ctx, query, orderID).Scan(
		&order.ID, &order.UserID, &order.Symbol, &order.OrderType, &order.Side,
		&order.Price, &order.StopPrice, &order.Size, &order.TimeStamp,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("order %d not found", orderID)
	}
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (s *PostgresOrderStore) Remove(orderID uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `DELETE FROM orders WHERE order_id = $1`
	_, err := s.pool.Exec(ctx, query, orderID)
	return err
}

func (s *PostgresOrderStore) Update(order *types.Order) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE orders
		SET size = $2, price = $3, updated_at = $4
		WHERE order_id = $1
	`

	result, err := s.pool.Exec(ctx, query, order.ID, order.Size, order.Price, time.Now())
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("order %d not found", order.ID)
	}

	return nil
}

func (s *PostgresOrderStore) GetAll() []*types.Order {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT order_id, user_id, symbol, order_type, side, price, stop_price, size, created_at
		FROM orders
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return []*types.Order{}
	}
	defer rows.Close()

	return s.scanOrders(rows)
}

func (s *PostgresOrderStore) GetByUser(userID string) []*types.Order {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT order_id, user_id, symbol, order_type, side, price, stop_price, size, created_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return []*types.Order{}
	}
	defer rows.Close()

	return s.scanOrders(rows)
}

func (s *PostgresOrderStore) GetBySide(side types.SideType) []*types.Order {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT order_id, user_id, symbol, order_type, side, price, stop_price, size, created_at
		FROM orders
		WHERE side = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, side)
	if err != nil {
		return []*types.Order{}
	}
	defer rows.Close()

	return s.scanOrders(rows)
}

func (s *PostgresOrderStore) Close() error {
	s.pool.Close()
	return nil
}

// scanOrders is a helper to scan multiple order rows
func (s *PostgresOrderStore) scanOrders(rows pgx.Rows) []*types.Order {
	var orders []*types.Order

	for rows.Next() {
		var order types.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.Symbol, &order.OrderType, &order.Side,
			&order.Price, &order.StopPrice, &order.Size, &order.TimeStamp,
		)
		if err != nil {
			continue
		}
		orders = append(orders, &order)
	}

	return orders
}
