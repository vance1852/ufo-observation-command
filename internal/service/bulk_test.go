package service

import (
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/domain"
)

func TestCreateRecoveryJobsBulkRejectsEmptyInput(t *testing.T) {
	svc := New(nil)
	if _, err := svc.CreateRecoveryJobsBulk(t.Context(), RequestMeta{}, nil); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestSearchRequestUsesStableDefaults(t *testing.T) {
	request := domain.SearchRequest{}
	request = request.Normalize()
	if request.Sort != domain.SortCreated || request.Limit != 50 || request.Offset != 0 {
		t.Fatalf("request=%+v", request)
	}
}

func TestValidateBulkForAcousticBuoyAcceptsSameAcousticBuoy(t *testing.T) {
	svc := New(nil).WithClock(func() time.Time { return time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC) })
	requests := []domain.RecoveryJobRequest{{SurveyMissionID: "p", AcousticBuoyID: "s", TaskCode: "S-1", ExpiresAt: time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)}}
	if err := svc.ValidateBulkForAcousticBuoy(requests, "s"); err != nil {
		t.Fatal(err)
	}
}
