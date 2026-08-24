package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"ufo-observation-command/internal/arrayops"
	"ufo-observation-command/internal/service"
)

func newAcousticControl() (*service.AcousticControl, *arrayops.Store) {
	store := arrayops.NewStore()
	return service.NewAcousticControl(store), store
}

func signedCalibrationBundle() arrayops.CalibrationBundle {
	return arrayops.CalibrationBundle{ID: "calibration_bundle-1", TenantID: "tenant-a", Version: "2.4.0", Digest: "sha256:verified", BuoyClasses: []string{"edge-v2"}, Labels: map[string]string{"ring": "pilot_dive"}, Signed: true}
}

func TestIdempotencySeparatesTenants(t *testing.T) {
	control, _ := newAcousticControl()
	if stored, err := control.SaveIdempotentResponse("tenant-a", "POST", "/survey_missions", "request-7", []byte(`{"id":"a"}`)); err != nil || !stored {
		t.Fatalf("first response stored=%v err=%v", stored, err)
	}
	if stored, err := control.SaveIdempotentResponse("tenant-b", "POST", "/survey_missions", "request-7", []byte(`{"id":"b"}`)); err != nil || !stored {
		t.Fatalf("second tenant stored=%v err=%v", stored, err)
	}
	value, ok, err := control.LoadIdempotentResponse("tenant-b", "POST", "/survey_missions", "request-7")
	if err != nil || !ok || string(value) != `{"id":"b"}` {
		t.Fatalf("tenant-b response=%s ok=%v err=%v", value, ok, err)
	}
}

func TestIdempotencySeparatesRoutes(t *testing.T) {
	control, _ := newAcousticControl()
	for _, path := range []string{"/survey_missions", "/calibration_bundles"} {
		stored, err := control.SaveIdempotentResponse("tenant-a", "POST", path, "request-8", []byte(path))
		if err != nil || !stored {
			t.Fatalf("path=%s stored=%v err=%v", path, stored, err)
		}
	}
	value, ok, err := control.LoadIdempotentResponse("tenant-a", "POST", "/calibration_bundles", "request-8")
	if err != nil || !ok || string(value) != "/calibration_bundles" {
		t.Fatalf("route response=%s ok=%v err=%v", value, ok, err)
	}
}

func TestCallbackIdentityIncludesCalibrationBundle(t *testing.T) {
	control, _ := newAcousticControl()
	base := arrayops.Callback{TenantID: "tenant-a", BuoyID: "buoy-1", CalibrationBundleID: "calibration_bundle-1", EventID: "callback-9", Status: "installed"}
	first, err := control.RegisterCallback(base)
	if err != nil || !first {
		t.Fatalf("first callback new=%v err=%v", first, err)
	}
	base.CalibrationBundleID = "calibration_bundle-2"
	second, err := control.RegisterCallback(base)
	if err != nil || !second {
		t.Fatalf("second calibration_bundle callback new=%v err=%v", second, err)
	}
}

func TestEnrollmentRejectsCrossTenantCalibrationBundle(t *testing.T) {
	control, store := newAcousticControl()
	calibration_bundle := signedCalibrationBundle()
	buoy := arrayops.Buoy{ID: "buoy-1", TenantID: "tenant-b", Class: "edge-v2"}
	if err := control.EnrollBuoy(calibration_bundle, buoy); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("enrollment err=%v", err)
	}
	if _, ok := store.Buoy(buoy.ID); ok {
		t.Fatal("cross-tenant buoy was persisted")
	}
}

func TestEnrollmentRejectsUnsupportedClass(t *testing.T) {
	control, store := newAcousticControl()
	calibration_bundle := signedCalibrationBundle()
	buoy := arrayops.Buoy{ID: "buoy-2", TenantID: calibration_bundle.TenantID, Class: "legacy-v1"}
	if err := control.EnrollBuoy(calibration_bundle, buoy); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("enrollment err=%v", err)
	}
	if _, ok := store.CalibrationBundle(calibration_bundle.ID); ok {
		t.Fatal("calibration_bundle was persisted before compatibility passed")
	}
}

func TestEnrollmentRequiresVerifiedSignature(t *testing.T) {
	control, store := newAcousticControl()
	calibration_bundle := signedCalibrationBundle()
	calibration_bundle.Signed = false
	buoy := arrayops.Buoy{ID: "buoy-3", TenantID: calibration_bundle.TenantID, Class: "edge-v2"}
	if err := control.EnrollBuoy(calibration_bundle, buoy); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("enrollment err=%v", err)
	}
	if _, ok := store.Buoy(buoy.ID); ok {
		t.Fatal("buoy was enrolled with an unverified calibration_bundle")
	}
}

func TestApprovalPinsCalibrationBundleDigest(t *testing.T) {
	control, _ := newAcousticControl()
	calibration_bundle := signedCalibrationBundle()
	survey_mission := arrayops.SurveyMission{ID: "survey_mission-1", TenantID: calibration_bundle.TenantID, CalibrationBundleID: calibration_bundle.ID, State: "running", Version: 1}
	if err := control.ApproveSurveyMission(survey_mission, calibration_bundle); err != nil {
		t.Fatal(err)
	}
	snapshot, ok := control.SnapshotSurveyMission(survey_mission.ID)
	if !ok || snapshot.ApprovedDigest != calibration_bundle.Digest {
		t.Fatalf("approved digest=%q ok=%v", snapshot.ApprovedDigest, ok)
	}
}

func TestPausedSurveyMissionCannotDispatch(t *testing.T) {
	control, store := newAcousticControl()
	survey_mission := arrayops.SurveyMission{ID: "survey_mission-2", TenantID: "tenant-a", CalibrationBundleID: "calibration_bundle-1", ApprovedDigest: "sha256:ok", State: "paused", Version: 4}
	if err := store.PutSurveyMission(survey_mission); err != nil {
		t.Fatal(err)
	}
	if err := control.Dispatch(survey_mission.ID, survey_mission.Version); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("dispatch err=%v", err)
	}
	snapshot, _ := store.SurveyMission(survey_mission.ID)
	if snapshot.State != "paused" || snapshot.Version != survey_mission.Version {
		t.Fatalf("survey_mission changed=%+v", snapshot)
	}
}

func TestPromotionRequiresHealthyPilotDives(t *testing.T) {
	control, store := newAcousticControl()
	survey_mission := arrayops.SurveyMission{ID: "survey_mission-3", TenantID: "tenant-a", State: "running", RequiredHealthy: 3, Healthy: 2, Version: 1}
	if err := store.PutSurveyMission(survey_mission); err != nil {
		t.Fatal(err)
	}
	if err := control.Promote(survey_mission.ID, survey_mission.Version); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("promotion err=%v", err)
	}
	snapshot, _ := store.SurveyMission(survey_mission.ID)
	if snapshot.State != "running" {
		t.Fatalf("survey_mission state=%s", snapshot.State)
	}
}

func TestPromotionRejectsFailedPilotDive(t *testing.T) {
	control, store := newAcousticControl()
	survey_mission := arrayops.SurveyMission{ID: "survey_mission-4", TenantID: "tenant-a", State: "running", RequiredHealthy: 2, Healthy: 2, Failed: 1, Version: 6}
	if err := store.PutSurveyMission(survey_mission); err != nil {
		t.Fatal(err)
	}
	if err := control.Promote(survey_mission.ID, survey_mission.Version); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("promotion err=%v", err)
	}
}

func TestRollbackUsesPreviousVersion(t *testing.T) {
	control, store := newAcousticControl()
	buoy := arrayops.Buoy{ID: "buoy-4", TenantID: "tenant-a", CurrentVersion: "2.4.0", PreviousVersion: "2.3.7", Generation: 9}
	if err := store.PutBuoy(buoy); err != nil {
		t.Fatal(err)
	}
	target, err := control.RollbackTarget(buoy.ID, buoy.Generation)
	if err != nil || target != buoy.PreviousVersion {
		t.Fatalf("rollback target=%q err=%v", target, err)
	}
}

func TestRollbackRejectsStaleGeneration(t *testing.T) {
	control, store := newAcousticControl()
	buoy := arrayops.Buoy{ID: "buoy-5", TenantID: "tenant-a", CurrentVersion: "2.4.0", PreviousVersion: "2.3.7", Generation: 12}
	if err := store.PutBuoy(buoy); err != nil {
		t.Fatal(err)
	}
	if _, err := control.RollbackTarget(buoy.ID, 11); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("rollback err=%v", err)
	}
}

func TestSafetyAlertStaysOpenWhileAlertsRemain(t *testing.T) {
	control, store := newAcousticControl()
	survey_mission := arrayops.SurveyMission{ID: "survey_mission-5", TenantID: "tenant-a", State: "paused"}
	if err := store.PutSurveyMission(survey_mission); err != nil {
		t.Fatal(err)
	}
	if err := control.CloseSafetyAlert(survey_mission.ID, 2); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("close err=%v", err)
	}
}

func TestSafetyAlertStaysOpenDuringRollback(t *testing.T) {
	control, store := newAcousticControl()
	survey_mission := arrayops.SurveyMission{ID: "survey_mission-6", TenantID: "tenant-a", State: "rolling_back"}
	if err := store.PutSurveyMission(survey_mission); err != nil {
		t.Fatal(err)
	}
	if err := control.CloseSafetyAlert(survey_mission.ID, 0); !errors.Is(err, arrayops.ErrConflict) {
		t.Fatalf("close err=%v", err)
	}
}

func TestCalibrationBundleSnapshotDoesNotShareLabels(t *testing.T) {
	control, store := newAcousticControl()
	calibration_bundle := signedCalibrationBundle()
	if err := store.PutCalibrationBundle(calibration_bundle); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := control.SnapshotCalibrationBundle(calibration_bundle.ID)
	snapshot.Labels["ring"] = "fleet"
	again, _ := control.SnapshotCalibrationBundle(calibration_bundle.ID)
	if again.Labels["ring"] != "pilot_dive" || calibration_bundle.Labels["ring"] != "pilot_dive" {
		t.Fatalf("labels leaked: stored=%v input=%v", again.Labels, calibration_bundle.Labels)
	}
}

func TestCalibrationBundleSnapshotDoesNotShareClasses(t *testing.T) {
	control, store := newAcousticControl()
	calibration_bundle := signedCalibrationBundle()
	if err := store.PutCalibrationBundle(calibration_bundle); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := control.SnapshotCalibrationBundle(calibration_bundle.ID)
	snapshot.BuoyClasses[0] = "legacy-v1"
	again, _ := control.SnapshotCalibrationBundle(calibration_bundle.ID)
	if again.BuoyClasses[0] != "edge-v2" || calibration_bundle.BuoyClasses[0] != "edge-v2" {
		t.Fatalf("classes leaked: stored=%v input=%v", again.BuoyClasses, calibration_bundle.BuoyClasses)
	}
}

func TestSurveyMissionSnapshotDoesNotShareBuoys(t *testing.T) {
	control, store := newAcousticControl()
	survey_mission := arrayops.SurveyMission{ID: "survey_mission-7", TenantID: "tenant-a", BuoyIDs: []string{"buoy-1", "buoy-2"}}
	if err := store.PutSurveyMission(survey_mission); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := control.SnapshotSurveyMission(survey_mission.ID)
	snapshot.BuoyIDs[0] = "buoy-x"
	again, _ := control.SnapshotSurveyMission(survey_mission.ID)
	if again.BuoyIDs[0] != "buoy-1" || survey_mission.BuoyIDs[0] != "buoy-1" {
		t.Fatalf("buoys leaked: stored=%v input=%v", again.BuoyIDs, survey_mission.BuoyIDs)
	}
}

func TestRestoredLabelsAreWritableAndIsolated(t *testing.T) {
	control, _ := newAcousticControl()
	snapshot := map[string]string{"ring": "pilot_dive"}
	restored := control.RestoreCalibrationBundleLabels(snapshot)
	restored["region"] = "north"
	restored["ring"] = "fleet"
	if snapshot["ring"] != "pilot_dive" || snapshot["region"] != "" {
		t.Fatalf("snapshot was mutated: %v", snapshot)
	}
}

func TestRetryWaitStopsOnCancellation(t *testing.T) {
	control, _ := newAcousticControl()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	err := control.WaitRetry(ctx, time.Second)
	if !errors.Is(err, context.Canceled) || time.Since(started) > 200*time.Millisecond {
		t.Fatalf("wait err=%v elapsed=%s", err, time.Since(started))
	}
}

func TestDownloadContextKeepsParentCancellation(t *testing.T) {
	control, _ := newAcousticControl()
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	err := control.RunDownload(parent, time.Minute, func(ctx context.Context) error { return ctx.Err() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("download err=%v", err)
	}
}

func TestWorkerLeaseRejectsForeignRelease(t *testing.T) {
	control, _ := newAcousticControl()
	now := time.Now()
	if !control.AcquireWorker("worker-a", now, time.Minute) {
		t.Fatal("worker-a did not acquire lease")
	}
	if control.ReleaseWorker("worker-b") {
		t.Fatal("worker-b released worker-a lease")
	}
	if control.AcquireWorker("worker-b", now.Add(time.Second), time.Minute) {
		t.Fatal("worker-b acquired an active foreign lease")
	}
}

func TestExpiredWorkerLeaseCannotRenew(t *testing.T) {
	control, _ := newAcousticControl()
	now := time.Now()
	if !control.AcquireWorker("worker-a", now, time.Second) {
		t.Fatal("worker-a did not acquire lease")
	}
	if control.RenewWorker("worker-a", now.Add(time.Second), time.Minute) {
		t.Fatal("expired lease was renewed")
	}
	if !control.AcquireWorker("worker-b", now.Add(time.Second), time.Minute) {
		t.Fatal("worker-b could not take expired lease")
	}
}

func TestConcurrentLaneCapacityDoesNotOversubscribe(t *testing.T) {
	control, _ := newAcousticControl()
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			results <- control.ReserveLane("tenant-a", "north", 1)
		}()
	}
	ready.Wait()
	close(start)
	var successes int
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 || control.LaneUsage("tenant-a", "north") != 1 {
		t.Fatalf("successes=%d used=%d", successes, control.LaneUsage("tenant-a", "north"))
	}
}

func TestSessionCannotCrossTenant(t *testing.T) {
	control, store := newAcousticControl()
	now := time.Now()
	if err := store.PutSession(arrayops.Session{Token: "token-1", TenantID: "tenant-a", UserID: "operator-1", Role: "release_manager", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := control.Authenticate("token-1", "tenant-b", "release_manager", now); !errors.Is(err, arrayops.ErrUnauthorized) {
		t.Fatalf("authentication err=%v", err)
	}
}

func TestExpiredSessionCannotAuthorize(t *testing.T) {
	control, store := newAcousticControl()
	now := time.Now()
	if err := store.PutSession(arrayops.Session{Token: "token-2", TenantID: "tenant-a", UserID: "operator-2", Role: "release_manager", ExpiresAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := control.Authenticate("token-2", "tenant-a", "release_manager", now); !errors.Is(err, arrayops.ErrUnauthorized) {
		t.Fatalf("authentication err=%v", err)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	control, store := newAcousticControl()
	now := time.Now()
	if err := store.PutSession(arrayops.Session{Token: "token-3", TenantID: "tenant-a", UserID: "operator-3", Role: "release_manager", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if !control.Logout("token-3", now) {
		t.Fatal("logout did not revoke token")
	}
	if err := control.Authenticate("token-3", "tenant-a", "release_manager", now); !errors.Is(err, arrayops.ErrUnauthorized) {
		t.Fatalf("authentication err=%v", err)
	}
}

func TestRoleChangeUpdatesAuthorization(t *testing.T) {
	control, store := newAcousticControl()
	now := time.Now()
	if err := store.PutSession(arrayops.Session{Token: "token-4", TenantID: "tenant-a", UserID: "operator-4", Role: "release_manager", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := control.ChangeRole("token-4", "auditor"); err != nil {
		t.Fatal(err)
	}
	if err := control.Authenticate("token-4", "tenant-a", "release_manager", now); !errors.Is(err, arrayops.ErrUnauthorized) {
		t.Fatalf("old role authentication err=%v", err)
	}
	if err := control.Authenticate("token-4", "tenant-a", "auditor", now); err != nil {
		t.Fatalf("new role authentication err=%v", err)
	}
}

func TestBuoyQueryDoesNotCrossTenant(t *testing.T) {
	control, store := newAcousticControl()
	for _, buoy := range []arrayops.Buoy{{ID: "buoy-a", TenantID: "tenant-a", Class: "edge-v2"}, {ID: "buoy-b", TenantID: "tenant-b", Class: "edge-v2"}} {
		if err := store.PutBuoy(buoy); err != nil {
			t.Fatal(err)
		}
	}
	page := control.Buoys(arrayops.Query{TenantID: "tenant-a", Limit: 10})
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].TenantID != "tenant-a" {
		t.Fatalf("page=%+v", page)
	}
}

func TestBuoyQueryTotalUsesClassFilter(t *testing.T) {
	control, store := newAcousticControl()
	for _, buoy := range []arrayops.Buoy{{ID: "buoy-a", TenantID: "tenant-a", Class: "edge-v2"}, {ID: "buoy-b", TenantID: "tenant-a", Class: "gateway-v3"}} {
		if err := store.PutBuoy(buoy); err != nil {
			t.Fatal(err)
		}
	}
	page := control.Buoys(arrayops.Query{TenantID: "tenant-a", Class: "edge-v2", Limit: 10})
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Class != "edge-v2" {
		t.Fatalf("page=%+v", page)
	}
}

func TestSignatureErrorKeepsHTTPClassification(t *testing.T) {
	control, _ := newAcousticControl()
	err := arrayops.WrapOperation("publish", &arrayops.SignatureError{Digest: "sha256:bad", Cause: arrayops.ErrConflict})
	status, code := control.ErrorResponse(err)
	if status != 422 || code != "signature_invalid" {
		t.Fatalf("status=%d code=%s err=%v", status, code, err)
	}
}
