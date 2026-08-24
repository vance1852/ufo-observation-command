package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) CloseIntegrityIncident(ctx context.Context, id string, now time.Time) error {
	return closeIntegrityIncident(ctx, p.pool, id, now)
}
func (t *transaction) CloseIntegrityIncident(ctx context.Context, id string, now time.Time) error {
	return closeIntegrityIncident(ctx, t.tx, id, now)
}

func (p *Postgres) GetIntegrityIncident(ctx context.Context, id string) (domain.IntegrityIncident, error) {
	var d domain.IntegrityIncident
	err := p.pool.QueryRow(ctx, `SELECT id,recovery_job_id,kind,status,reason,due_at,closed_at FROM integrity_incidents WHERE id=$1`, id).Scan(&d.ID, &d.RecoveryJobID, &d.Kind, &d.Status, &d.Reason, &d.DueAt, &d.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.IntegrityIncident{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.IntegrityIncident{}, fmt.Errorf("get safety_alert: %w", err)
	}
	return d, nil
}

func closeIntegrityIncident(ctx context.Context, q sqler, id string, now time.Time) error {
	result, err := q.Exec(ctx, `UPDATE integrity_incidents SET status='closed',closed_at=$1 WHERE id=$2 AND status IN ('open','in_progress')`, now, id)
	if err != nil {
		return fmt.Errorf("close safety_alert: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func (p *Postgres) MarkIntegrityIncidentInProgress(ctx context.Context, id string) error {
	return markIntegrityIncidentInProgress(ctx, p.pool, id)
}

func (t *transaction) MarkIntegrityIncidentInProgress(ctx context.Context, id string) error {
	return markIntegrityIncidentInProgress(ctx, t.tx, id)
}

func markIntegrityIncidentInProgress(ctx context.Context, q sqler, id string) error {
	result, err := q.Exec(ctx, `UPDATE integrity_incidents SET status='in_progress' WHERE id=$1 AND status='open'`, id)
	if err != nil {
		return fmt.Errorf("mark safety_alert in progress: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}
