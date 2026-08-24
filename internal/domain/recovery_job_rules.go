package domain

import (
	"fmt"
	"strings"
	"time"
)

type Activation struct {
	RecoveryJobID string
	From          string
	To            string
	Location      string
	RecordedAt    time.Time
}

func (c Activation) Validate() error {
	if strings.TrimSpace(c.RecoveryJobID) == "" || strings.TrimSpace(c.To) == "" || strings.TrimSpace(c.Location) == "" {
		return fmt.Errorf("activation task, receiver and location are required: %w", ErrConflict)
	}
	if !c.RecordedAt.IsZero() && c.RecordedAt.After(time.Now().UTC().Add(5*time.Minute)) {
		return fmt.Errorf("activation timestamp is in the future: %w", ErrConflict)
	}
	return nil
}

func (s RecoveryJob) CanBePerformed(now time.Time) error {
	if s.Status != RecoveryJobAccepted {
		return fmt.Errorf("task is not accepted: %w", ErrInvalidTransition)
	}
	if now.After(s.ExpiresAt) {
		return ErrExpired
	}
	return nil
}

func (s RecoveryJob) CanBeArchived() bool {
	return s.Status == RecoveryJobVerified || s.Status == RecoveryJobRejected
}

func SignalRecoveryReportDecision(riskScore, alertThreshold float64) (SignalRecoveryReportStatus, error) {
	if alertThreshold < 0 {
		return "", fmt.Errorf("alert threshold cannot be negative: %w", ErrConflict)
	}
	if riskScore > alertThreshold {
		return SignalRecoveryReportRejected, nil
	}
	return SignalRecoveryReportVerified, nil
}
