package service

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
)

func (s *Service) ComplianceReport(ctx context.Context, survey_missionID string) (domain.ComplianceReport, error) {
	if _, err := s.repo.GetSurveyMission(ctx, survey_missionID); err != nil {
		return domain.ComplianceReport{}, err
	}
	repo, ok := s.repo.(interface {
		ComplianceReport(context.Context, string, time.Time) (domain.ComplianceReport, error)
	})
	if !ok {
		return domain.ComplianceReport{}, fmt.Errorf("report repository unavailable")
	}
	return repo.ComplianceReport(ctx, survey_missionID, s.now())
}

func (s *Service) PublicRecoveryJob(task domain.RecoveryJob) map[string]any {
	return map[string]any{"id": task.ID, "task_code": domain.RedactTaskCode(task.TaskCode), "status": task.Status, "expires_at": task.ExpiresAt, "version": task.Version}
}
