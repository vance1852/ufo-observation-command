package domain

import (
	"fmt"
	"strings"
	"time"
)

type IntegrityIncidentStatus string

const (
	IntegrityIncidentOpen       IntegrityIncidentStatus = "open"
	IntegrityIncidentInProgress IntegrityIncidentStatus = "in_progress"
	IntegrityIncidentClosed     IntegrityIncidentStatus = "closed"
)

type IntegrityIncident struct {
	ID            string
	RecoveryJobID string
	Kind          string
	Status        IntegrityIncidentStatus
	Reason        string
	DueAt         time.Time
	ClosedAt      *time.Time
}

func (d IntegrityIncident) Validate(now time.Time) error {
	if d.RecoveryJobID == "" || strings.TrimSpace(d.Kind) == "" || strings.TrimSpace(d.Reason) == "" {
		return fmt.Errorf("safety_alert fields are required: %w", ErrConflict)
	}
	switch d.Kind {
	case "reassess", "repeat_acoustic_buoy", "safety_adjustment", "close_record":
	default:
		return fmt.Errorf("safety_alert kind is invalid: %w", ErrConflict)
	}
	if d.DueAt.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("safety_alert due time is too old: %w", ErrConflict)
	}
	if d.Status == IntegrityIncidentClosed && d.ClosedAt == nil {
		return fmt.Errorf("closed safety_alert needs closed_at: %w", ErrConflict)
	}
	return nil
}

func (d IntegrityIncident) IsDue(now time.Time) bool {
	return d.Status != IntegrityIncidentClosed && !d.DueAt.After(now)
}
