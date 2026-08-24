package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (s *Service) CreateRecoveryJobsBulk(ctx context.Context, meta RequestMeta, requests []domain.RecoveryJobRequest) ([]domain.BulkItemResult, error) {
	now := s.now()
	if len(requests) == 0 {
		return nil, fmt.Errorf("at least one task is required: %w", domain.ErrConflict)
	}
	if err := domain.ValidateBulkRequests(requests, now); err != nil {
		return nil, err
	}
	inputs := make([]repository.RecoveryJobInput, len(requests))
	for i, request := range requests {
		inputs[i] = repository.RecoveryJobInput{SurveyMissionID: request.SurveyMissionID, AcousticBuoyID: request.AcousticBuoyID, TaskCode: request.TaskCode, ExpiresAt: request.ExpiresAt}
	}
	var recovery_jobs []domain.RecoveryJob
	err := s.repo.InTx(ctx, func(tx repository.Repository) error {
		for _, input := range inputs {
			placement, ok := tx.(interface {
				ValidateSurveyMissionAcousticBuoy(context.Context, string, string) error
			})
			if !ok {
				return fmt.Errorf("task placement repository unavailable")
			}
			if err := placement.ValidateSurveyMissionAcousticBuoy(ctx, input.SurveyMissionID, input.AcousticBuoyID); err != nil {
				return err
			}
			task, err := tx.CreateRecoveryJob(ctx, input)
			if err != nil {
				return err
			}
			recovery_jobs = append(recovery_jobs, task)
		}
		return tx.WriteAudit(ctx, audit(meta, "task_dive_window", requests[0].SurveyMissionID, "create_bulk", "success", nil))
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.BulkItemResult, len(recovery_jobs))
	for i, task := range recovery_jobs {
		result[i] = domain.BulkItemResult{Index: i, TaskCode: task.TaskCode, RecoveryJobID: task.ID}
	}
	return result, nil
}

func (s *Service) ValidateBulkForAcousticBuoy(requests []domain.RecoveryJobRequest, acoustic_buoyID string) error {
	if err := domain.ValidateBulkRequests(requests, s.now()); err != nil {
		return err
	}
	for _, request := range requests {
		if request.AcousticBuoyID != acoustic_buoyID {
			return fmt.Errorf("task acoustic_buoy differs from dive_window acoustic_buoy: %w", domain.ErrConflict)
		}
	}
	return nil
}
