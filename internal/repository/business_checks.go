package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) ValidateSurveyMissionAcousticBuoy(ctx context.Context, survey_missionID, acoustic_buoyID string) error {
	return validateSurveyMissionAcousticBuoy(ctx, p.pool, survey_missionID, acoustic_buoyID)
}

func (t *transaction) ValidateSurveyMissionAcousticBuoy(ctx context.Context, survey_missionID, acoustic_buoyID string) error {
	return validateSurveyMissionAcousticBuoy(ctx, t.tx, survey_missionID, acoustic_buoyID)
}

func validateSurveyMissionAcousticBuoy(ctx context.Context, q sqler, survey_missionID, acoustic_buoyID string) error {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM acoustic_buoys WHERE id=$1 AND survey_mission_id=$2)`, acoustic_buoyID, survey_missionID).Scan(&exists); err != nil {
		return fmt.Errorf("validate survey_mission acoustic_buoy: %w", err)
	}
	if !exists {
		return fmt.Errorf("acoustic_buoy does not belong to survey_mission: %w", domain.ErrConflict)
	}
	return nil
}

func (p *Postgres) ValidateSignalRecoveryReportTarget(ctx context.Context, taskID, dive_windowID string) error {
	return validateSignalRecoveryReportTarget(ctx, p.pool, taskID, dive_windowID)
}

func (t *transaction) ValidateSignalRecoveryReportTarget(ctx context.Context, taskID, dive_windowID string) error {
	return validateSignalRecoveryReportTarget(ctx, t.tx, taskID, dive_windowID)
}

func validateSignalRecoveryReportTarget(ctx context.Context, q sqler, taskID, dive_windowID string) error {
	var taskStatus, dive_windowStatus string
	err := q.QueryRow(ctx, `SELECT s.status,b.status FROM dive_window_items bs JOIN recovery_jobs s ON s.id=bs.recovery_job_id JOIN dive_windows b ON b.id=bs.dive_window_id WHERE bs.recovery_job_id=$1 AND bs.dive_window_id=$2`, taskID, dive_windowID).Scan(&taskStatus, &dive_windowStatus)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("task is not attached to dive_window: %w", domain.ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("validate signal_recovery_report target: %w", err)
	}
	if taskStatus != string(domain.RecoveryJobInProgress) || dive_windowStatus != string(domain.DiveWindowRunning) {
		return fmt.Errorf("task and acoustic_buoy round are not ready for an signal_recovery_report: %w", domain.ErrInvalidTransition)
	}
	return nil
}

func (p *Postgres) SignalRecoveryReportTaskID(ctx context.Context, signal_recovery_reportID string) (string, error) {
	return signal_recovery_reportRecoveryJobID(ctx, p.pool, signal_recovery_reportID)
}

func (t *transaction) SignalRecoveryReportTaskID(ctx context.Context, signal_recovery_reportID string) (string, error) {
	return signal_recovery_reportRecoveryJobID(ctx, t.tx, signal_recovery_reportID)
}

func signal_recovery_reportRecoveryJobID(ctx context.Context, q sqler, signal_recovery_reportID string) (string, error) {
	var taskID string
	if err := q.QueryRow(ctx, `SELECT recovery_job_id FROM signal_recovery_reports WHERE id=$1`, signal_recovery_reportID).Scan(&taskID); err == pgx.ErrNoRows {
		return "", domain.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("get signal_recovery_report task: %w", err)
	}
	return taskID, nil
}
