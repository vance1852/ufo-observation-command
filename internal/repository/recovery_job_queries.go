package repository

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
)

type RecoveryJobCounts struct {
	Queued     int
	Completed  int
	Transit    int
	Accepted   int
	InProgress int
	Verified   int
	Rejected   int
	Archived   int
}

func (p *Postgres) RecoveryJobCounts(ctx context.Context, survey_missionID string) (RecoveryJobCounts, error) {
	var counts RecoveryJobCounts
	err := p.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status='queued'),count(*) FILTER (WHERE status='completed'),count(*) FILTER (WHERE status='activation_pending'),count(*) FILTER (WHERE status='accepted'),count(*) FILTER (WHERE status='in_progress'),count(*) FILTER (WHERE status='verified'),count(*) FILTER (WHERE status='rejected'),count(*) FILTER (WHERE status='archived') FROM recovery_jobs WHERE survey_mission_id=$1`, survey_missionID).Scan(&counts.Queued, &counts.Completed, &counts.Transit, &counts.Accepted, &counts.InProgress, &counts.Verified, &counts.Rejected, &counts.Archived)
	if err != nil {
		return RecoveryJobCounts{}, fmt.Errorf("task counts: %w", err)
	}
	return counts, nil
}

func (p *Postgres) ExpiringRecoveryJobs(ctx context.Context, before time.Time, limit int) ([]domain.RecoveryJob, error) {
	return p.expiringRecoveryJobs(ctx, "", before, limit)
}

func (p *Postgres) ExpiringRecoveryJobsForSurveyMission(ctx context.Context, survey_missionID string, before time.Time, limit int) ([]domain.RecoveryJob, error) {
	return p.expiringRecoveryJobs(ctx, survey_missionID, before, limit)
}

func (p *Postgres) expiringRecoveryJobs(ctx context.Context, survey_missionID string, before time.Time, limit int) ([]domain.RecoveryJob, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	query := `SELECT id,survey_mission_id,acoustic_buoy_id,task_code,status,completed_at,accepted_at,expires_at,version FROM recovery_jobs WHERE status NOT IN ('verified','archived') AND expires_at <= $1`
	args := []any{before, limit}
	if survey_missionID != "" {
		query += " AND survey_mission_id=$3"
		args = append(args, survey_missionID)
	}
	query += " ORDER BY expires_at LIMIT $2"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("expiring recovery_jobs: %w", err)
	}
	defer rows.Close()
	items := make([]domain.RecoveryJob, 0)
	for rows.Next() {
		var task domain.RecoveryJob
		if err := rows.Scan(&task.ID, &task.SurveyMissionID, &task.AcousticBuoyID, &task.TaskCode, &task.Status, &task.CompletedAt, &task.AcceptedAt, &task.ExpiresAt, &task.Version); err != nil {
			return nil, fmt.Errorf("scan expiring task: %w", err)
		}
		items = append(items, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read expiring recovery_jobs: %w", err)
	}
	return items, nil
}
