package domain

import "testing"

func TestStatusCountsIncludesOnlyObservedStatuses(t *testing.T) {
	counts := StatusCounts([]RecoveryJob{{Status: RecoveryJobAccepted}, {Status: RecoveryJobAccepted}, {Status: RecoveryJobRejected}})
	if counts[RecoveryJobAccepted] != 2 || counts[RecoveryJobRejected] != 1 || counts[RecoveryJobVerified] != 0 {
		t.Fatalf("counts=%v", counts)
	}
}

func TestSurveyMissionProgressCompleteRequiresNoRejectedRecoveryJobs(t *testing.T) {
	progress := SurveyMissionProgress{Required: 2, Completed: 2}
	if !progress.Complete() {
		t.Fatal("complete progress rejected")
	}
	progress.Rejected = 1
	if progress.Complete() {
		t.Fatal("rejected progress marked complete")
	}
}

func TestArrayOperatorCanAction(t *testing.T) {
	array_operator := ArrayOperator{ID: "o", Name: "Supervisor", Role: RoleSafetySupervisor}
	if !array_operator.Can("archive") || !array_operator.Can("review_signal_recovery_report") {
		t.Fatal("safety_supervisor permissions missing")
	}
	field := ArrayOperator{ID: "f", Name: "Field", Role: RoleAcousticBuoyOperator}
	if field.Can("archive") {
		t.Fatal("acoustic_buoy_operator can archive")
	}
}
