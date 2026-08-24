package integration

import (
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
)

func TestSurveyMissionProgressCountsEachAcousticBuoyRequirementOnce(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "progress"}, service.CreateSurveyMissionRequest{
		Code: "PROGRESS", Name: "Progress", Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "P-1", ArrayOpsLane: "A", RequiredSuccesses: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewPostgres(pool)
	for _, code := range []string{"P-S1", "P-S2"} {
		if _, err := repo.CreateRecoveryJob(ctx, repository.RecoveryJobInput{SurveyMissionID: survey_mission.SurveyMission.ID, AcousticBuoyID: survey_mission.AcousticBuoyIDs[0], TaskCode: code, ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	progress, err := repo.SurveyMissionProgress(ctx, survey_mission.SurveyMission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if progress.AcousticBuoys != 1 || progress.Required != 2 {
		t.Fatalf("progress=%+v", progress)
	}
}

func TestComplianceReportContainsOnlyItsSurveyMissionRecoveryJobs(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	create := func(code string) service.CreateSurveyMissionResponse {
		survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: code}, service.CreateSurveyMissionRequest{
			Code: code, Name: code, Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
			AcousticBuoys: []repository.AcousticBuoyInput{{Code: code + "-MANAGED_DEVICE", ArrayOpsLane: "A", RequiredSuccesses: 1}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return survey_mission
	}
	first, second := create("REPORT-A"), create("REPORT-B")
	repo := repository.NewPostgres(pool)
	for _, item := range []struct{ survey_mission, acoustic_buoy, code string }{{first.SurveyMission.ID, first.AcousticBuoyIDs[0], "REPORT-S1"}, {second.SurveyMission.ID, second.AcousticBuoyIDs[0], "REPORT-S2"}} {
		if _, err := repo.CreateRecoveryJob(ctx, repository.RecoveryJobInput{SurveyMissionID: item.survey_mission, AcousticBuoyID: item.acoustic_buoy, TaskCode: item.code, ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	report, err := repo.ComplianceReport(ctx, first.SurveyMission.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Expiring) != 1 || report.Expiring[0].SurveyMissionID != first.SurveyMission.ID {
		t.Fatalf("expiring=%+v", report.Expiring)
	}
}

func TestDiveWindowRejectsRecoveryJobsThatAreNotReadyForInProgress(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "dive_window"}, service.CreateSurveyMissionRequest{
		Code: "ROUND-READY", Name: "DiveWindow", Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "B-1", ArrayOpsLane: "A", RequiredSuccesses: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := svc.CreateRecoveryJob(ctx, service.RequestMeta{RequestID: "task"}, repository.RecoveryJobInput{SurveyMissionID: survey_mission.SurveyMission.ID, AcousticBuoyID: survey_mission.AcousticBuoyIDs[0], TaskCode: "B-S1", ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateDiveWindow(ctx, service.RequestMeta{RequestID: "dive_window-create"}, repository.DiveWindowInput{Code: "B-1", Method: "daily-acoustic_buoy", Capacity: 1}, []string{task.ID})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("err=%v", err)
	}
	var dive_windows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dive_windows WHERE code='B-1'`).Scan(&dive_windows); err != nil {
		t.Fatal(err)
	}
	if dive_windows != 0 {
		t.Fatalf("partial dive_window count=%d", dive_windows)
	}
}

func TestDiveWindowAndAssignmentRejectSkippedStates(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	repo := repository.NewPostgres(pool)
	dive_windowID, err := repo.CreateDiveWindow(ctx, repository.DiveWindowInput{Code: "STATE-B", Method: "daily-acoustic_buoy", Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteDiveWindow(ctx, dive_windowID, 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("dive_window err=%v", err)
	}
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repo)
	now := time.Now().UTC()
	survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "assignment"}, service.CreateSurveyMissionRequest{
		Code: "ASSIGN-STATE", Name: "Assignment", Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "A-1", ArrayOpsLane: "A", RequiredSuccesses: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assignment := repository.NewAssignment(survey_mission.SurveyMission.ID, survey_mission.AcousticBuoyIDs[0], array_operator, now, now.Add(time.Hour))
	if err := repo.CreateAssignment(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	if err := repo.AdvanceAssignment(ctx, assignment.ID, "completed", 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("assignment err=%v", err)
	}
}

func TestRecoveryJobRejectsAcousticBuoyFromAnotherSurveyMission(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	create := func(code string) service.CreateSurveyMissionResponse {
		survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: code}, service.CreateSurveyMissionRequest{
			Code: code, Name: code, Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
			AcousticBuoys: []repository.AcousticBuoyInput{{Code: code + "-MANAGED_DEVICE", ArrayOpsLane: "A", RequiredSuccesses: 1}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return survey_mission
	}
	first := create("PLAN-A")
	second := create("PLAN-B")
	_, err := svc.CreateRecoveryJob(ctx, service.RequestMeta{RequestID: "mismatch"}, repository.RecoveryJobInput{
		SurveyMissionID: first.SurveyMission.ID, AcousticBuoyID: second.AcousticBuoyIDs[0], TaskCode: "MISMATCH-01", ExpiresAt: now.Add(time.Hour),
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error=%v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM recovery_jobs WHERE task_code='MISMATCH-01'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("mismatched task persisted: %d", count)
	}
}

func TestExpiredRecoveryJobReconciliationWritesAudit(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	survey_mission, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "expiry-survey_mission"}, service.CreateSurveyMissionRequest{
		Code: "EXPIRY-PLAN", Name: "Expiry", Timezone: "UTC", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "EXPIRY-MANAGED_DEVICE", ArrayOpsLane: "A", RequiredSuccesses: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var taskID string
	if err := pool.QueryRow(ctx, `INSERT INTO recovery_jobs(id,survey_mission_id,acoustic_buoy_id,task_code,status,expires_at,version) VALUES (gen_random_uuid(),$1,$2,'EXPIRED-01','queued',$3,1) RETURNING id`, survey_mission.SurveyMission.ID, survey_mission.AcousticBuoyIDs[0], now.Add(-time.Minute)).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	result, err := svc.MarkExpiredRecoveryJobs(ctx, now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Marked != 1 || result.Failed != 0 {
		t.Fatalf("result=%+v", result)
	}
	var status string
	var audits int
	if err := pool.QueryRow(ctx, `SELECT status FROM recovery_jobs WHERE id=$1`, taskID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE object_id=$1 AND action='expire'`, taskID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if status != string(domain.RecoveryJobRejected) || audits != 1 {
		t.Fatalf("status=%s audits=%d", status, audits)
	}
}
