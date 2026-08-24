package repository

import (
	"context"
	"errors"
	"fmt"

	"ufo-observation-command/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) CreateArrayOperator(ctx context.Context, array_operator domain.ArrayOperator) error {
	return createArrayOperator(ctx, p.pool, array_operator)
}
func (t *transaction) CreateArrayOperator(ctx context.Context, array_operator domain.ArrayOperator) error {
	return createArrayOperator(ctx, t.tx, array_operator)
}

func (p *Postgres) GetArrayOperator(ctx context.Context, id string) (domain.ArrayOperator, error) {
	return getArrayOperator(ctx, p.pool, id)
}
func (t *transaction) GetArrayOperator(ctx context.Context, id string) (domain.ArrayOperator, error) {
	return getArrayOperator(ctx, t.tx, id)
}

func createArrayOperator(ctx context.Context, q sqler, array_operator domain.ArrayOperator) error {
	if err := array_operator.Validate(); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `INSERT INTO array_operators(id,name,role) VALUES ($1,$2,$3)`, array_operator.ID, array_operator.Name, array_operator.Role)
	return wrapWrite(err)
}

func getArrayOperator(ctx context.Context, q sqler, id string) (domain.ArrayOperator, error) {
	var array_operator domain.ArrayOperator
	err := q.QueryRow(ctx, `SELECT id,name,role FROM array_operators WHERE id=$1`, id).Scan(&array_operator.ID, &array_operator.Name, &array_operator.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ArrayOperator{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ArrayOperator{}, fmt.Errorf("get array_operator: %w", err)
	}
	return array_operator, nil
}
