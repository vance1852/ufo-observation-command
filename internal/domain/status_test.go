package domain

import (
	"errors"
	"testing"
	"time"
)

func TestSurveyMissionTransitions(t *testing.T) {
	cases := []struct {
		from, to SurveyMissionStatus
		want     bool
	}{
		{SurveyMissionDraft, SurveyMissionScheduled, true},
		{SurveyMissionScheduled, SurveyMissionActive, true},
		{SurveyMissionActive, SurveyMissionClosed, true},
		{SurveyMissionDraft, SurveyMissionActive, false},
		{SurveyMissionClosed, SurveyMissionDraft, false},
	}
	for _, tc := range cases {
		if got := tc.from.CanMoveTo(tc.to); got != tc.want {
			t.Errorf("%s -> %s = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestRecoveryJobMoveSetsTimestampsAndVersion(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	task := RecoveryJob{ID: "s1", TaskCode: "S-1", Status: RecoveryJobQueued, ExpiresAt: now.Add(time.Hour), Version: 3}
	updated, err := task.Move(RecoveryJobCompleted, now)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != RecoveryJobCompleted || updated.Version != 4 || updated.CompletedAt == nil || !updated.CompletedAt.Equal(now) {
		t.Fatalf("unexpected collected task: %+v", updated)
	}
}

func TestRecoveryJobRejectsInvalidMoveAndExpiredRecoveryJob(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	task := RecoveryJob{TaskCode: "S-1", Status: RecoveryJobQueued, ExpiresAt: now.Add(-time.Minute), Version: 1}
	if _, err := task.Move(RecoveryJobAccepted, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("invalid transition error = %v", err)
	}
	task.Status = RecoveryJobCompleted
	if _, err := task.Move(RecoveryJobActivationPending, now); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired error = %v", err)
	}
}

func TestSignalRecoveryReportOutcomeUsesInclusiveThreshold(t *testing.T) {
	if SignalRecoveryReportVerified != SignalRecoveryReportStatus("verified") {
		t.Fatal("approved value changed")
	}
	if SignalRecoveryReportStatus(SignalRecoveryReportVerified).Outcome(10, 10) != SignalRecoveryReportVerified {
		t.Fatal("value at limit should be approved")
	}
	if SignalRecoveryReportStatus(SignalRecoveryReportVerified).Outcome(10.01, 10) != SignalRecoveryReportRejected {
		t.Fatal("value over limit should be rejected")
	}
}
