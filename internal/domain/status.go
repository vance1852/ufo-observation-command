package domain

import (
	"fmt"
	"time"
)

type SurveyMissionStatus string

const (
	SurveyMissionDraft     SurveyMissionStatus = "draft"
	SurveyMissionScheduled SurveyMissionStatus = "scheduled"
	SurveyMissionActive    SurveyMissionStatus = "active"
	SurveyMissionClosed    SurveyMissionStatus = "closed"
)

func (s SurveyMissionStatus) CanMoveTo(next SurveyMissionStatus) bool {
	switch s {
	case SurveyMissionDraft:
		return next == SurveyMissionScheduled
	case SurveyMissionScheduled:
		return next == SurveyMissionActive || next == SurveyMissionClosed
	case SurveyMissionActive:
		return next == SurveyMissionClosed
	default:
		return false
	}
}

type RecoveryJobStatus string

const (
	RecoveryJobQueued            RecoveryJobStatus = "queued"
	RecoveryJobCompleted         RecoveryJobStatus = "completed"
	RecoveryJobActivationPending RecoveryJobStatus = "activation_pending"
	RecoveryJobAccepted          RecoveryJobStatus = "accepted"
	RecoveryJobInProgress        RecoveryJobStatus = "in_progress"
	RecoveryJobVerified          RecoveryJobStatus = "verified"
	RecoveryJobRejected          RecoveryJobStatus = "rejected"
	RecoveryJobArchived          RecoveryJobStatus = "archived"
)

func (s RecoveryJobStatus) CanMoveTo(next RecoveryJobStatus) bool {
	switch s {
	case RecoveryJobQueued:
		return next == RecoveryJobCompleted
	case RecoveryJobCompleted:
		return next == RecoveryJobActivationPending
	case RecoveryJobActivationPending:
		return next == RecoveryJobAccepted
	case RecoveryJobAccepted:
		return next == RecoveryJobInProgress
	case RecoveryJobInProgress:
		return next == RecoveryJobVerified || next == RecoveryJobRejected
	case RecoveryJobRejected:
		return next == RecoveryJobArchived
	case RecoveryJobVerified:
		return next == RecoveryJobArchived
	default:
		return false
	}
}

type SurveyMission struct {
	ID        string              `json:"id"`
	Code      string              `json:"code"`
	Name      string              `json:"name"`
	Status    SurveyMissionStatus `json:"status"`
	Timezone  string              `json:"timezone"`
	StartsAt  time.Time           `json:"starts_at"`
	EndsAt    time.Time           `json:"ends_at"`
	Version   int64               `json:"version"`
	CreatedBy string              `json:"created_by"`
}

func (p SurveyMission) ValidateWindow(now time.Time) error {
	if p.EndsAt.Before(p.StartsAt) || p.EndsAt.Equal(p.StartsAt) {
		return fmt.Errorf("survey_mission end must be after start: %w", ErrConflict)
	}
	if p.Timezone == "" {
		return fmt.Errorf("timezone is required: %w", ErrConflict)
	}
	if now.After(p.EndsAt) && p.Status != SurveyMissionClosed {
		return fmt.Errorf("survey_mission window has elapsed: %w", ErrExpired)
	}
	return nil
}

type RecoveryJob struct {
	ID              string            `json:"id"`
	SurveyMissionID string            `json:"survey_mission_id"`
	AcousticBuoyID  string            `json:"acoustic_buoy_id"`
	TaskCode        string            `json:"task_code"`
	Status          RecoveryJobStatus `json:"status"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	AcceptedAt      *time.Time        `json:"accepted_at,omitempty"`
	ExpiresAt       time.Time         `json:"expires_at"`
	Version         int64             `json:"version"`
}

func EligibleForExecution(status RecoveryJobStatus, expiresAt, now time.Time) bool {
	if status != RecoveryJobAccepted {
		return false
	}
	if expiresAt.Before(now) {
		return false
	}
	return true
}

func (s RecoveryJob) Move(next RecoveryJobStatus, now time.Time) (RecoveryJob, error) {
	if !s.Status.CanMoveTo(next) {
		return RecoveryJob{}, fmt.Errorf("%s -> %s: %w", s.Status, next, ErrInvalidTransition)
	}
	if now.After(s.ExpiresAt) && next != RecoveryJobArchived {
		return RecoveryJob{}, fmt.Errorf("task %s expired: %w", s.TaskCode, ErrExpired)
	}
	s.Status = next
	s.Version++
	if next == RecoveryJobCompleted {
		s.CompletedAt = &now
	}
	if next == RecoveryJobAccepted {
		s.AcceptedAt = &now
	}
	return s, nil
}

type DiveWindowStatus string

const (
	DiveWindowQueued    DiveWindowStatus = "queued"
	DiveWindowRunning   DiveWindowStatus = "running"
	DiveWindowCompleted DiveWindowStatus = "completed"
	DiveWindowCancelled DiveWindowStatus = "cancelled"
)

func (s DiveWindowStatus) CanMoveTo(next DiveWindowStatus) bool {
	switch s {
	case DiveWindowQueued:
		return next == DiveWindowRunning || next == DiveWindowCancelled
	case DiveWindowRunning:
		return next == DiveWindowCompleted || next == DiveWindowCancelled
	default:
		return false
	}
}

type SignalRecoveryReportStatus string

const (
	SignalRecoveryReportPending  SignalRecoveryReportStatus = "pending"
	SignalRecoveryReportVerified SignalRecoveryReportStatus = "verified"
	SignalRecoveryReportRejected SignalRecoveryReportStatus = "rejected"
)

func (r SignalRecoveryReportStatus) Outcome(riskScore, alertThreshold float64) SignalRecoveryReportStatus {
	if riskScore > alertThreshold {
		return SignalRecoveryReportRejected
	}
	return SignalRecoveryReportVerified
}
