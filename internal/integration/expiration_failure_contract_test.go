package integration

import (
	"testing"
	"time"

	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
)

// TestExpiredRecoveryJobStaysProcessableWhenAuditWriteFails reproduces the
// reported defect: when storage rejects the expire audit write, the task must
// remain in a processable stage (it must stay in the pending queue) instead of
// having its lifecycle advanced out of it. The reconciliation must also leave
// other due tasks alone rather than aborting the whole batch.
func TestExpiredRecoveryJobStaysProcessableWhenAuditWriteFails(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "expiry-plan"}, service.CreateSurveyMissionRequest{
		Code: "EXPIRY-FAIL", Name: "ExpiryFail", Timezone: "UTC", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "EXPIRY-FAIL-DEVICE", ArrayOpsLane: "A", RequiredSuccesses: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var taskID string
	if err := pool.QueryRow(ctx, `INSERT INTO recovery_jobs(id,survey_mission_id,acoustic_buoy_id,task_code,status,expires_at,version) VALUES (gen_random_uuid(),$1,$2,'EXPIRY-FAIL-01','queued',$3,1) RETURNING id`, survey_mission.SurveyMission.ID, survey_mission.AcousticBuoyIDs[0], now.Add(-time.Minute)).Scan(&taskID); err != nil {
		t.Fatal(err)
	}

	// Storage rejects every audit_events write (thunderstorm storage outage).
	if _, err := pool.Exec(ctx, `CREATE OR REPLACE FUNCTION _expire_fail_audit() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'storage rejects write'; END; $$ LANGUAGE plpgsql`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE TRIGGER _expire_fail_audit_trigger BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION _expire_fail_audit()`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Exec(ctx, `DROP TRIGGER IF EXISTS _expire_fail_audit_trigger ON audit_events`)
		pool.Exec(ctx, `DROP FUNCTION IF EXISTS _expire_fail_audit()`)
	}()

	result, err := svc.MarkExpiredRecoveryJobs(ctx, now, 10)
	if err != nil {
		t.Fatalf("reconciliation returned error: %v", err)
	}
	if result.Marked != 0 {
		t.Fatalf("task was advanced while the audit write failed: marked=%d", result.Marked)
	}
	if result.Failed != 1 {
		t.Fatalf("failed task not counted: result=%+v", result)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM recovery_jobs WHERE id=$1`, taskID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "queued" {
		t.Fatalf("cleanup failed but task advanced to %s (left the processable queue)", status)
	}

	var audits int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE object_id=$1`, taskID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 0 {
		t.Fatalf("partial audit trail persisted without the state change: audits=%d", audits)
	}
}
