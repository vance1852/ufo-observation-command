package domain

import (
	"fmt"
	"strings"
	"time"
)

type RecoveryJobRequest struct {
	SurveyMissionID string
	AcousticBuoyID  string
	TaskCode        string
	ExpiresAt       time.Time
}

type BulkItemResult struct {
	Index         int    `json:"index"`
	TaskCode      string `json:"task_code"`
	RecoveryJobID string `json:"recovery_job_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

func (r RecoveryJobRequest) Validate(now time.Time) error {
	if strings.TrimSpace(r.SurveyMissionID) == "" || strings.TrimSpace(r.AcousticBuoyID) == "" || strings.TrimSpace(r.TaskCode) == "" {
		return fmt.Errorf("task survey_mission, acoustic_buoy and code are required: %w", ErrConflict)
	}
	if r.ExpiresAt.Before(now) {
		return ErrExpired
	}
	return nil
}

func ValidateBulkRequests(requests []RecoveryJobRequest, now time.Time) error {
	seen := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		if err := request.Validate(now); err != nil {
			return err
		}
		if _, ok := seen[request.TaskCode]; ok {
			return fmt.Errorf("duplicate external code %s: %w", request.TaskCode, ErrConflict)
		}
		seen[request.TaskCode] = struct{}{}
	}
	return nil
}
