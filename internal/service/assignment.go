package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (s *Service) AssignAcousticBuoy(ctx context.Context, meta RequestMeta, assignment domain.Assignment, array_operator domain.ArrayOperator) error {
	if err := domain.CanAssign(array_operator, assignment); err != nil {
		return err
	}
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		repo, ok := tx.(interface {
			CreateAssignment(context.Context, domain.Assignment) error
			ValidateSurveyMissionAcousticBuoy(context.Context, string, string) error
		})
		if !ok {
			return fmt.Errorf("assignment repository unavailable")
		}
		if err := repo.ValidateSurveyMissionAcousticBuoy(ctx, assignment.SurveyMissionID, assignment.AcousticBuoyID); err != nil {
			return err
		}
		if err := repo.CreateAssignment(ctx, assignment); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "assignment", assignment.ID, "create", "success", nil))
	})
}

func (s *Service) AdvanceAssignment(ctx context.Context, meta RequestMeta, id, next string, version int64) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		repo, ok := tx.(interface {
			AdvanceAssignment(context.Context, string, string, int64) error
		})
		if !ok {
			return fmt.Errorf("assignment repository unavailable")
		}
		if err := repo.AdvanceAssignment(ctx, id, next, version); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "assignment", id, next, "success", nil))
	})
}
