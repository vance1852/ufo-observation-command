package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) ListArrayOperators(ctx context.Context, role domain.ArrayOperatorRole, limit, offset int) ([]domain.ArrayOperator, int, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{limit, offset}
	where := "WHERE TRUE"
	if role != "" {
		args = append(args, role)
		where += fmt.Sprintf(" AND role=$%d", len(args))
	}
	rows, err := p.pool.Query(ctx, fmt.Sprintf(`SELECT id,name,role FROM array_operators %s ORDER BY name LIMIT $1 OFFSET $2`, where), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list array_operators: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ArrayOperator, 0)
	for rows.Next() {
		var item domain.ArrayOperator
		if err := rows.Scan(&item.ID, &item.Name, &item.Role); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countArgs := args[2:]
	countWhere := "WHERE TRUE"
	if role != "" {
		countWhere += " AND role=$1"
	}
	if err := p.pool.QueryRow(ctx, "SELECT count(*) FROM array_operators "+countWhere, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count array_operators: %w", err)
	}
	return items, total, nil
}

func (p *Postgres) RenameArrayOperator(ctx context.Context, id, name string) error {
	return renameArrayOperator(ctx, p.pool, id, name)
}

func (t *transaction) RenameArrayOperator(ctx context.Context, id, name string) error {
	return renameArrayOperator(ctx, t.tx, id, name)
}

func renameArrayOperator(ctx context.Context, q sqler, id, name string) error {
	result, err := q.Exec(ctx, `UPDATE array_operators SET name=$1 WHERE id=$2`, name, id)
	if err != nil {
		return fmt.Errorf("rename array_operator: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrNotFound
	}
	return nil
}
