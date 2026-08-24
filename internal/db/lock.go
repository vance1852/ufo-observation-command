package db

import (
	"context"
	"fmt"
	"time"
)

func AdvisoryLock(ctx context.Context, pool *Pool, key int64, fn func() error) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire advisory lock connection: %w", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, key); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, key)
	}()
	return fn()
}

func WaitForDatabase(ctx context.Context, pool *Pool, attempts int, delay time.Duration) error {
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		if err := pool.Ping(ctx); err == nil {
			return nil
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return fmt.Errorf("database did not become ready")
}
