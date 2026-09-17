package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InitDBPool initializes a PostgreSQL connection pool using pgxpool.
func InitDBPool(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// WithTenantContext executes database queries within an isolated transaction where PostgreSQL RLS is enforced
// by running `SET LOCAL app.current_org_id = $1`.
func WithTenantContext(ctx context.Context, pool *pgxpool.Pool, orgID string, fn func(tx pgx.Tx) error) error {
	if orgID == "" {
		return fmt.Errorf("organization ID cannot be empty for tenant-isolated database operations")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tenant transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	// Set tenant context for PostgreSQL Kernel Row-Level Security (RLS)
	// Use parameterized query to prevent SQL injection (pgx/v5 supports parameterized SET LOCAL)
	_, err = tx.Exec(ctx, "SET LOCAL app.current_org_id = $1", orgID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("failed to set tenant RLS context: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tenant transaction: %w", err)
	}

	return nil
}
