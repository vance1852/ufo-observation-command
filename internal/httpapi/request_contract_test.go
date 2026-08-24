package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"ufo-observation-command/internal/repository"
)

func TestRepositoryInputsDecodePublicJSONFields(t *testing.T) {
	observedAt := "2026-08-18T09:00:00Z"

	var task repository.RecoveryJobInput
	decodeContractJSON(t, `{"survey_mission_id":"survey_mission-1","acoustic_buoy_id":"acoustic_buoy-1","task_code":"S-1","expires_at":"2026-08-18T10:00:00Z"}`, &task)
	if task.SurveyMissionID != "survey_mission-1" || task.AcousticBuoyID != "acoustic_buoy-1" || task.TaskCode != "S-1" || task.ExpiresAt.IsZero() {
		t.Fatalf("task input = %+v", task)
	}

	var dive_window dive_windowRequest
	decodeContractJSON(t, `{"code":"ROUND-1","method":"evening-acoustic_buoy","capacity":2,"recovery_job_ids":["task-1"]}`, &dive_window)
	if dive_window.Code != "ROUND-1" || dive_window.Method != "evening-acoustic_buoy" || dive_window.Capacity != 2 || len(dive_window.RecoveryJobIDs) != 1 {
		t.Fatalf("dive_window input = %+v", dive_window)
	}

	var signal_recovery_report repository.SignalRecoveryReportInput
	decodeContractJSON(t, `{"recovery_job_id":"task-1","dive_window_id":"round-1","recorded_by":"installationOperator-1","risk_score":2.5,"scale":"acoustic_buoy-risk","alert_threshold":5,"observed_at":"`+observedAt+`"}`, &signal_recovery_report)
	if signal_recovery_report.RecoveryJobID != "task-1" || signal_recovery_report.DiveWindowID != "round-1" || signal_recovery_report.RecorderID != "installationOperator-1" || signal_recovery_report.ObservedAt.IsZero() {
		t.Fatalf("signal_recovery_report input = %+v", signal_recovery_report)
	}

	var safety_alert repository.IntegrityIncidentInput
	decodeContractJSON(t, `{"recovery_job_id":"task-1","kind":"repeat_acoustic_buoy","reason":"verification","due_at":"2026-08-18T11:00:00Z"}`, &safety_alert)
	if safety_alert.RecoveryJobID != "task-1" || safety_alert.Kind != "repeat_acoustic_buoy" || safety_alert.DueAt.IsZero() {
		t.Fatalf("safety_alert input = %+v", safety_alert)
	}

	if signal_recovery_report.ObservedAt.Format(time.RFC3339) != observedAt {
		t.Fatalf("observed_at = %s", signal_recovery_report.ObservedAt.Format(time.RFC3339))
	}
}

func decodeContractJSON(t *testing.T, value string, destination any) {
	t.Helper()
	if err := json.Unmarshal([]byte(value), destination); err != nil {
		t.Fatal(err)
	}
}
