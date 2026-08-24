package repository

import (
	"context"
	"fmt"
	"time"
)

func (p *Postgres) ActivateDue(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	result, err := p.pool.Exec(ctx, `UPDATE assignments SET status='active',version=version+1 WHERE id IN (SELECT id FROM assignments WHERE status='queued' AND starts_at <= $1 AND ends_at > $1 ORDER BY starts_at LIMIT $2 FOR UPDATE SKIP LOCKED)`, now, limit)
	if err != nil {
		return 0, fmt.Errorf("activate due assignments: %w", err)
	}
	return int(result.RowsAffected()), nil
}
