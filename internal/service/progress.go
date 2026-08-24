package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (s *Service) SurveyMissionProgress(ctx context.Context, survey_missionID string) (domain.SurveyMissionProgress, error) {
	if _, err := s.repo.GetSurveyMission(ctx, survey_missionID); err != nil {
		return domain.SurveyMissionProgress{}, err
	}
	repo, ok := s.repo.(interface {
		SurveyMissionProgress(context.Context, string) (domain.SurveyMissionProgress, error)
	})
	if !ok {
		return domain.SurveyMissionProgress{}, fmt.Errorf("progress repository unavailable")
	}
	return repo.SurveyMissionProgress(ctx, survey_missionID)
}

func (s *Service) AuditHistory(ctx context.Context, objectType, objectID string, limit int) ([]domain.AuditSummary, error) {
	repo, ok := s.repo.(interface {
		AuditHistory(context.Context, string, string, int) ([]domain.AuditSummary, error)
	})
	if !ok {
		return nil, fmt.Errorf("audit repository unavailable")
	}
	return repo.AuditHistory(ctx, objectType, objectID, limit)
}
