package domain

import (
	"testing"
	"time"
)

func TestInProgressEligibilityRequiresCurrentAcceptedRecoveryJob(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		status    RecoveryJobStatus
		expiresAt time.Time
	}{
		{name: "not accepted", status: RecoveryJobQueued, expiresAt: now.Add(time.Hour)},
		{name: "expired", status: RecoveryJobAccepted, expiresAt: now.Add(-time.Minute)},
	}
	for _, test := range tests {
		if EligibleForExecution(test.status, test.expiresAt, now) {
			t.Fatalf("%s task is eligible", test.name)
		}
	}
}
