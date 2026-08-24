package arrayops

import (
	"fmt"
	"slices"
	"strings"
)

func IdempotencyKey(tenantID, method, path, key string) (string, error) {
	values := []string{tenantID, method, path, key}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("idempotency scope is incomplete: %w", ErrInvalid)
		}
	}
	return strings.Join(values, "\x00"), nil
}

func CallbackKey(callback Callback) (string, error) {
	if callback.TenantID == "" || callback.BuoyID == "" || callback.CalibrationBundleID == "" || callback.EventID == "" {
		return "", fmt.Errorf("callback identity is incomplete: %w", ErrInvalid)
	}
	return strings.Join([]string{callback.TenantID, callback.BuoyID, callback.CalibrationBundleID, callback.EventID}, "\x00"), nil
}

func CalibrationBundleSupports(calibration_bundle CalibrationBundle, buoy Buoy) error {
	if calibration_bundle.TenantID != buoy.TenantID {
		return fmt.Errorf("calibration_bundle and buoy tenants differ: %w", ErrConflict)
	}
	if !calibration_bundle.Signed || calibration_bundle.Digest == "" {
		return fmt.Errorf("calibration_bundle signature is not verified: %w", ErrConflict)
	}
	if !slices.Contains(calibration_bundle.BuoyClasses, buoy.Class) {
		return fmt.Errorf("calibration_bundle does not support buoy class %s: %w", buoy.Class, ErrConflict)
	}
	return nil
}

func CanDispatch(survey_mission SurveyMission) error {
	if survey_mission.State != "running" {
		return fmt.Errorf("survey_mission %s cannot dispatch from %s: %w", survey_mission.ID, survey_mission.State, ErrConflict)
	}
	if survey_mission.ApprovedDigest == "" {
		return fmt.Errorf("survey_mission %s has no approved calibration_bundle digest: %w", survey_mission.ID, ErrConflict)
	}
	return nil
}

func CanPromote(survey_mission SurveyMission) error {
	if survey_mission.State != "running" {
		return fmt.Errorf("survey_mission %s is not running: %w", survey_mission.ID, ErrConflict)
	}
	if survey_mission.Failed > 0 || survey_mission.Healthy < survey_mission.RequiredHealthy {
		return fmt.Errorf("pilot_dive health gate is not satisfied: %w", ErrConflict)
	}
	return nil
}

func RollbackVersion(buoy Buoy) (string, error) {
	if buoy.PreviousVersion == "" || buoy.PreviousVersion == buoy.CurrentVersion {
		return "", fmt.Errorf("buoy %s has no rollback target: %w", buoy.ID, ErrConflict)
	}
	return buoy.PreviousVersion, nil
}

func CheckGeneration(buoy Buoy, expected int64) error {
	if expected <= 0 || buoy.Generation != expected {
		return fmt.Errorf("buoy generation changed: %w", ErrConflict)
	}
	return nil
}

func CanCloseAlert(survey_mission SurveyMission, openAlerts int) error {
	if openAlerts < 0 {
		return fmt.Errorf("open alert count is invalid: %w", ErrInvalid)
	}
	if openAlerts > 0 || survey_mission.State == "rolling_back" {
		return fmt.Errorf("survey_mission still has unresolved safety work: %w", ErrConflict)
	}
	return nil
}
