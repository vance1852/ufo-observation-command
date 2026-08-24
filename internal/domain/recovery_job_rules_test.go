package domain

import (
	"errors"
	"testing"
	"time"
)

func TestActivationValidation(t *testing.T) {
	c := Activation{RecoveryJobID: "s1", To: "array_operator-2", Location: "lab", RecordedAt: time.Now().UTC()}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Activation{RecoveryJobID: "s1"}).Validate(); !errors.Is(err, ErrConflict) {
		t.Fatalf("missing activation fields = %v", err)
	}
}

func TestRecoveryJobInProgressRequiresAcceptedAndUnexpired(t *testing.T) {
	now := time.Now().UTC()
	s := RecoveryJob{Status: RecoveryJobAccepted, ExpiresAt: now.Add(time.Hour)}
	if err := s.CanBePerformed(now); err != nil {
		t.Fatal(err)
	}
	s.Status = RecoveryJobCompleted
	if err := s.CanBePerformed(now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("status error = %v", err)
	}
	s.Status = RecoveryJobAccepted
	if err := s.CanBePerformed(now.Add(2 * time.Hour)); !errors.Is(err, ErrExpired) {
		t.Fatalf("expiry error = %v", err)
	}
}

func TestSignalRecoveryReportDecisionRejectsNegativeLimit(t *testing.T) {
	status, err := SignalRecoveryReportDecision(1, 2)
	if err != nil || status != SignalRecoveryReportVerified {
		t.Fatalf("decision = %s, %v", status, err)
	}
	if _, err := SignalRecoveryReportDecision(1, -1); !errors.Is(err, ErrConflict) {
		t.Fatalf("negative limit error = %v", err)
	}
}
