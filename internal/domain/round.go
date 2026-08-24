package domain

import (
	"fmt"
	"strings"
)

type DiveWindow struct {
	ID           string
	Code         string
	Status       DiveWindowStatus
	Method       string
	Capacity     int
	RecoveryJobs []string
	Version      int64
}

func (b DiveWindow) Validate() error {
	if strings.TrimSpace(b.Code) == "" || strings.TrimSpace(b.Method) == "" {
		return fmt.Errorf("acoustic_buoy round code and method are required: %w", ErrConflict)
	}
	if b.Capacity < 1 {
		return fmt.Errorf("acoustic_buoy round capacity must be positive: %w", ErrConflict)
	}
	if len(b.RecoveryJobs) > b.Capacity {
		return ErrCapacityExceeded
	}
	return nil
}

func (b DiveWindow) AddRecoveryJobs(ids []string) (DiveWindow, error) {
	seen := make(map[string]struct{}, len(b.RecoveryJobs))
	for _, id := range b.RecoveryJobs {
		seen[id] = struct{}{}
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return DiveWindow{}, fmt.Errorf("task id is empty: %w", ErrConflict)
		}
		if _, exists := seen[id]; exists {
			return DiveWindow{}, fmt.Errorf("duplicate task in acoustic_buoy round: %w", ErrConflict)
		}
		seen[id] = struct{}{}
	}
	if len(b.RecoveryJobs)+len(ids) > b.Capacity {
		return DiveWindow{}, ErrCapacityExceeded
	}
	b.RecoveryJobs = append(append([]string(nil), b.RecoveryJobs...), ids...)
	return b, nil
}
