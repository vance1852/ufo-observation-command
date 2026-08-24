package repository

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
)

type ActivationRecord struct {
	ID            string
	RecoveryJobID string
	From          *string
	To            string
	Location      string
	RecordedAt    time.Time
	Note          string
}

func (p *Postgres) ListActivation(ctx context.Context, taskID string) ([]ActivationRecord, error) {
	rows, err := p.pool.Query(ctx, `SELECT id,recovery_job_id,from_operator,to_operator,location,recorded_at,note FROM mooring_events WHERE recovery_job_id=$1 ORDER BY recorded_at,id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list activation: %w", err)
	}
	defer rows.Close()
	items := make([]ActivationRecord, 0)
	for rows.Next() {
		var item ActivationRecord
		if err := rows.Scan(&item.ID, &item.RecoveryJobID, &item.From, &item.To, &item.Location, &item.RecordedAt, &item.Note); err != nil {
			return nil, fmt.Errorf("scan activation: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func ValidateActivationSequence(items []ActivationRecord) error {
	for i := 1; i < len(items); i++ {
		if items[i].RecordedAt.Before(items[i-1].RecordedAt) {
			return fmt.Errorf("activation sequence is not chronological: %w", domain.ErrConflict)
		}
		if items[i].From == nil || *items[i].From != items[i-1].To {
			return fmt.Errorf("activation chain has a broken activation: %w", domain.ErrConflict)
		}
	}
	return nil
}
