package clock

import (
	"testing"
	"time"
)

func TestFixedClockReturnsConfiguredInstant(t *testing.T) {
	instant := time.Date(2026, 8, 18, 1, 2, 3, 0, time.UTC)
	if got := (Fixed{Value: instant}).Now(); !got.Equal(instant) {
		t.Fatalf("got %v", got)
	}
}

func TestInWindowUsesStartInclusiveEndExclusive(t *testing.T) {
	start := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	if !InWindow(start, start, end) {
		t.Fatal("start should be included")
	}
	if InWindow(end, start, end) {
		t.Fatal("end should be excluded")
	}
	if !InWindow(start.Add(30*time.Minute), start, end) {
		t.Fatal("middle should be included")
	}
}

func TestDeadlinePassedNormalizesTimezone(t *testing.T) {
	deadline := time.Date(2026, 8, 18, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	now := deadline.UTC().Add(time.Second)
	if !DeadlinePassed(now, deadline) {
		t.Fatal("deadline should be passed")
	}
}
