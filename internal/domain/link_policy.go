package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrIncompatibleLoad = errors.New("incompatible chamber load")
	ErrReleaseBlocked   = errors.New("release blocked")
)

type SpecimenMaterial string

const (
	MaterialPaper   SpecimenMaterial = "paper"
	MaterialTextile SpecimenMaterial = "textile"
	MaterialWood    SpecimenMaterial = "wood"
	MaterialMetal   SpecimenMaterial = "metal"
	MaterialWet     SpecimenMaterial = "wet"
)

type ChamberLoad struct {
	LotID          string
	Material       SpecimenMaterial
	VolumeLitres   int
	RequiresAnoxia bool
}

type ReleaseEvidence struct {
	CycleID             string
	TreatmentPPM        float64
	ResidualPPM         float64
	TreatmentMeasuredAt time.Time
	ResidualMeasuredAt  time.Time
	OperatorID          string
	ReviewerID          string
}

func ValidateChamberLoad(capacityLitres int, loads []ChamberLoad) error {
	if capacityLitres <= 0 || len(loads) == 0 {
		return fmt.Errorf("invalid chamber loading plan: %w", ErrConflict)
	}
	total := 0
	mode := loads[0].RequiresAnoxia
	seen := make(map[string]struct{}, len(loads))
	for _, load := range loads {
		if strings.TrimSpace(load.LotID) == "" || load.VolumeLitres <= 0 {
			return fmt.Errorf("invalid specimen lot: %w", ErrConflict)
		}
		if _, exists := seen[load.LotID]; exists {
			return fmt.Errorf("duplicate specimen lot %s: %w", load.LotID, ErrConflict)
		}
		seen[load.LotID] = struct{}{}
		if load.Material == MaterialWet || load.RequiresAnoxia != mode {
			return fmt.Errorf("lot %s cannot share this cycle: %w", load.LotID, ErrIncompatibleLoad)
		}
		total += load.VolumeLitres
	}
	if total > capacityLitres {
		return fmt.Errorf("planned volume %d exceeds chamber capacity %d: %w", total, capacityLitres, ErrConflict)
	}
	return nil
}

func ValidateReleaseEvidence(now time.Time, evidence ReleaseEvidence) error {
	if strings.TrimSpace(evidence.CycleID) == "" || strings.TrimSpace(evidence.OperatorID) == "" || strings.TrimSpace(evidence.ReviewerID) == "" {
		return fmt.Errorf("release evidence is incomplete: %w", ErrConflict)
	}
	if evidence.OperatorID == evidence.ReviewerID {
		return fmt.Errorf("release requires an independent reviewer: %w", ErrReleaseBlocked)
	}
	if evidence.TreatmentPPM < 450 || evidence.ResidualPPM > 1.0 {
		return fmt.Errorf("installation readings do not satisfy release thresholds: %w", ErrReleaseBlocked)
	}
	if evidence.TreatmentMeasuredAt.IsZero() || evidence.ResidualMeasuredAt.IsZero() || evidence.ResidualMeasuredAt.Before(evidence.TreatmentMeasuredAt) {
		return fmt.Errorf("gas sample chronology is invalid: %w", ErrReleaseBlocked)
	}
	if now.Sub(evidence.ResidualMeasuredAt) > 2*time.Hour || evidence.ResidualMeasuredAt.After(now.Add(time.Minute)) {
		return fmt.Errorf("residual sample is stale or future-dated: %w", ErrReleaseBlocked)
	}
	return nil
}
