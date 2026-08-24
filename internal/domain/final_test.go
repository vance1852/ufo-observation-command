package domain

import (
	"errors"
	"testing"
	"time"
)

func TestSearchDescendingSortAndPagination(t *testing.T) {
	items := []RecoveryJob{{ID: "a", TaskCode: "A", Status: RecoveryJobAccepted}, {ID: "c", TaskCode: "C", Status: RecoveryJobAccepted}, {ID: "b", TaskCode: "B", Status: RecoveryJobAccepted}}
	got := SearchRecoveryJobs(items, SearchRequest{Sort: SortCode, Desc: true, Limit: 2})
	if len(got) != 2 || got[0].ID != "c" || got[1].ID != "b" {
		t.Fatalf("got=%+v", got)
	}
}

func TestSameSurveyMissionRequiresAllRecoveryJobsToMatch(t *testing.T) {
	if !SameSurveyMission([]RecoveryJob{{SurveyMissionID: "p"}, {SurveyMissionID: "p"}}) {
		t.Fatal("same cases rejected")
	}
	if SameSurveyMission([]RecoveryJob{{SurveyMissionID: "p"}, {SurveyMissionID: "q"}}) {
		t.Fatal("different cases accepted")
	}
}

func TestIntegrityIncidentRejectsTooOldDueDate(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	d := IntegrityIncident{RecoveryJobID: "s", Kind: "reassess", Status: IntegrityIncidentOpen, Reason: "bad", DueAt: now.Add(-25 * time.Hour)}
	if err := d.Validate(now); !errors.Is(err, ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestDiveWindowRejectsBlankRecoveryJobID(t *testing.T) {
	b := DiveWindow{Code: "B", Method: "m", Capacity: 2}
	if _, err := b.AddRecoveryJobs([]string{""}); !errors.Is(err, ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestUTCWindowRejectsLocalTime(t *testing.T) {
	start := time.Date(2026, 8, 18, 9, 0, 0, 0, time.Local)
	end := start.Add(time.Hour)
	if err := ValidateUTCWindow(start, end); !errors.Is(err, ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestSurveyMissionSummaryContainsStableFields(t *testing.T) {
	p := SurveyMission{ID: "p", Code: "P-1", Status: SurveyMissionDraft, Version: 3}
	summary := p.Summary()
	if summary["id"] != "p" || summary["version"] != int64(3) {
		t.Fatalf("summary=%+v", summary)
	}
}
