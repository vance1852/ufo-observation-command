package domain

import (
	"errors"
	"testing"
)

func TestDiveWindowAddsRecoveryJobsWithoutSharingSlice(t *testing.T) {
	b := DiveWindow{Code: "B-1", Method: "water-ph", Capacity: 3, RecoveryJobs: []string{"s1"}}
	updated, err := b.AddRecoveryJobs([]string{"s2"})
	if err != nil {
		t.Fatal(err)
	}
	updated.RecoveryJobs[0] = "changed"
	if b.RecoveryJobs[0] != "s1" {
		t.Fatal("dive_window input slice was polluted")
	}
	if len(updated.RecoveryJobs) != 2 {
		t.Fatalf("recovery_jobs = %v", updated.RecoveryJobs)
	}
}

func TestDiveWindowRejectsDuplicateAndCapacityOverflow(t *testing.T) {
	b := DiveWindow{Code: "B-1", Method: "water-ph", Capacity: 2, RecoveryJobs: []string{"s1"}}
	if _, err := b.AddRecoveryJobs([]string{"s1"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := b.AddRecoveryJobs([]string{"s2", "s3"}); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("capacity error = %v", err)
	}
}

func TestDiveWindowValidation(t *testing.T) {
	if err := (DiveWindow{Code: "B", Method: "m", Capacity: 1}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (DiveWindow{Code: "B", Method: "m", Capacity: 0}).Validate(); !errors.Is(err, ErrConflict) {
		t.Fatalf("invalid capacity error = %v", err)
	}
}
