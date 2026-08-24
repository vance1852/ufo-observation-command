package domain

import (
	"errors"
	"testing"
	"time"
)

func TestValidateChamberLoad(t *testing.T) {
	valid := []ChamberLoad{{LotID: "lot-a", Material: MaterialPaper, VolumeLitres: 20, RequiresAnoxia: true}, {LotID: "lot-b", Material: MaterialTextile, VolumeLitres: 25, RequiresAnoxia: true}}
	if err := ValidateChamberLoad(50, valid); err != nil {
		t.Fatalf("valid load: %v", err)
	}
	cases := []struct {
		name   string
		loads  []ChamberLoad
		target error
	}{
		{"duplicate", append(valid, valid[0]), ErrConflict},
		{"wet", []ChamberLoad{{LotID: "wet", Material: MaterialWet, VolumeLitres: 5}}, ErrIncompatibleLoad},
		{"mixed mode", []ChamberLoad{{LotID: "a", Material: MaterialPaper, VolumeLitres: 5}, {LotID: "b", Material: MaterialWood, VolumeLitres: 5, RequiresAnoxia: true}}, ErrIncompatibleLoad},
		{"capacity", valid, ErrConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capacity := 50
			if tc.name == "capacity" {
				capacity = 10
			}
			if err := ValidateChamberLoad(capacity, tc.loads); !errors.Is(err, tc.target) {
				t.Fatalf("got %v, want %v", err, tc.target)
			}
		})
	}
}

func TestValidateReleaseEvidence(t *testing.T) {
	now := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	valid := ReleaseEvidence{CycleID: "cycle-1", TreatmentPPM: 500, ResidualPPM: .8, TreatmentMeasuredAt: now.Add(-8 * time.Hour), ResidualMeasuredAt: now.Add(-30 * time.Minute), OperatorID: "operator-a", ReviewerID: "reviewer-b"}
	if err := ValidateReleaseEvidence(now, valid); err != nil {
		t.Fatalf("valid evidence: %v", err)
	}
	sameReviewer := valid
	sameReviewer.ReviewerID = sameReviewer.OperatorID
	if err := ValidateReleaseEvidence(now, sameReviewer); !errors.Is(err, ErrReleaseBlocked) {
		t.Fatalf("same reviewer: %v", err)
	}
	stale := valid
	stale.ResidualMeasuredAt = now.Add(-3 * time.Hour)
	if err := ValidateReleaseEvidence(now, stale); !errors.Is(err, ErrReleaseBlocked) {
		t.Fatalf("stale sample: %v", err)
	}
	unsafe := valid
	unsafe.ResidualPPM = 1.2
	if err := ValidateReleaseEvidence(now, unsafe); !errors.Is(err, ErrReleaseBlocked) {
		t.Fatalf("unsafe residual: %v", err)
	}
}
