package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type TxOptions struct {
	Isolation  pgx.TxIsoLevel
	AccessMode pgx.TxAccessMode
}

func (p *Postgres) InSerializable(ctx context.Context, fn func(Repository) error) error {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return fmt.Errorf("begin serializable transaction: %w", err)
	}
	if err := fn(&transaction{tx: tx}); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit serializable transaction: %w", err)
	}
	return nil
}

func WithReadOnly(ctx context.Context, p *Postgres, fn func(context.Context, Repository) error) error {
	return p.InTx(ctx, func(repo Repository) error { return fn(ctx, repo) })
}
