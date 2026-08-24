package repository

import (
	"context"
	"time"

	"ufo-observation-command/internal/domain"
)

type Page struct {
	Items  []domain.RecoveryJob `json:"items"`
	Total  int                  `json:"total"`
	Offset int                  `json:"offset"`
	Limit  int                  `json:"limit"`
}

type AcousticBuoyInput struct {
	SurveyMissionID   string `json:"survey_mission_id"`
	Code              string `json:"code"`
	ArrayOpsLane      string `json:"arrayops_lane"`
	RequiredSuccesses int    `json:"required_successes"`
}

type RecoveryJobInput struct {
	SurveyMissionID string    `json:"survey_mission_id"`
	AcousticBuoyID  string    `json:"acoustic_buoy_id"`
	TaskCode        string    `json:"task_code"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type ActivationInput struct {
	RecoveryJobID string
	From          *string
	To            string
	Location      string
	RecordedAt    time.Time
	Note          string
}

type DiveWindowInput struct {
	Code     string `json:"code"`
	Method   string `json:"method"`
	Capacity int    `json:"capacity"`
}

type SignalRecoveryReportInput struct {
	RecoveryJobID  string    `json:"recovery_job_id"`
	DiveWindowID   string    `json:"dive_window_id"`
	RecorderID     string    `json:"recorded_by"`
	RiskScore      float64   `json:"risk_score"`
	Scale          string    `json:"scale"`
	AlertThreshold float64   `json:"alert_threshold"`
	ObservedAt     time.Time `json:"observed_at"`
}

type IntegrityIncidentInput struct {
	RecoveryJobID string    `json:"recovery_job_id"`
	Kind          string    `json:"kind"`
	Reason        string    `json:"reason"`
	DueAt         time.Time `json:"due_at"`
}

type AuditInput struct {
	RequestID       string
	ArrayOperatorID *string
	ObjectType      string
	ObjectID        string
	Action          string
	Outcome         string
	Detail          []byte
}

type Repository interface {
	InTx(context.Context, func(Repository) error) error
	CreateSurveyMission(context.Context, *domain.SurveyMission) error
	GetSurveyMission(context.Context, string) (domain.SurveyMission, error)
	AdvanceSurveyMission(context.Context, string, domain.SurveyMissionStatus, int64) error
	CreateAcousticBuoy(context.Context, AcousticBuoyInput) (string, error)
	CreateRecoveryJob(context.Context, RecoveryJobInput) (domain.RecoveryJob, error)
	GetRecoveryJob(context.Context, string) (domain.RecoveryJob, error)
	MoveRecoveryJob(context.Context, string, domain.RecoveryJobStatus, int64, time.Time) error
	RecordActivation(context.Context, ActivationInput) error
	CreateDiveWindow(context.Context, DiveWindowInput) (string, error)
	AttachRecoveryJobs(context.Context, string, []string) error
	CreateSignalRecoveryReport(context.Context, SignalRecoveryReportInput) (string, error)
	ReviewSignalRecoveryReportRecord(context.Context, string, bool, int64, time.Time) error
	CreateIntegrityIncident(context.Context, IntegrityIncidentInput) (string, error)
	ListRecoveryJobs(context.Context, int, int, string, domain.RecoveryJobStatus) (Page, error)
	DueIntegrityIncidents(context.Context, time.Time, int) ([]IntegrityIncidentInput, error)
	WriteAudit(context.Context, AuditInput) error
	Close() error
}
