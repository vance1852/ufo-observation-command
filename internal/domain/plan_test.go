package domain

import (
	"errors"
	"testing"
	"time"
)

func TestSurveyMissionWindowValidation(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	valid := SurveyMission{Timezone: "Asia/Shanghai", StartsAt: now, EndsAt: now.Add(time.Hour), Status: SurveyMissionDraft}
	if err := valid.ValidateWindow(now); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.EndsAt = invalid.StartsAt
	if err := invalid.ValidateWindow(now); !errors.Is(err, ErrConflict) {
		t.Fatalf("equal window error = %v", err)
	}
	late := valid
	late.Status = SurveyMissionScheduled
	if err := late.ValidateWindow(now.Add(2 * time.Hour)); !errors.Is(err, ErrExpired) {
		t.Fatalf("late window error = %v", err)
	}
}

func TestSurveyMissionCollectionWindow(t *testing.T) {
	start := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	p := SurveyMission{Status: SurveyMissionActive, StartsAt: start, EndsAt: start.Add(time.Hour)}
	if !p.CanExecuteAt(start) {
		t.Fatal("start boundary should be included")
	}
	if p.CanExecuteAt(start.Add(time.Hour)) {
		t.Fatal("end boundary should be excluded")
	}
	if p.RemainingWindow(start.Add(30*time.Minute)) != 30*time.Minute {
		t.Fatal("remaining window mismatch")
	}
	if p.RemainingWindow(start.Add(2*time.Hour)) != 0 {
		t.Fatal("expired window should be zero")
	}
}

func TestAcousticBuoyValidation(t *testing.T) {
	acoustic_buoy := AcousticBuoy{Code: "S-1", ArrayOpsLane: "A-101", RequiredSuccesses: 2}
	if err := acoustic_buoy.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []AcousticBuoy{{ArrayOpsLane: "x", RequiredSuccesses: 1}, {Code: "x", RequiredSuccesses: 0}, {Code: "x", ArrayOpsLane: "x", RequiredSuccesses: 2, Completed: 3}} {
		if err := invalid.Validate(); !errors.Is(err, ErrConflict) {
			t.Fatalf("invalid acoustic_buoy error = %v", err)
		}
	}
}
