package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) CreateRecoveryJobsBulk(ctx context.Context, inputs []RecoveryJobInput) ([]domain.RecoveryJob, error) {
	var recovery_jobs []domain.RecoveryJob
	err := p.InTx(ctx, func(tx Repository) error {
		for _, input := range inputs {
			task, err := tx.CreateRecoveryJob(ctx, input)
			if err != nil {
				return fmt.Errorf("create task %s: %w", input.TaskCode, err)
			}
			recovery_jobs = append(recovery_jobs, task)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return append([]domain.RecoveryJob(nil), recovery_jobs...), nil
}

func ValidateBulkCapacity(inputs []RecoveryJobInput, capacity int) error {
	if capacity < 1 || len(inputs) > capacity {
		return domain.ErrCapacityExceeded
	}
	return nil
}
