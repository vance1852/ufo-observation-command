package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ReconcileResult struct {
	Scanned int
	Marked  int
	Failed  int
	// Skipped counts scanned candidates that were not confirmed as execution
	// results because a concurrent site (e.g. a delayed replica catching up)
	// transitioned them first. Skipped candidates are neither successes nor
	// failures: separating them keeps cross-site aggregation from being
	// distorted and prevents prematurely consuming capacity.
	Skipped int
}

func (p *Postgres) MarkExpiredRecoveryJobs(ctx context.Context, now time.Time, limit int) (ReconcileResult, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("begin expired task reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT id FROM recovery_jobs WHERE status IN ('queued','completed','activation_pending','accepted','in_progress') AND expires_at < $1 ORDER BY expires_at LIMIT $2 FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("select expired recovery_jobs: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ReconcileResult{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return ReconcileResult{}, err
	}
	rows.Close()
	result := ReconcileResult{Scanned: len(ids)}
	for _, id := range ids {
		updated, err := tx.Exec(ctx, `UPDATE recovery_jobs SET status='rejected',version=version+1 WHERE id=$1 AND status IN ('queued','completed','activation_pending','accepted','in_progress')`, id)
		if err != nil {
			return ReconcileResult{}, fmt.Errorf("mark task %s expired: %w", id, err)
		}
		if updated.RowsAffected() != 1 {
			// The candidate was scanned but could not be confirmed as an
			// execution result: a concurrent site (for example a delayed
			// replica catching up) transitioned it first. Count it as skipped,
			// not failed, so cross-site aggregation is not distorted and the
			// capacity metric is not prematurely consumed.
			result.Skipped++
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO audit_events(id,request_id,object_type,object_id,action,outcome,detail) VALUES ($1,$2,'task',$3,'expire','success','{}'::jsonb)`, uuid.NewString(), "worker:task-expiration", id); err != nil {
			return ReconcileResult{}, fmt.Errorf("audit expired task %s: %w", id, err)
		}
		result.Marked++
	}
	if err := tx.Commit(ctx); err != nil {
		return ReconcileResult{}, fmt.Errorf("commit expired task reconciliation: %w", err)
	}
	return result, nil
}
