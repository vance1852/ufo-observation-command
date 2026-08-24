package domain

import (
	"errors"
	"testing"
	"time"
)

func TestArrayOperatorRolesAndPermissions(t *testing.T) {
	cases := []struct {
		role       ArrayOperatorRole
		permission Permission
		want       bool
	}{
		{RoleAcousticBuoyOperator, PermissionRecoveryJobComplete, true},
		{RoleAcousticBuoyOperator, PermissionSignalRecoveryReportReview, false},
		{RoleInstallationOperator, PermissionSignalRecoveryReportRecord, true},
		{RoleQualityReviewer, PermissionSignalRecoveryReportReview, true},
		{RoleSafetySupervisor, PermissionIntegrityIncidentClose, true},
	}
	for _, tc := range cases {
		array_operator := ArrayOperator{ID: "op", Name: "ArrayOperator", Role: tc.role}
		if got := array_operator.Has(tc.permission); got != tc.want {
			t.Errorf("role=%s permission=%s got=%v want=%v", tc.role, tc.permission, got, tc.want)
		}
	}
}

func TestArrayOperatorValidationRejectsUnknownRole(t *testing.T) {
	if err := (ArrayOperator{ID: "op", Name: "ArrayOperator", Role: "unknown"}).Validate(); !errors.Is(err, ErrConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestAssignmentLifecycle(t *testing.T) {
	start := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	a := Assignment{ID: "a", SurveyMissionID: "p", AcousticBuoyID: "s", ArrayOperatorID: "o", StartsAt: start, EndsAt: start.Add(time.Hour), Status: "queued"}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if !a.CanMoveTo("active") || a.CanMoveTo("completed") {
		t.Fatal("assignment transition graph is wrong")
	}
	if a.ActiveAt(start) {
		t.Fatal("queued assignment is not active")
	}
	a.Status = "active"
	if !a.ActiveAt(start.Add(time.Minute)) {
		t.Fatal("active assignment not active in window")
	}
	if a.ActiveAt(start.Add(time.Hour)) {
		t.Fatal("end boundary should be inactive")
	}
}

func TestRecoveryJobFilterAndSearch(t *testing.T) {
	recovery_jobs := []RecoveryJob{
		{ID: "2", SurveyMissionID: "p", TaskCode: "S-002", Status: RecoveryJobAccepted, ExpiresAt: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)},
		{ID: "1", SurveyMissionID: "p", TaskCode: "S-001", Status: RecoveryJobCompleted, ExpiresAt: time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)},
		{ID: "3", SurveyMissionID: "q", TaskCode: "S-003", Status: RecoveryJobAccepted, ExpiresAt: time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)},
	}
	request := SearchRequest{Filter: RecoveryJobFilter{SurveyMissionID: " p ", Search: "002"}, Sort: SortExpiry, Limit: 10}
	items := SearchRecoveryJobs(recovery_jobs, request)
	if len(items) != 1 || items[0].ID != "2" {
		t.Fatalf("search result = %+v", items)
	}
	if !(RecoveryJobFilter{Status: RecoveryJobAccepted}).Matches(recovery_jobs[0]) {
		t.Fatal("status filter did not match")
	}
}

func TestBulkValidationDetectsDuplicateAndExpiredInput(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	valid := []RecoveryJobRequest{{SurveyMissionID: "p", AcousticBuoyID: "s", TaskCode: "S-1", ExpiresAt: now.Add(time.Hour)}}
	if err := ValidateBulkRequests(valid, now); err != nil {
		t.Fatal(err)
	}
	duplicate := append(valid, valid[0])
	if err := ValidateBulkRequests(duplicate, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate error = %v", err)
	}
	expired := []RecoveryJobRequest{{SurveyMissionID: "p", AcousticBuoyID: "s", TaskCode: "S-2", ExpiresAt: now.Add(-time.Second)}}
	if err := ValidateBulkRequests(expired, now); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired error = %v", err)
	}
}

func TestStateMachinesReachExpectedStates(t *testing.T) {
	survey_missionStates := DefaultSurveyMissionMachine().Reachable("draft")
	if len(survey_missionStates) != 4 {
		t.Fatalf("survey_mission states = %v", survey_missionStates)
	}
	if err := DefaultRecoveryJobMachine().ValidatePath([]string{"queued", "completed", "activation_pending", "accepted", "in_progress", "rejected", "archived"}); err != nil {
		t.Fatal(err)
	}
	if err := DefaultDiveWindowMachine().ValidatePath([]string{"queued", "completed"}); err == nil {
		t.Fatal("invalid dive_window path accepted")
	}
}

func TestConstraintSetAndRedaction(t *testing.T) {
	if err := (ConstraintSet{MaxRecoveryJobsPerDiveWindow: 2, MinimumRemainingTTL: time.Hour}).Validate(); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	task := RecoveryJob{Status: RecoveryJobAccepted, ExpiresAt: now.Add(2 * time.Hour)}
	if !(ConstraintSet{MaxRecoveryJobsPerDiveWindow: 2, MinimumRemainingTTL: time.Hour}).AllowsRecoveryJob(task, now) {
		t.Fatal("valid task rejected")
	}
	if RedactTaskCode("S-1234") != "S-**34" {
		t.Fatalf("redaction mismatch")
	}
	if RedactArrayOpsLane("North Gate") != "N***e" {
		t.Fatalf("arrayops_lane redaction mismatch")
	}
}

func TestValidationHelpers(t *testing.T) {
	if err := ValidateBusinessCode("PLAN-001"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBusinessCode("bad code"); !errors.Is(err, ErrConflict) {
		t.Fatalf("code error = %v", err)
	}
	start := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	if err := ValidateUTCWindow(start, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePositiveVersion(0); !errors.Is(err, ErrConflict) {
		t.Fatalf("version error = %v", err)
	}
	if err := ValidatePage(0, 101); !errors.Is(err, ErrConflict) {
		t.Fatalf("page error = %v", err)
	}
	if err := ValidateSignalRecoveryReport(1, 2, "mg/L"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReason("ok"); !errors.Is(err, ErrConflict) {
		t.Fatalf("reason error = %v", err)
	}
}
