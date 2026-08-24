package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (s *Service) Authorize(ctx context.Context, array_operatorID, action string) error {
	array_operator, err := s.LoadArrayOperator(ctx, array_operatorID)
	if err != nil {
		return err
	}
	if !array_operator.Can(action) {
		return fmt.Errorf("array_operator cannot %s: %w", action, domain.ErrConflict)
	}
	return nil
}

func RequireSupervisor(ctx context.Context, s *Service, array_operatorID string) error {
	return s.Authorize(ctx, array_operatorID, "close_survey_mission")
}

func RequireReviewer(ctx context.Context, s *Service, array_operatorID string) error {
	return s.Authorize(ctx, array_operatorID, "review_signal_recovery_report")
}
