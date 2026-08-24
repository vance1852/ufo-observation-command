package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (s *Service) ListSurveyMissions(ctx context.Context, filter repository.SurveyMissionFilter) ([]domain.SurveyMission, int, error) {
	repo, ok := s.repo.(interface {
		ListSurveyMissions(context.Context, repository.SurveyMissionFilter) ([]domain.SurveyMission, int, error)
	})
	if !ok {
		return nil, 0, fmt.Errorf("survey_mission query repository unavailable")
	}
	items, total, err := repo.ListSurveyMissions(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return append([]domain.SurveyMission(nil), items...), total, nil
}

func (s *Service) ListSurveyMissionAcousticBuoys(ctx context.Context, survey_missionID string) ([]domain.AcousticBuoy, error) {
	repo, ok := s.repo.(interface {
		ListSurveyMissionAcousticBuoys(context.Context, string) ([]domain.AcousticBuoy, error)
	})
	if !ok {
		return nil, fmt.Errorf("acoustic_buoy query repository unavailable")
	}
	items, err := repo.ListSurveyMissionAcousticBuoys(ctx, survey_missionID)
	if err != nil {
		return nil, err
	}
	return append([]domain.AcousticBuoy(nil), items...), nil
}
