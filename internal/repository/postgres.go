package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ufo-observation-command/internal/db"
	"ufo-observation-command/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Postgres struct {
	pool *db.Pool
}

func NewPostgres(pool *db.Pool) *Postgres { return &Postgres{pool: pool} }

func (p *Postgres) InTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(&transaction{tx: tx}); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (p *Postgres) Close() error { p.pool.Close(); return nil }

func (p *Postgres) CreateSurveyMission(ctx context.Context, survey_mission *domain.SurveyMission) error {
	return createSurveyMission(ctx, p.pool, survey_mission)
}

func (p *Postgres) GetSurveyMission(ctx context.Context, id string) (domain.SurveyMission, error) {
	return getSurveyMission(ctx, p.pool, id)
}

func (p *Postgres) AdvanceSurveyMission(ctx context.Context, id string, status domain.SurveyMissionStatus, version int64) error {
	return advanceSurveyMission(ctx, p.pool, id, status, version)
}

func (p *Postgres) CreateAcousticBuoy(ctx context.Context, in AcousticBuoyInput) (string, error) {
	return createAcousticBuoy(ctx, p.pool, in)
}
func (p *Postgres) CreateRecoveryJob(ctx context.Context, in RecoveryJobInput) (domain.RecoveryJob, error) {
	return createRecoveryJob(ctx, p.pool, in)
}
func (p *Postgres) GetRecoveryJob(ctx context.Context, id string) (domain.RecoveryJob, error) {
	return getRecoveryJob(ctx, p.pool, id)
}
func (p *Postgres) MoveRecoveryJob(ctx context.Context, id string, status domain.RecoveryJobStatus, version int64, now time.Time) error {
	return moveRecoveryJob(ctx, p.pool, id, status, version, now)
}
func (p *Postgres) RecordActivation(ctx context.Context, in ActivationInput) error {
	return recordActivation(ctx, p.pool, in)
}
func (p *Postgres) CreateDiveWindow(ctx context.Context, in DiveWindowInput) (string, error) {
	return createDiveWindow(ctx, p.pool, in)
}
func (p *Postgres) AttachRecoveryJobs(ctx context.Context, dive_windowID string, taskIDs []string) error {
	return attachRecoveryJobs(ctx, p.pool, dive_windowID, taskIDs)
}
func (p *Postgres) CreateSignalRecoveryReport(ctx context.Context, in SignalRecoveryReportInput) (string, error) {
	return createSignalRecoveryReport(ctx, p.pool, in)
}
func (p *Postgres) ReviewSignalRecoveryReportRecord(ctx context.Context, id string, accepted bool, version int64, now time.Time) error {
	return reviewSignalRecoveryReportRecord(ctx, p.pool, id, accepted, version, now)
}
func (p *Postgres) CreateIntegrityIncident(ctx context.Context, in IntegrityIncidentInput) (string, error) {
	return createIntegrityIncident(ctx, p.pool, in)
}
func (p *Postgres) ListRecoveryJobs(ctx context.Context, offset, limit int, survey_missionID string, status domain.RecoveryJobStatus) (Page, error) {
	return listRecoveryJobs(ctx, p.pool, offset, limit, survey_missionID, status)
}
func (p *Postgres) DueIntegrityIncidents(ctx context.Context, before time.Time, limit int) ([]IntegrityIncidentInput, error) {
	return dueIntegrityIncidents(ctx, p.pool, before, limit)
}
func (p *Postgres) WriteAudit(ctx context.Context, in AuditInput) error {
	return writeAudit(ctx, p.pool, in)
}

type transaction struct{ tx pgx.Tx }

func (t *transaction) InTx(_ context.Context, _ func(Repository) error) error {
	return errors.New("nested transaction")
}
func (t *transaction) Close() error { return nil }
func (t *transaction) CreateSurveyMission(ctx context.Context, p *domain.SurveyMission) error {
	return createSurveyMission(ctx, t.tx, p)
}
func (t *transaction) GetSurveyMission(ctx context.Context, id string) (domain.SurveyMission, error) {
	return getSurveyMission(ctx, t.tx, id)
}
func (t *transaction) AdvanceSurveyMission(ctx context.Context, id string, s domain.SurveyMissionStatus, v int64) error {
	return advanceSurveyMission(ctx, t.tx, id, s, v)
}
func (t *transaction) CreateAcousticBuoy(ctx context.Context, in AcousticBuoyInput) (string, error) {
	return createAcousticBuoy(ctx, t.tx, in)
}
func (t *transaction) CreateRecoveryJob(ctx context.Context, in RecoveryJobInput) (domain.RecoveryJob, error) {
	return createRecoveryJob(ctx, t.tx, in)
}
func (t *transaction) GetRecoveryJob(ctx context.Context, id string) (domain.RecoveryJob, error) {
	return getRecoveryJob(ctx, t.tx, id)
}
func (t *transaction) MoveRecoveryJob(ctx context.Context, id string, s domain.RecoveryJobStatus, v int64, now time.Time) error {
	return moveRecoveryJob(ctx, t.tx, id, s, v, now)
}
func (t *transaction) RecordActivation(ctx context.Context, in ActivationInput) error {
	return recordActivation(ctx, t.tx, in)
}
func (t *transaction) CreateDiveWindow(ctx context.Context, in DiveWindowInput) (string, error) {
	return createDiveWindow(ctx, t.tx, in)
}
func (t *transaction) AttachRecoveryJobs(ctx context.Context, id string, ids []string) error {
	return attachRecoveryJobs(ctx, t.tx, id, ids)
}
func (t *transaction) CreateSignalRecoveryReport(ctx context.Context, in SignalRecoveryReportInput) (string, error) {
	return createSignalRecoveryReport(ctx, t.tx, in)
}
func (t *transaction) ReviewSignalRecoveryReportRecord(ctx context.Context, id string, accepted bool, v int64, now time.Time) error {
	return reviewSignalRecoveryReportRecord(ctx, t.tx, id, accepted, v, now)
}
func (t *transaction) CreateIntegrityIncident(ctx context.Context, in IntegrityIncidentInput) (string, error) {
	return createIntegrityIncident(ctx, t.tx, in)
}
func (t *transaction) ListRecoveryJobs(ctx context.Context, offset, limit int, survey_missionID string, status domain.RecoveryJobStatus) (Page, error) {
	return listRecoveryJobs(ctx, t.tx, offset, limit, survey_missionID, status)
}
func (t *transaction) DueIntegrityIncidents(ctx context.Context, before time.Time, limit int) ([]IntegrityIncidentInput, error) {
	return dueIntegrityIncidents(ctx, t.tx, before, limit)
}
func (t *transaction) WriteAudit(ctx context.Context, in AuditInput) error {
	return writeAudit(ctx, t.tx, in)
}

type sqler interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func createSurveyMission(ctx context.Context, q sqler, survey_mission *domain.SurveyMission) error {
	_, err := q.Exec(ctx, `INSERT INTO survey_missions(id,code,name,status,timezone,starts_at,ends_at,version,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, survey_mission.ID, survey_mission.Code, survey_mission.Name, survey_mission.Status, survey_mission.Timezone, survey_mission.StartsAt, survey_mission.EndsAt, survey_mission.Version, survey_mission.CreatedBy)
	return wrapWrite(err)
}

func getSurveyMission(ctx context.Context, q sqler, id string) (domain.SurveyMission, error) {
	var p domain.SurveyMission
	err := q.QueryRow(ctx, `SELECT id,code,name,status,timezone,starts_at,ends_at,version,created_by FROM survey_missions WHERE id=$1`, id).Scan(&p.ID, &p.Code, &p.Name, &p.Status, &p.Timezone, &p.StartsAt, &p.EndsAt, &p.Version, &p.CreatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SurveyMission{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SurveyMission{}, fmt.Errorf("get survey_mission: %w", err)
	}
	return p, nil
}

func advanceSurveyMission(ctx context.Context, q sqler, id string, status domain.SurveyMissionStatus, version int64) error {
	result, err := q.Exec(ctx, `UPDATE survey_missions SET status=$1,version=version+1 WHERE id=$2 AND version=$3`, status, id, version)
	if err != nil {
		return fmt.Errorf("advance survey_mission: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func createAcousticBuoy(ctx context.Context, q sqler, in AcousticBuoyInput) (string, error) {
	id := uuid.NewString()
	_, err := q.Exec(ctx, `INSERT INTO acoustic_buoys(id,survey_mission_id,code,arrayops_lane,required_successes) VALUES ($1,$2,$3,$4,$5)`, id, in.SurveyMissionID, in.Code, in.ArrayOpsLane, in.RequiredSuccesses)
	return wrapID(err, id)
}

func createRecoveryJob(ctx context.Context, q sqler, in RecoveryJobInput) (domain.RecoveryJob, error) {
	s := domain.RecoveryJob{ID: uuid.NewString(), SurveyMissionID: in.SurveyMissionID, AcousticBuoyID: in.AcousticBuoyID, TaskCode: in.TaskCode, Status: domain.RecoveryJobQueued, ExpiresAt: in.ExpiresAt, Version: 1}
	_, err := q.Exec(ctx, `INSERT INTO recovery_jobs(id,survey_mission_id,acoustic_buoy_id,task_code,status,expires_at,version) VALUES ($1,$2,$3,$4,$5,$6,$7)`, s.ID, s.SurveyMissionID, s.AcousticBuoyID, s.TaskCode, s.Status, s.ExpiresAt, s.Version)
	return wrapRecoveryJob(err, s)
}

func getRecoveryJob(ctx context.Context, q sqler, id string) (domain.RecoveryJob, error) {
	var s domain.RecoveryJob
	err := q.QueryRow(ctx, `SELECT id,survey_mission_id,acoustic_buoy_id,task_code,status,completed_at,accepted_at,expires_at,version FROM recovery_jobs WHERE id=$1`, id).Scan(&s.ID, &s.SurveyMissionID, &s.AcousticBuoyID, &s.TaskCode, &s.Status, &s.CompletedAt, &s.AcceptedAt, &s.ExpiresAt, &s.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RecoveryJob{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RecoveryJob{}, fmt.Errorf("get task: %w", err)
	}
	return s, nil
}

func moveRecoveryJob(ctx context.Context, q sqler, id string, status domain.RecoveryJobStatus, version int64, now time.Time) error {
	if status == domain.RecoveryJobCompleted {
		result, err := q.Exec(ctx, `WITH eligible_acoustic_buoy AS (
			SELECT id FROM acoustic_buoys WHERE id=(SELECT acoustic_buoy_id FROM recovery_jobs WHERE id=$3) AND completed_installs < required_successes FOR UPDATE
		), updated AS (
			UPDATE recovery_jobs SET status=$1,version=version+1,completed_at=$2 WHERE id=$3 AND version=$4 AND EXISTS (SELECT 1 FROM eligible_acoustic_buoy) RETURNING acoustic_buoy_id
		)
		UPDATE acoustic_buoys SET completed_installs=completed_installs+1 WHERE id IN (SELECT acoustic_buoy_id FROM updated)`, status, now, id, version)
		if err != nil {
			return fmt.Errorf("complete task: %w", err)
		}
		if result.RowsAffected() != 1 {
			return domain.ErrConflict
		}
		return nil
	}
	result, err := q.Exec(ctx, `UPDATE recovery_jobs SET status=$1,version=version+1,completed_at=CASE WHEN $1='completed' THEN $2 ELSE completed_at END,accepted_at=CASE WHEN $1='accepted' THEN $2 ELSE accepted_at END WHERE id=$3 AND version=$4`, status, now, id, version)
	if err != nil {
		return fmt.Errorf("move task: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func recordActivation(ctx context.Context, q sqler, in ActivationInput) error {
	_, err := q.Exec(ctx, `INSERT INTO mooring_events(id,recovery_job_id,from_operator,to_operator,location,recorded_at,note) VALUES ($1,$2,$3,$4,$5,$6,$7)`, uuid.NewString(), in.RecoveryJobID, in.From, in.To, in.Location, in.RecordedAt, in.Note)
	return wrapWrite(err)
}

func createDiveWindow(ctx context.Context, q sqler, in DiveWindowInput) (string, error) {
	id := uuid.NewString()
	_, err := q.Exec(ctx, `INSERT INTO dive_windows(id,code,status,method,capacity) VALUES ($1,$2,'queued',$3,$4)`, id, in.Code, in.Method, in.Capacity)
	return wrapID(err, id)
}

func attachRecoveryJobs(ctx context.Context, q sqler, dive_windowID string, taskIDs []string) error {
	for _, taskID := range taskIDs {
		if _, err := q.Exec(ctx, `INSERT INTO dive_window_items(dive_window_id,recovery_job_id) VALUES ($1,$2)`, dive_windowID, taskID); err != nil {
			return wrapWrite(err)
		}
		result, err := q.Exec(ctx, `UPDATE recovery_jobs SET status='in_progress',version=version+1 WHERE id=$1 AND status='accepted' AND expires_at >= now()`, taskID)
		if err != nil {
			return wrapWrite(err)
		}
		if result.RowsAffected() != 1 {
			return fmt.Errorf("task %s is not eligible for acoustic_buoy-round execution: %w", taskID, domain.ErrInvalidTransition)
		}
	}
	return nil
}

func createSignalRecoveryReport(ctx context.Context, q sqler, in SignalRecoveryReportInput) (string, error) {
	id := uuid.NewString()
	_, err := q.Exec(ctx, `INSERT INTO signal_recovery_reports(id,recovery_job_id,dive_window_id,recorded_by,status,risk_score,scale,alert_threshold,observed_at) VALUES ($1,$2,$3,$4,'pending',$5,$6,$7,$8)`, id, in.RecoveryJobID, in.DiveWindowID, in.RecorderID, in.RiskScore, in.Scale, in.AlertThreshold, in.ObservedAt)
	return wrapID(err, id)
}

func reviewSignalRecoveryReportRecord(ctx context.Context, q sqler, id string, accepted bool, version int64, now time.Time) error {
	status := domain.SignalRecoveryReportVerified
	if !accepted {
		status = domain.SignalRecoveryReportRejected
	}
	result, err := q.Exec(ctx, `UPDATE signal_recovery_reports SET status=$1,reviewed_at=$2,version=version+1 WHERE id=$3 AND status='pending' AND version=$4`, status, now, id, version)
	if err != nil {
		return fmt.Errorf("review signal_recovery_report: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func createIntegrityIncident(ctx context.Context, q sqler, in IntegrityIncidentInput) (string, error) {
	id := uuid.NewString()
	_, err := q.Exec(ctx, `INSERT INTO integrity_incidents(id,recovery_job_id,kind,status,reason,due_at) VALUES ($1,$2,$3,'open',$4,$5)`, id, in.RecoveryJobID, in.Kind, in.Reason, in.DueAt)
	return wrapID(err, id)
}

func listRecoveryJobs(ctx context.Context, q sqler, offset, limit int, survey_missionID string, status domain.RecoveryJobStatus) (Page, error) {
	page := Page{Offset: offset, Limit: limit, Items: make([]domain.RecoveryJob, 0)}
	args := []any{limit, offset}
	where := "WHERE TRUE"
	if survey_missionID != "" {
		args = append(args, survey_missionID)
		where += fmt.Sprintf(" AND survey_mission_id=$%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status=$%d", len(args))
	}
	query := fmt.Sprintf(`SELECT id,survey_mission_id,acoustic_buoy_id,task_code,status,completed_at,accepted_at,expires_at,version FROM recovery_jobs %s ORDER BY created_at DESC LIMIT $1 OFFSET $2`, where)
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return Page{}, fmt.Errorf("list recovery_jobs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s domain.RecoveryJob
		if err := rows.Scan(&s.ID, &s.SurveyMissionID, &s.AcousticBuoyID, &s.TaskCode, &s.Status, &s.CompletedAt, &s.AcceptedAt, &s.ExpiresAt, &s.Version); err != nil {
			return Page{}, fmt.Errorf("scan task: %w", err)
		}
		page.Items = append(page.Items, s)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("list task rows: %w", err)
	}
	countWhere := "WHERE TRUE"
	countArgs := make([]any, 0, len(args)-2)
	if survey_missionID != "" {
		countArgs = append(countArgs, survey_missionID)
		countWhere += fmt.Sprintf(" AND survey_mission_id=$%d", len(countArgs))
	}
	if status != "" {
		countArgs = append(countArgs, status)
		countWhere += fmt.Sprintf(" AND status=$%d", len(countArgs))
	}
	countQuery := fmt.Sprintf("SELECT count(*) FROM recovery_jobs %s", countWhere)
	if err := q.QueryRow(ctx, countQuery, countArgs...).Scan(&page.Total); err != nil {
		return Page{}, fmt.Errorf("count recovery_jobs: %w", err)
	}
	return page, nil
}

func dueIntegrityIncidents(ctx context.Context, q sqler, before time.Time, limit int) ([]IntegrityIncidentInput, error) {
	rows, err := q.Query(ctx, `SELECT recovery_job_id,kind,reason,due_at FROM integrity_incidents WHERE status IN ('open','in_progress') AND due_at <= $1 ORDER BY due_at LIMIT $2`, before, limit)
	if err != nil {
		return nil, fmt.Errorf("due integrity_incidents: %w", err)
	}
	defer rows.Close()
	out := make([]IntegrityIncidentInput, 0)
	for rows.Next() {
		var item IntegrityIncidentInput
		if err := rows.Scan(&item.RecoveryJobID, &item.Kind, &item.Reason, &item.DueAt); err != nil {
			return nil, fmt.Errorf("scan safety_alert: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func writeAudit(ctx context.Context, q sqler, in AuditInput) error {
	detail := in.Detail
	if len(detail) == 0 {
		detail = []byte(`{}`)
	}
	if !json.Valid(detail) {
		return fmt.Errorf("invalid audit detail: %w", domain.ErrConflict)
	}
	_, err := q.Exec(ctx, `INSERT INTO audit_events(id,request_id,array_operator_id,object_type,object_id,action,outcome,detail) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, uuid.NewString(), in.RequestID, in.ArrayOperatorID, in.ObjectType, in.ObjectID, in.Action, in.Outcome, detail)
	return wrapWrite(err)
}

func wrapWrite(err error) error {
	if err == nil {
		return nil
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "22003", "22P02", "23502", "23503", "23505", "23514":
			return fmt.Errorf("repository write: %w: %w", err, domain.ErrConflict)
		}
	}
	return fmt.Errorf("repository write: %w", err)
}
func wrapID(err error, id string) (string, error) {
	if err != nil {
		return "", wrapWrite(err)
	}
	return id, nil
}
func wrapRecoveryJob(err error, s domain.RecoveryJob) (domain.RecoveryJob, error) {
	if err != nil {
		return domain.RecoveryJob{}, wrapWrite(err)
	}
	return s, nil
}
