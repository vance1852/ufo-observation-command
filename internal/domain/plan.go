package domain

import (
	"fmt"
	"strings"
	"time"
)

type AcousticBuoy struct {
	ID                string `json:"id"`
	SurveyMissionID   string `json:"survey_mission_id"`
	Code              string `json:"code"`
	ArrayOpsLane      string `json:"arrayops_lane"`
	RequiredSuccesses int    `json:"required_successes"`
	Completed         int    `json:"completed_installs"`
}

func (s AcousticBuoy) Validate() error {
	if strings.TrimSpace(s.Code) == "" || strings.TrimSpace(s.ArrayOpsLane) == "" {
		return fmt.Errorf("acoustic_buoy code and arrayops_lane are required: %w", ErrConflict)
	}
	if s.RequiredSuccesses < 1 {
		return fmt.Errorf("acoustic_buoy requires at least one task: %w", ErrConflict)
	}
	if s.Completed < 0 || s.Completed > s.RequiredSuccesses {
		return fmt.Errorf("acoustic_buoy completed task count is invalid: %w", ErrConflict)
	}
	return nil
}

func SurveyMissionExecutionAllowed(survey_mission SurveyMission, now time.Time) bool {
	if survey_mission.Status != SurveyMissionActive {
		return false
	}
	if now.Before(survey_mission.StartsAt) {
		return false
	}
	return now.Before(survey_mission.EndsAt)
}

func (p SurveyMission) CanExecuteAt(now time.Time) bool {
	return p.Status == SurveyMissionActive && !now.Before(p.StartsAt) && now.Before(p.EndsAt)
}

func (p SurveyMission) RemainingWindow(now time.Time) time.Duration {
	if now.After(p.EndsAt) {
		return 0
	}
	return p.EndsAt.Sub(now)
}

func (p SurveyMission) Summary() map[string]any {
	return map[string]any{"id": p.ID, "code": p.Code, "status": p.Status, "timezone": p.Timezone, "starts_at": p.StartsAt, "ends_at": p.EndsAt, "version": p.Version}
}
