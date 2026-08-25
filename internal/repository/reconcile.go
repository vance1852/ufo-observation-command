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
}

// processableStatuses are the recovery_job stages that still belong in the
// pending/processable queue. Expiration only ever clears tasks out of these
// stages; anything outside this set is already out of the queue.
var processableStatuses = []string{"queued", "completed", "activation_pending", "accepted", "in_progress"}

// MarkExpiredRecoveryJobs expires recovery_jobs whose deadline has elapsed.
//
// Each task is moved out of the processable queue inside its own transaction so
// that a single rejected write (the "storage rejects the write" failure) only
// stalls that one task: its state change and its audit row commit together or
// roll back together. When a task's cleanup fails it stays in its current
// processable stage and is counted in Failed; the remaining tasks in the batch
// are still reconciled. This keeps expired cleanup from advancing a task's
// lifecycle while leaving it absent from the pending queue.
func (p *Postgres) MarkExpiredRecoveryJobs(ctx context.Context, now time.Time, limit int) (ReconcileResult, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if p == nil || p.pool == nil {
		return ReconcileResult{}, fmt.Errorf("expired task reconciliation repository is nil")
	}
	rows, err := p.pool.Query(ctx, `SELECT id FROM recovery_jobs WHERE status = ANY($1) AND expires_at < $2 ORDER BY expires_at LIMIT $3 FOR UPDATE SKIP LOCKED`, processableStatuses, now, limit)
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("select expired recovery_jobs: %w", err)
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return ReconcileResult{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return ReconcileResult{}, err
	}
	rows.Close()

	result := ReconcileResult{Scanned: len(ids)}
	for _, id := range ids {
		marked, err := p.expireRecoveryJob(ctx, id)
		if err != nil || !marked {
			// Cleanup failed (rejected write) or the task already left the
			// processable queue: either way it stays where it is.
			result.Failed++
			continue
		}
		result.Marked++
	}
	return result, nil
}

// expireRecoveryJob moves a single expired recovery_job out of the processable
// queue and records the expire audit in the same transaction. It returns
// (true, nil) when the task was advanced to rejected, (false, nil) when the
// task was no longer in a processable stage (race lost/already moved), and
// (false, err) when the state change or audit write was rejected — in which
// case the transaction rolls back and the task remains in its processable stage.
func (p *Postgres) expireRecoveryJob(ctx context.Context, id string) (bool, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin expired task reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	updated, err := tx.Exec(ctx, `UPDATE recovery_jobs SET status='rejected',version=version+1 WHERE id=$1 AND status = ANY($2)`, id, processableStatuses)
	if err != nil {
		return false, fmt.Errorf("mark task %s expired: %w", id, err)
	}
	if updated.RowsAffected() != 1 {
		// The task already left the processable queue between selection and update.
		// Nothing to expire and nothing to audit; leave it where it is.
		return false, tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events(id,request_id,object_type,object_id,action,outcome,detail) VALUES ($1,$2,'task',$3,'expire','success','{}'::jsonb)`, uuid.NewString(), "worker:task-expiration", id); err != nil {
		// Storage rejected the audit write. Roll back so the task keeps its
		// current processable status instead of advancing its lifecycle while
		// leaving the pending queue with no durable audit trail.
		return false, fmt.Errorf("audit expired task %s: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit expired task reconciliation: %w", err)
	}
	return true, nil
}
