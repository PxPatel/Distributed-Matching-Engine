package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 001_initial_schema.sql
var initialSchema string

// RunMigrations executes all database migrations
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Simple migration runner - in production, use a proper migration tool
	// like golang-migrate, but this works for our initial schema
	_, err := pool.Exec(ctx, initialSchema)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
