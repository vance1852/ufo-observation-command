package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) ChangeArrayOperatorRole(ctx context.Context, id string, role domain.ArrayOperatorRole) error {
	if err := (domain.ArrayOperator{ID: id, Name: "valid", Role: role}).Validate(); err != nil {
		return err
	}
	result, err := p.pool.Exec(ctx, `UPDATE array_operators SET role=$1 WHERE id=$2`, role, id)
	if err != nil {
		return fmt.Errorf("change array_operator role: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func (p *Postgres) ArrayOperatorsForRole(ctx context.Context, role domain.ArrayOperatorRole) ([]domain.ArrayOperator, error) {
	rows, err := p.pool.Query(ctx, `SELECT id,name,role FROM array_operators WHERE role=$1 ORDER BY name`, role)
	if err != nil {
		return nil, fmt.Errorf("array_operators for role: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ArrayOperator, 0)
	for rows.Next() {
		var item domain.ArrayOperator
		if err := rows.Scan(&item.ID, &item.Name, &item.Role); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
