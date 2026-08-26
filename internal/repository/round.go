package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) StartDiveWindow(ctx context.Context, id string, version int64) error {
	return changeDiveWindow(ctx, p.pool, id, domain.DiveWindowRunning, version)
}
func (p *Postgres) CompleteDiveWindow(ctx context.Context, id string, version int64) error {
	return changeDiveWindow(ctx, p.pool, id, domain.DiveWindowCompleted, version)
}
func (p *Postgres) CancelDiveWindow(ctx context.Context, id string, version int64) error {
	return changeDiveWindow(ctx, p.pool, id, domain.DiveWindowCancelled, version)
}
func (t *transaction) StartDiveWindow(ctx context.Context, id string, version int64) error {
	return changeDiveWindow(ctx, t.tx, id, domain.DiveWindowRunning, version)
}
func (t *transaction) CompleteDiveWindow(ctx context.Context, id string, version int64) error {
	return changeDiveWindow(ctx, t.tx, id, domain.DiveWindowCompleted, version)
}
func (t *transaction) CancelDiveWindow(ctx context.Context, id string, version int64) error {
	return changeDiveWindow(ctx, t.tx, id, domain.DiveWindowCancelled, version)
}

func changeDiveWindow(ctx context.Context, q sqler, id string, status domain.DiveWindowStatus, version int64) error {
	if status == domain.DiveWindowCancelled {
		var changed int
		err := q.QueryRow(ctx, `WITH changed AS (
			UPDATE dive_windows SET status='cancelled',version=version+1,completed_at=now()
			WHERE id=$1 AND version=$2 AND status IN ('queued','running')
			AND NOT EXISTS (SELECT 1 FROM signal_recovery_reports WHERE dive_window_id=$1)
			RETURNING id
		), restored AS (
			UPDATE recovery_jobs s SET status='accepted',version=version+1
			FROM dive_window_items bs, changed
			WHERE bs.dive_window_id=changed.id AND s.id=bs.recovery_job_id AND s.status='in_progress'
			RETURNING s.id
		)
		SELECT count(*) FROM changed`, id, version).Scan(&changed)
		if err != nil {
			return fmt.Errorf("cancel dive_window: %w", err)
		}
		if changed != 1 {
			return domain.ErrConflict
		}
		return nil
	}
	allowedFrom := []string{}
	switch status {
	case domain.DiveWindowRunning:
		allowedFrom = []string{string(domain.DiveWindowQueued)}
	case domain.DiveWindowCompleted:
		allowedFrom = []string{string(domain.DiveWindowRunning)}
	default:
		return domain.ErrInvalidTransition
	}
	result, err := q.Exec(ctx, `UPDATE dive_windows SET status=$1,version=version+1,started_at=CASE WHEN $1='running' THEN now() ELSE started_at END,completed_at=CASE WHEN $1='completed' THEN now() ELSE completed_at END
		WHERE id=$2 AND version=$3 AND status=ANY($4)
		AND ($1 <> 'completed' OR NOT EXISTS (
			SELECT 1 FROM dive_window_items bs
			LEFT JOIN signal_recovery_reports r ON r.dive_window_id=bs.dive_window_id AND r.recovery_job_id=bs.recovery_job_id
			WHERE bs.dive_window_id=$2 AND (r.id IS NULL OR r.status='pending')
		))`, status, id, version, allowedFrom)
	if err != nil {
		return fmt.Errorf("change dive_window: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}
