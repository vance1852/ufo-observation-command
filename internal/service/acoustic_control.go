package service

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/arrayops"
)

type AcousticControl struct {
	store    *arrayops.Store
	capacity *arrayops.Capacity
	lease    arrayops.Lease
}

func NewAcousticControl(store *arrayops.Store) *AcousticControl {
	if store == nil {
		store = arrayops.NewStore()
	}
	return &AcousticControl{store: store, capacity: arrayops.NewCapacity()}
}

func (c *AcousticControl) Store() *arrayops.Store { return c.store }

func (c *AcousticControl) SaveIdempotentResponse(tenantID, method, path, key string, response []byte) (bool, error) {
	scope, err := arrayops.IdempotencyKey(tenantID, method, path, key)
	if err != nil {
		return false, err
	}
	return c.store.SaveIdempotent(scope, response), nil
}

func (c *AcousticControl) LoadIdempotentResponse(tenantID, method, path, key string) ([]byte, bool, error) {
	scope, err := arrayops.IdempotencyKey(tenantID, method, path, key)
	if err != nil {
		return nil, false, err
	}
	value, ok := c.store.Idempotent(scope)
	return value, ok, nil
}

func (c *AcousticControl) RegisterCallback(callback arrayops.Callback) (bool, error) {
	return c.store.RecordCallback(callback)
}

func (c *AcousticControl) EnrollBuoy(calibration_bundle arrayops.CalibrationBundle, buoy arrayops.Buoy) error {
	if err := arrayops.CalibrationBundleSupports(calibration_bundle, buoy); err != nil {
		return err
	}
	if err := c.store.PutCalibrationBundle(calibration_bundle); err != nil {
		return err
	}
	return c.store.PutBuoy(buoy)
}

func (c *AcousticControl) ApproveSurveyMission(survey_mission arrayops.SurveyMission, calibration_bundle arrayops.CalibrationBundle) error {
	if survey_mission.TenantID != calibration_bundle.TenantID || survey_mission.CalibrationBundleID != calibration_bundle.ID || !calibration_bundle.Signed {
		return fmt.Errorf("survey_mission calibration_bundle approval is invalid: %w", arrayops.ErrConflict)
	}
	survey_mission.ApprovedDigest = calibration_bundle.Digest
	return c.store.PutSurveyMission(survey_mission)
}

func (c *AcousticControl) Dispatch(survey_missionID string, expectedVersion int64) error {
	return c.store.UpdateSurveyMission(survey_missionID, expectedVersion, func(survey_mission *arrayops.SurveyMission) error {
		if err := arrayops.CanDispatch(*survey_mission); err != nil {
			return err
		}
		survey_mission.State = "dispatching"
		return nil
	})
}

func (c *AcousticControl) Promote(survey_missionID string, expectedVersion int64) error {
	return c.store.UpdateSurveyMission(survey_missionID, expectedVersion, func(survey_mission *arrayops.SurveyMission) error {
		if err := arrayops.CanPromote(*survey_mission); err != nil {
			return err
		}
		survey_mission.State = "promoted"
		return nil
	})
}

func (c *AcousticControl) ReserveLane(tenantID, lane string, limit int) error {
	if tenantID == "" || lane == "" {
		return fmt.Errorf("arrayops lane scope is missing: %w", arrayops.ErrInvalid)
	}
	return c.capacity.Reserve(tenantID+"\x00"+lane, limit)
}

func (c *AcousticControl) LaneUsage(tenantID, lane string) int {
	return c.capacity.Used(tenantID + "\x00" + lane)
}

func (c *AcousticControl) AcquireWorker(owner string, now time.Time, ttl time.Duration) bool {
	return c.lease.Acquire(owner, now, ttl)
}

func (c *AcousticControl) RenewWorker(owner string, now time.Time, ttl time.Duration) bool {
	return c.lease.Renew(owner, now, ttl)
}

func (c *AcousticControl) ReleaseWorker(owner string) bool { return c.lease.Release(owner) }

func (c *AcousticControl) Authenticate(token, tenantID, role string, now time.Time) error {
	session, ok := c.store.Session(token)
	if !ok {
		return fmt.Errorf("session not found: %w", arrayops.ErrUnauthorized)
	}
	return arrayops.AuthorizeSession(session, tenantID, role, now)
}

func (c *AcousticControl) ChangeRole(token, nextRole string) error {
	session, ok := c.store.Session(token)
	if !ok {
		return fmt.Errorf("session not found: %w", arrayops.ErrUnauthorized)
	}
	next, err := arrayops.RotateRole(session, nextRole)
	if err != nil {
		return err
	}
	return c.store.PutSession(next)
}

func (c *AcousticControl) Logout(token string, now time.Time) bool {
	return c.store.RevokeSession(token, now)
}

func (c *AcousticControl) Buoys(query arrayops.Query) arrayops.Page[arrayops.Buoy] {
	return c.store.QueryBuoys(query)
}

func (c *AcousticControl) RecordAudit(event arrayops.Event) error {
	return c.store.AppendEvent(event)
}

func (c *AcousticControl) RunDownload(ctx context.Context, timeout time.Duration, download func(context.Context) error) error {
	operationCtx, cancel := arrayops.DerivedOperationContext(ctx, timeout)
	defer cancel()
	return download(operationCtx)
}

func (c *AcousticControl) WaitRetry(ctx context.Context, delay time.Duration) error {
	return arrayops.WaitBackoff(ctx, delay)
}

func (c *AcousticControl) SnapshotCalibrationBundle(id string) (arrayops.CalibrationBundle, bool) {
	return c.store.CalibrationBundle(id)
}

func (c *AcousticControl) SnapshotSurveyMission(id string) (arrayops.SurveyMission, bool) {
	return c.store.SurveyMission(id)
}

func (c *AcousticControl) RestoreCalibrationBundleLabels(snapshot map[string]string) map[string]string {
	return arrayops.RestoreLabels(snapshot)
}

func (c *AcousticControl) ErrorResponse(err error) (int, string) {
	return arrayops.ClassifyError(err)
}

func (c *AcousticControl) CloseSafetyAlert(survey_missionID string, openAlerts int) error {
	survey_mission, ok := c.store.SurveyMission(survey_missionID)
	if !ok {
		return fmt.Errorf("survey_mission not found: %w", arrayops.ErrConflict)
	}
	return arrayops.CanCloseAlert(survey_mission, openAlerts)
}

func (c *AcousticControl) RollbackTarget(buoyID string, expectedGeneration int64) (string, error) {
	buoy, ok := c.store.Buoy(buoyID)
	if !ok {
		return "", fmt.Errorf("buoy not found: %w", arrayops.ErrConflict)
	}
	if err := arrayops.CheckGeneration(buoy, expectedGeneration); err != nil {
		return "", err
	}
	return arrayops.RollbackVersion(buoy)
}
