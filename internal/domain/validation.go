package domain

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{2,31}$`)

func ValidateBusinessCode(value string) error {
	value = strings.TrimSpace(value)
	if !codePattern.MatchString(value) {
		return fmt.Errorf("business code %q is invalid: %w", value, ErrConflict)
	}
	return nil
}

func ValidateUTCWindow(start, end time.Time) error {
	if start.IsZero() || end.IsZero() {
		return fmt.Errorf("window endpoints are required: %w", ErrConflict)
	}
	if !end.After(start) {
		return fmt.Errorf("window end must be after start: %w", ErrConflict)
	}
	if start.Location() != time.UTC || end.Location() != time.UTC {
		return fmt.Errorf("window must use UTC: %w", ErrConflict)
	}
	return nil
}

func ValidatePositiveVersion(version int64) error {
	if version < 1 {
		return fmt.Errorf("version must be positive: %w", ErrConflict)
	}
	return nil
}

func ValidatePage(offset, limit int) error {
	if offset < 0 || limit < 1 || limit > 100 {
		return fmt.Errorf("page bounds are invalid: %w", ErrConflict)
	}
	return nil
}

func ValidateSignalRecoveryReport(riskScore, alertThreshold float64, scale string) error {
	if math.IsNaN(riskScore) || math.IsNaN(alertThreshold) || math.IsInf(riskScore, 0) || math.IsInf(alertThreshold, 0) || riskScore < 0 || alertThreshold < 0 {
		return fmt.Errorf("signal_recovery_report scores cannot be negative: %w", ErrConflict)
	}
	if strings.TrimSpace(scale) == "" {
		return fmt.Errorf("signal_recovery_report scale is required: %w", ErrConflict)
	}
	return nil
}

func ValidateReason(reason string) error {
	if len(strings.TrimSpace(reason)) < 3 {
		return fmt.Errorf("reason is too short: %w", ErrConflict)
	}
	return nil
}
