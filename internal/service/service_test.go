package service

import (
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func TestCreateDiveWindowRejectsEmptyAndOversizedRequestsBeforePersistence(t *testing.T) {
	svc := New(nil)
	if _, err := svc.CreateDiveWindow(t.Context(), RequestMeta{}, repository.DiveWindowInput{Code: "B", Method: "m", Capacity: 1}, nil); !errors.Is(err, domain.ErrCapacityExceeded) {
		t.Fatalf("empty dive_window error = %v", err)
	}
	if _, err := svc.CreateDiveWindow(t.Context(), RequestMeta{}, repository.DiveWindowInput{Code: "B", Method: "m", Capacity: 1}, []string{"a", "b"}); !errors.Is(err, domain.ErrCapacityExceeded) {
		t.Fatalf("oversized dive_window error = %v", err)
	}
}

func TestCreateSurveyMissionRejectsMissingAcousticBuoysBeforeTransaction(t *testing.T) {
	svc := New(nil)
	now := time.Now().UTC()
	_, err := svc.CreateSurveyMission(t.Context(), RequestMeta{}, CreateSurveyMissionRequest{Code: "P", Name: "SurveyMission", Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: "array_operator", AcousticBuoys: nil})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("missing acoustic_buoys error = %v", err)
	}
}

func TestReviewSignalRecoveryReportRejectsInvalidRecoveryJobVersion(t *testing.T) {
	if domain.RecoveryJobStatus("in_progress").CanMoveTo(domain.RecoveryJobRejected) == false {
		t.Fatal("in-progress acoustic_buoy should support rejection")
	}
	if domain.RecoveryJobStatus("queued").CanMoveTo(domain.RecoveryJobRejected) {
		t.Fatal("queued task cannot be rejected")
	}
}
