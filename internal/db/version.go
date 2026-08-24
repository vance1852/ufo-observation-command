package db

import (
	"context"
	"fmt"
)

func CurrentMigration(ctx context.Context, pool *Pool) (int, error) {
	var version int
	if err := pool.QueryRow(ctx, `SELECT COALESCE(max(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("current migration: %w", err)
	}
	return version, nil
}

func RequireMigration(ctx context.Context, pool *Pool, expected int) error {
	version, err := CurrentMigration(ctx, pool)
	if err != nil {
		return err
	}
	if version != expected {
		return fmt.Errorf("migration version %d does not match expected %d", version, expected)
	}
	return nil
}
