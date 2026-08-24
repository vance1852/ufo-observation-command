package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type TransactionPolicy struct {
	Timeout       time.Duration
	Serializable  bool
	RetryAttempts int
}

func (p TransactionPolicy) Validate() error {
	if p.Timeout <= 0 {
		return fmt.Errorf("transaction timeout must be positive")
	}
	if p.RetryAttempts < 1 {
		return fmt.Errorf("transaction retry attempts must be positive")
	}
	return nil
}

func (p *Postgres) RunWithPolicy(ctx context.Context, policy TransactionPolicy, fn func(Repository) error) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	for attempt := 1; attempt <= policy.RetryAttempts; attempt++ {
		transactionCtx, cancel := context.WithTimeout(ctx, policy.Timeout)
		var err error
		if policy.Serializable {
			err = p.InSerializable(transactionCtx, fn)
		} else {
			err = p.InTx(transactionCtx, fn)
		}
		cancel()
		if err == nil {
			return nil
		}
		if !retryableTransactionError(err) || attempt == policy.RetryAttempts {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 10 * time.Millisecond):
		}
	}
	return fmt.Errorf("transaction policy exhausted")
}

func retryableTransactionError(err error) bool {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return false
	}
	switch postgresError.Code {
	case "40001", "40P01":
		return true
	default:
		return false
	}
}
