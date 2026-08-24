package domain

import (
	"fmt"
	"time"
)

type Assignment struct {
	ID              string    `json:"id"`
	SurveyMissionID string    `json:"survey_mission_id"`
	AcousticBuoyID  string    `json:"acoustic_buoy_id"`
	ArrayOperatorID string    `json:"array_operator_id"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	Status          string    `json:"status"`
}

func (a Assignment) Validate() error {
	if a.ID == "" || a.SurveyMissionID == "" || a.AcousticBuoyID == "" || a.ArrayOperatorID == "" {
		return fmt.Errorf("assignment ids are required: %w", ErrConflict)
	}
	if !a.EndsAt.After(a.StartsAt) {
		return fmt.Errorf("assignment window is invalid: %w", ErrConflict)
	}
	if a.Status != "queued" && a.Status != "active" && a.Status != "completed" && a.Status != "cancelled" {
		return fmt.Errorf("assignment status is invalid: %w", ErrConflict)
	}
	return nil
}

func (a Assignment) ActiveAt(now time.Time) bool {
	return a.Status == "active" && !now.Before(a.StartsAt) && now.Before(a.EndsAt)
}

func AssignmentSourcesFor(next string) ([]string, bool) {
	sources := map[string][]string{
		"active":    {"queued"},
		"completed": {"active"},
		"cancelled": {"queued", "active"},
	}
	allowed, ok := sources[next]
	if !ok {
		return nil, false
	}
	return append([]string(nil), allowed...), true
}

func (a Assignment) CanMoveTo(next string) bool {
	switch a.Status {
	case "queued":
		return next == "active" || next == "cancelled"
	case "active":
		return next == "completed" || next == "cancelled"
	default:
		return false
	}
}
