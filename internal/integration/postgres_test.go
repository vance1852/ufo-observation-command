package integration

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"ufo-observation-command/internal/db"
	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
	"github.com/google/uuid"
)

func openDatabase(t *testing.T) (*db.Pool, context.Context) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE TABLE audit_events, integrity_incidents, signal_recovery_reports, dive_window_items, dive_windows, mooring_events, recovery_jobs, acoustic_buoys, survey_missions, array_operators, idempotency_keys CASCADE`); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return pool, ctx
}

func insertArrayOperator(t *testing.T, ctx context.Context, pool *db.Pool, role string) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO array_operators(id,name,role) VALUES ($1,$2,$3)`, id, "Test ArrayOperator", role); err != nil {
		t.Fatal(err)
	}
	return id
}

func createWorkflow(t *testing.T, ctx context.Context, svc *service.Service, array_operator string) (service.CreateSurveyMissionResponse, domain.RecoveryJob) {
	t.Helper()
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	svc.WithClock(func() time.Time { return now })
	survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "req-create"}, service.CreateSurveyMissionRequest{
		Code: "PLAN-001", Name: "North river survey", Timezone: "Asia/Shanghai", StartsAt: now, EndsAt: now.Add(24 * time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "N-01", ArrayOpsLane: "A-101", RequiredSuccesses: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ScheduleSurveyMission(ctx, service.RequestMeta{RequestID: "req-schedule"}, survey_mission.SurveyMission.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.ActivateSurveyMission(ctx, service.RequestMeta{RequestID: "req-collect"}, survey_mission.SurveyMission.ID, 2); err != nil {
		t.Fatal(err)
	}
	task, err := svc.CreateRecoveryJob(ctx, service.RequestMeta{RequestID: "req-task"}, repository.RecoveryJobInput{SurveyMissionID: survey_mission.SurveyMission.ID, AcousticBuoyID: survey_mission.AcousticBuoyIDs[0], TaskCode: "S-001", ExpiresAt: now.Add(12 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CompleteRecoveryJob(ctx, service.RequestMeta{RequestID: "req-collect-task"}, task.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.ActivationRecoveryJob(ctx, service.RequestMeta{RequestID: "req-activation"}, repository.ActivationInput{RecoveryJobID: task.ID, To: array_operator, Location: "A-101", RecordedAt: now}, 2); err != nil {
		t.Fatal(err)
	}
	if err := svc.AcceptRecoveryJob(ctx, service.RequestMeta{RequestID: "req-receive"}, repository.ActivationInput{RecoveryJobID: task.ID, To: array_operator, Location: "AcousticBuoy bay", RecordedAt: now}, 3); err != nil {
		t.Fatal(err)
	}
	page, err := svc.ListRecoveryJobs(ctx, 0, 10, survey_mission.SurveyMission.ID, domain.RecoveryJobAccepted)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("task listing = %+v, %v", page, err)
	}
	return survey_mission, page.Items[0]
}

func TestRecoveryJobWorkflowPersistsAcrossOperations(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	survey_mission, task := createWorkflow(t, ctx, svc, array_operator)
	if survey_mission.SurveyMission.Status != domain.SurveyMissionDraft {
		t.Fatalf("created survey_mission status = %s", survey_mission.SurveyMission.Status)
	}
	if task.Status != domain.RecoveryJobAccepted {
		t.Fatalf("task status = %s", task.Status)
	}
	if task.Version != 4 {
		t.Fatalf("task version = %d", task.Version)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE object_id=$1`, task.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 4 {
		t.Fatalf("audit count = %d, want 4", auditCount)
	}
}

func TestSurveyMissionCreationRollsBackWhenASecondAcousticBuoyViolatesConstraint(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "safety_supervisor")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	_, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "req-rollback"}, service.CreateSurveyMissionRequest{
		Code: "PLAN-ROLLBACK", Name: "Rollback test", Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "GOOD", ArrayOpsLane: "A", RequiredSuccesses: 1}, {Code: "BAD", ArrayOpsLane: "B", RequiredSuccesses: 0}},
	})
	if err == nil {
		t.Fatal("invalid second acoustic_buoy was accepted")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM survey_missions WHERE code='PLAN-ROLLBACK'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("survey_mission survived rollback: %d", count)
	}
}

func TestConcurrentRecoveryJobTransitionAllowsOnlyOneVersionWinner(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	repo := repository.NewPostgres(pool)
	survey_mission, task := createWorkflow(t, ctx, svc, array_operator)
	_ = survey_mission
	// The workflow leaves the task at received/version 4. Two workers race to start testing.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- repo.MoveRecoveryJob(ctx, task.ID, domain.RecoveryJobInProgress, 4, time.Now().UTC())
		}()
	}
	wg.Wait()
	close(results)
	var success, conflict int
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, domain.ErrConflict) {
			conflict++
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}

func TestMigrationAndStateSurviveDatabaseReopen(t *testing.T) {
	pool, ctx := openDatabase(t)
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	_, task := createWorkflow(t, ctx, svc, array_operator)
	pool.Close()
	url := os.Getenv("DATABASE_URL")
	reopened, err := db.Open(ctx, url, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	loaded, err := repository.NewPostgres(reopened).GetRecoveryJob(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TaskCode != "S-001" || loaded.Status != domain.RecoveryJobAccepted {
		t.Fatalf("reopened task = %+v", loaded)
	}
}
