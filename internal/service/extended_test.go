package service

import (
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func TestAuthorizeRequiresRepositoryAndPersurvey_mission(t *testing.T) {
	svc := New(nil)
	if err := svc.Authorize(t.Context(), "array_operator", "complete"); err == nil {
		t.Fatal("authorization succeeded without repository")
	}
}

func TestOpenIntegrityIncidentRejectsOldDueTime(t *testing.T) {
	svc := New(nil).WithClock(func() time.Time { return time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC) })
	_, err := svc.OpenIntegrityIncident(t.Context(), RequestMeta{}, repository.IntegrityIncidentInput{RecoveryJobID: "s1", Kind: "reassess", Reason: "bad", DueAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestValidateBulkForAcousticBuoyRejectsMixedAcousticBuoys(t *testing.T) {
	svc := New(nil).WithClock(time.Now)
	requests := []domain.RecoveryJobRequest{{SurveyMissionID: "p", AcousticBuoyID: "s1", TaskCode: "S-1", ExpiresAt: time.Now().Add(time.Hour)}, {SurveyMissionID: "p", AcousticBuoyID: "s2", TaskCode: "S-2", ExpiresAt: time.Now().Add(time.Hour)}}
	if err := svc.ValidateBulkForAcousticBuoy(requests, "s1"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}
