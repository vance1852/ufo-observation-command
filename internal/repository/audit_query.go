package repository

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/audit"
)

func (p *Postgres) QueryAudit(ctx context.Context, filter audit.Filter, from, to time.Time, limit, offset int) ([]audit.Event, int, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	where := "WHERE created_at >= $1 AND created_at < $2"
	args := []any{from, to, limit, offset}
	countWhere := "WHERE created_at >= $1 AND created_at < $2"
	countArgs := []any{from, to}
	if filter.ObjectType != "" {
		args = append(args, filter.ObjectType)
		where += fmt.Sprintf(" AND object_type=$%d", len(args))
		countArgs = append(countArgs, filter.ObjectType)
		countWhere += fmt.Sprintf(" AND object_type=$%d", len(countArgs))
	}
	if filter.ObjectID != "" {
		args = append(args, filter.ObjectID)
		where += fmt.Sprintf(" AND object_id=$%d", len(args))
		countArgs = append(countArgs, filter.ObjectID)
		countWhere += fmt.Sprintf(" AND object_id=$%d", len(countArgs))
	}
	if filter.Action != "" {
		args = append(args, filter.Action)
		where += fmt.Sprintf(" AND action=$%d", len(args))
		countArgs = append(countArgs, filter.Action)
		countWhere += fmt.Sprintf(" AND action=$%d", len(countArgs))
	}
	if filter.Outcome != "" {
		args = append(args, filter.Outcome)
		where += fmt.Sprintf(" AND outcome=$%d", len(args))
		countArgs = append(countArgs, filter.Outcome)
		countWhere += fmt.Sprintf(" AND outcome=$%d", len(countArgs))
	}
	rows, err := p.pool.Query(ctx, fmt.Sprintf(`SELECT request_id,array_operator_id,object_type,object_id,action,outcome,detail,created_at FROM audit_events %s ORDER BY created_at DESC LIMIT $3 OFFSET $4`, where), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit: %w", err)
	}
	defer rows.Close()
	items := make([]audit.Event, 0)
	for rows.Next() {
		var item audit.Event
		if err := rows.Scan(&item.RequestID, &item.ArrayOperatorID, &item.ObjectType, &item.ObjectID, &item.Action, &item.Outcome, &item.Detail, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := p.pool.QueryRow(ctx, "SELECT count(*) FROM audit_events "+countWhere, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}
	return items, total, nil
}
