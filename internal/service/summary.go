package service

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
)

type Summary struct {
	SurveyMission domain.SurveyMission
	Progress      domain.SurveyMissionProgress
	Counts        any
}

func (s *Service) SurveyMissionSummary(ctx context.Context, survey_missionID string) (Summary, error) {
	survey_mission, err := s.repo.GetSurveyMission(ctx, survey_missionID)
	if err != nil {
		return Summary{}, err
	}
	progress, err := s.SurveyMissionProgress(ctx, survey_missionID)
	if err != nil {
		return Summary{}, fmt.Errorf("summary progress: %w", err)
	}
	return Summary{SurveyMission: survey_mission, Progress: progress, Counts: progress}, nil
}

func (s *Service) ExpiringRecoveryJobs(ctx context.Context, beforeUnix int64, limit int) ([]domain.RecoveryJob, error) {
	repo, ok := s.repo.(interface {
		ExpiringRecoveryJobs(context.Context, time.Time, int) ([]domain.RecoveryJob, error)
	})
	if !ok {
		return nil, fmt.Errorf("task query repository unavailable")
	}
	return repo.ExpiringRecoveryJobs(ctx, time.Unix(beforeUnix, 0).UTC(), limit)
}
