package repository

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) OpenIntegrityIncidentCount(ctx context.Context, survey_missionID string) (int, error) {
	var count int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM integrity_incidents d JOIN recovery_jobs s ON s.id=d.recovery_job_id WHERE s.survey_mission_id=$1 AND d.status IN ('open','in_progress')`, survey_missionID).Scan(&count); err != nil {
		return 0, fmt.Errorf("open safety_alert count: %w", err)
	}
	return count, nil
}

func (p *Postgres) ComplianceReport(ctx context.Context, survey_missionID string, now time.Time) (domain.ComplianceReport, error) {
	progress, err := p.SurveyMissionProgress(ctx, survey_missionID)
	if err != nil {
		return domain.ComplianceReport{}, err
	}
	expiring, err := p.ExpiringRecoveryJobsForSurveyMission(ctx, survey_missionID, now.Add(48*time.Hour), 100)
	if err != nil {
		return domain.ComplianceReport{}, err
	}
	count, err := p.OpenIntegrityIncidentCount(ctx, survey_missionID)
	if err != nil {
		return domain.ComplianceReport{}, err
	}
	return domain.ComplianceReport{SurveyMissionID: survey_missionID, GeneratedAt: now.UTC(), Progress: progress, Expiring: expiring, OpenIntegrityIncidents: count}, nil
}
