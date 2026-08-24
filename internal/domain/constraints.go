package domain

import (
	"fmt"
	"time"
)

type ConstraintSet struct {
	MaxRecoveryJobsPerDiveWindow int
	MinimumRemainingTTL          time.Duration
	RequireActivationChain       bool
}

func (c ConstraintSet) Validate() error {
	if c.MaxRecoveryJobsPerDiveWindow < 1 {
		return fmt.Errorf("max recovery_jobs per dive_window must be positive: %w", ErrConflict)
	}
	if c.MinimumRemainingTTL < 0 {
		return fmt.Errorf("minimum ttl cannot be negative: %w", ErrConflict)
	}
	return nil
}

func (c ConstraintSet) AllowsRecoveryJob(task RecoveryJob, now time.Time) bool {
	if task.Status != RecoveryJobAccepted {
		return false
	}
	return task.ExpiresAt.Sub(now) >= c.MinimumRemainingTTL
}

func SameSurveyMission(recovery_jobs []RecoveryJob) bool {
	if len(recovery_jobs) < 2 {
		return true
	}
	survey_mission := recovery_jobs[0].SurveyMissionID
	for _, task := range recovery_jobs[1:] {
		if task.SurveyMissionID != survey_mission {
			return false
		}
	}
	return true
}
