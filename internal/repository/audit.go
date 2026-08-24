package repository

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) AuditHistory(ctx context.Context, objectType, objectID string, limit int) ([]domain.AuditSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := p.pool.Query(ctx, `SELECT object_type,object_id,action,outcome,created_at FROM audit_events WHERE object_type=$1 AND object_id=$2 ORDER BY created_at DESC LIMIT $3`, objectType, objectID, limit)
	if err != nil {
		return nil, fmt.Errorf("audit history: %w", err)
	}
	defer rows.Close()
	out := make([]domain.AuditSummary, 0)
	for rows.Next() {
		var item domain.AuditSummary
		if err := rows.Scan(&item.ObjectType, &item.ObjectID, &item.Action, &item.Outcome, &item.At); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func auditTime(now time.Time) time.Time { return now.UTC() }
