package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/httpapi"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
)

func apiRequest(t *testing.T, handler http.Handler, method, path string, body any, array_operatorID string, wantStatus int, target any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "http-contract")
	if array_operatorID != "" {
		req.Header.Set("X-ArrayOperator-ID", array_operatorID)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, res.Code, wantStatus, res.Body.String())
	}
	if res.Header().Get("X-Request-ID") != "http-contract" {
		t.Fatalf("%s %s request id=%q", method, path, res.Header().Get("X-Request-ID"))
	}
	if target != nil {
		if err := json.Unmarshal(res.Body.Bytes(), target); err != nil {
			t.Fatalf("%s %s decode response: %v body=%s", method, path, err, res.Body.String())
		}
	}
}

func createArrayOperatorHTTP(t *testing.T, handler http.Handler, name string, role domain.ArrayOperatorRole) domain.ArrayOperator {
	t.Helper()
	var array_operator domain.ArrayOperator
	apiRequest(t, handler, http.MethodPost, "/v1/array_operators", map[string]any{"name": name, "role": role}, "", http.StatusCreated, &array_operator)
	if array_operator.ID == "" || array_operator.Role != role {
		t.Fatalf("array_operator=%+v", array_operator)
	}
	return array_operator
}

func TestHTTPWorkflowCoversPublicBackendRoutes(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	now := time.Now().UTC().Truncate(time.Second)
	svc := service.New(repository.NewPostgres(pool)).WithClock(func() time.Time { return now })
	handler := httpapi.New(svc, pool.Ping).Handler()

	apiRequest(t, handler, http.MethodGet, "/healthz", nil, "", http.StatusOK, nil)
	apiRequest(t, handler, http.MethodGet, "/readyz", nil, "", http.StatusOK, nil)

	field := createArrayOperatorHTTP(t, handler, "AcousticBuoyOperator Lin", domain.RoleAcousticBuoyOperator)
	installationOperator := createArrayOperatorHTTP(t, handler, "InstallationOperator Zhao", domain.RoleInstallationOperator)
	reviewer := createArrayOperatorHTTP(t, handler, "Safety Reviewer Chen", domain.RoleQualityReviewer)
	apiRequest(t, handler, http.MethodPost, "/v1/array_operators/"+field.ID+"/rename", map[string]any{"name": "Senior AcousticBuoyOperator Lin"}, field.ID, http.StatusOK, nil)
	apiRequest(t, handler, http.MethodGet, "/v1/array_operators?role=acoustic_buoy_operator", nil, "", http.StatusOK, nil)

	request := map[string]any{
		"code": "HTTP-PLAN", "name": "HTTP workflow", "timezone": "UTC",
		"starts_at": now.Add(-time.Minute), "ends_at": now.Add(24 * time.Hour), "created_by": field.ID,
		"acoustic_buoys": []map[string]any{{"code": "MANAGED_DEVICE-01", "arrayops_lane": "A-101", "required_successes": 2}},
	}
	var survey_mission service.CreateSurveyMissionResponse
	apiRequest(t, handler, http.MethodPost, "/v1/cases", request, field.ID, http.StatusCreated, &survey_mission)
	if survey_mission.SurveyMission.ID == "" || len(survey_mission.AcousticBuoyIDs) != 1 {
		t.Fatalf("survey_mission=%+v", survey_mission)
	}
	survey_missionID, acoustic_buoyID := survey_mission.SurveyMission.ID, survey_mission.AcousticBuoyIDs[0]
	apiRequest(t, handler, http.MethodGet, "/v1/cases?search=HTTP", nil, "", http.StatusOK, nil)
	apiRequest(t, handler, http.MethodGet, "/v1/cases/"+survey_missionID+"/acoustic_buoys", nil, "", http.StatusOK, nil)
	apiRequest(t, handler, http.MethodPost, "/v1/cases/"+survey_missionID+"/schedule", map[string]any{"version": 1}, field.ID, http.StatusOK, nil)
	apiRequest(t, handler, http.MethodPost, "/v1/cases/"+survey_missionID+"/activate", map[string]any{"version": 2}, field.ID, http.StatusOK, nil)

	var assignment domain.Assignment
	apiRequest(t, handler, http.MethodPost, "/v1/assignments", map[string]any{
		"survey_mission_id": survey_missionID, "acoustic_buoy_id": acoustic_buoyID, "array_operator_id": field.ID,
		"starts_at": now.Add(-time.Minute), "ends_at": now.Add(time.Hour),
	}, field.ID, http.StatusCreated, &assignment)
	apiRequest(t, handler, http.MethodPost, "/v1/assignments/"+assignment.ID+"/advance", map[string]any{"status": "active", "version": 1}, field.ID, http.StatusOK, nil)
	apiRequest(t, handler, http.MethodPost, "/v1/assignments/"+assignment.ID+"/advance", map[string]any{"status": "completed", "version": 2}, field.ID, http.StatusOK, nil)

	createRecoveryJob := func(code string) domain.RecoveryJob {
		var task domain.RecoveryJob
		apiRequest(t, handler, http.MethodPost, "/v1/recovery_jobs", map[string]any{
			"survey_mission_id": survey_missionID, "acoustic_buoy_id": acoustic_buoyID, "task_code": code, "expires_at": now.Add(12 * time.Hour),
		}, field.ID, http.StatusCreated, &task)
		apiRequest(t, handler, http.MethodPost, "/v1/recovery_jobs/"+task.ID+"/complete", map[string]any{"version": 1}, field.ID, http.StatusOK, nil)
		apiRequest(t, handler, http.MethodPost, "/v1/recovery_jobs/"+task.ID+"/activation", map[string]any{
			"to_operator": field.ID, "location": "A-101", "recorded_at": now, "version": 2,
		}, field.ID, http.StatusOK, nil)
		apiRequest(t, handler, http.MethodPost, "/v1/recovery_jobs/"+task.ID+"/accept", map[string]any{
			"from_operator": field.ID, "to_operator": installationOperator.ID, "location": "East acoustic_buoy station", "recorded_at": now.Add(time.Minute), "version": 3,
		}, installationOperator.ID, http.StatusOK, nil)
		return task
	}

	first := createRecoveryJob("HTTP-TASK-01")
	apiRequest(t, handler, http.MethodGet, "/v1/recovery_jobs?survey_mission_id="+survey_missionID+"&status=accepted", nil, "", http.StatusOK, nil)
	var dive_window map[string]string
	apiRequest(t, handler, http.MethodPost, "/v1/dive_windows", map[string]any{
		"code": "HTTP-ROUND-01", "method": "evening-acoustic_buoy", "capacity": 1, "recovery_job_ids": []string{first.ID},
	}, installationOperator.ID, http.StatusCreated, &dive_window)
	dive_windowID := dive_window["id"]
	apiRequest(t, handler, http.MethodPost, "/v1/dive_windows/"+dive_windowID+"/start?version=1", nil, installationOperator.ID, http.StatusOK, nil)
	var signal_recovery_report map[string]string
	apiRequest(t, handler, http.MethodPost, "/v1/signal_recovery_report", map[string]any{
		"recovery_job_id": first.ID, "dive_window_id": dive_windowID, "recorded_by": installationOperator.ID,
		"risk_score": 3.5, "scale": "acoustic_buoy-risk", "alert_threshold": 5.0, "observed_at": now,
	}, installationOperator.ID, http.StatusCreated, &signal_recovery_report)
	apiRequest(t, handler, http.MethodPost, "/v1/signal_recovery_report/"+signal_recovery_report["id"]+"/review", map[string]any{
		"recovery_job_id": first.ID, "accepted": true, "signal_recovery_report_version": 1, "task_version": 5,
	}, reviewer.ID, http.StatusOK, nil)
	apiRequest(t, handler, http.MethodPost, "/v1/dive_windows/"+dive_windowID+"/complete?version=2", nil, installationOperator.ID, http.StatusOK, nil)

	var safety_alert map[string]string
	apiRequest(t, handler, http.MethodPost, "/v1/integrity_incidents", map[string]any{
		"recovery_job_id": first.ID, "kind": "close_record", "reason": "scheduled record closure", "due_at": now.Add(time.Hour),
	}, field.ID, http.StatusCreated, &safety_alert)
	apiRequest(t, handler, http.MethodPost, "/v1/integrity_incidents/"+safety_alert["id"]+"/start", nil, field.ID, http.StatusOK, nil)
	apiRequest(t, handler, http.MethodPost, "/v1/integrity_incidents/"+safety_alert["id"]+"/close", nil, field.ID, http.StatusOK, nil)
	apiRequest(t, handler, http.MethodPost, "/v1/recovery_jobs/"+first.ID+"/archive", map[string]any{"version": 6}, field.ID, http.StatusOK, nil)

	second := createRecoveryJob("HTTP-TASK-02")
	var cancelDiveWindow map[string]string
	apiRequest(t, handler, http.MethodPost, "/v1/dive_windows", map[string]any{
		"code": "HTTP-ROUND-02", "method": "evening-acoustic_buoy", "capacity": 1, "recovery_job_ids": []string{second.ID},
	}, installationOperator.ID, http.StatusCreated, &cancelDiveWindow)
	apiRequest(t, handler, http.MethodPost, "/v1/dive_windows/"+cancelDiveWindow["id"]+"/cancel?version=1", nil, installationOperator.ID, http.StatusOK, nil)
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM recovery_jobs WHERE id=$1`, second.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != string(domain.RecoveryJobAccepted) {
		t.Fatalf("cancelled dive_window left task status=%s", status)
	}
	var collected int
	if err := pool.QueryRow(ctx, `SELECT completed_installs FROM acoustic_buoys WHERE id=$1`, acoustic_buoyID).Scan(&collected); err != nil {
		t.Fatal(err)
	}
	if collected != 2 {
		t.Fatalf("acoustic_buoy completed_installs=%d want=2", collected)
	}

	apiRequest(t, handler, http.MethodGet, "/v1/cases/"+survey_missionID+"/progress", nil, "", http.StatusOK, nil)
	apiRequest(t, handler, http.MethodGet, "/v1/cases/"+survey_missionID+"/report", nil, "", http.StatusOK, nil)
	apiRequest(t, handler, http.MethodGet, fmt.Sprintf("/v1/audit/acoustic_buoy_task/%s?limit=20", first.ID), nil, "", http.StatusOK, nil)
	apiRequest(t, handler, http.MethodPost, "/v1/cases/"+survey_missionID+"/close", map[string]any{"version": 3}, field.ID, http.StatusOK, nil)
}
