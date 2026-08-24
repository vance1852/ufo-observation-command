package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) SurveyMissionProgress(ctx context.Context, survey_missionID string) (domain.SurveyMissionProgress, error) {
	return survey_missionProgress(ctx, p.pool, survey_missionID)
}

func survey_missionProgress(ctx context.Context, q sqler, survey_missionID string) (domain.SurveyMissionProgress, error) {
	var progress domain.SurveyMissionProgress
	progress.SurveyMissionID = survey_missionID
	err := q.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM acoustic_buoys WHERE survey_mission_id=$1),
		COALESCE((SELECT sum(required_successes) FROM acoustic_buoys WHERE survey_mission_id=$1),0),
		count(*) FILTER (WHERE status='completed'),
		count(*) FILTER (WHERE status='accepted'),
		count(*) FILTER (WHERE status='in_progress'),
		count(*) FILTER (WHERE status='verified'),
		count(*) FILTER (WHERE status='rejected'),
		count(*) FILTER (WHERE status='archived')
		FROM recovery_jobs WHERE survey_mission_id=$1`, survey_missionID).Scan(&progress.AcousticBuoys, &progress.Required, &progress.Completed, &progress.Accepted, &progress.InProgress, &progress.Verified, &progress.Rejected, &progress.Archived)
	if err != nil {
		return domain.SurveyMissionProgress{}, fmt.Errorf("survey_mission progress: %w", err)
	}
	return progress, nil
}
