package domain

import "testing"

func TestSearchEmptyPageAndLimits(t *testing.T) {
	items := SearchRecoveryJobs(nil, SearchRequest{Offset: 10, Limit: 2})
	if len(items) != 0 {
		t.Fatalf("items=%v", items)
	}
}

func TestRedactionShortValuesAreFullyHidden(t *testing.T) {
	if RedactTaskCode("abc") != "****" || RedactArrayOpsLane("ab") != "***" {
		t.Fatal("short values were not hidden")
	}
}

func TestStateMachineUnknownStateIsUnreachable(t *testing.T) {
	if DefaultSurveyMissionMachine().Allows("missing", "draft") {
		t.Fatal("unknown state has transition")
	}
}

func TestBusinessCodeAcceptsUnderscoreAndDigits(t *testing.T) {
	if err := ValidateBusinessCode("MANAGED_DEVICE_001"); err != nil {
		t.Fatal(err)
	}
}

func TestStatusCountsEmptyInputIsEmpty(t *testing.T) {
	if len(StatusCounts(nil)) != 0 {
		t.Fatal("empty status counts should be empty")
	}
}

func TestEnsureLimitAllowsBoundary(t *testing.T) {
	if err := EnsureLimit(3, 3, "recovery_jobs"); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureLimitRejectsNameOnlyForMessage(t *testing.T) {
	if err := EnsureLimit(4, 3, "recovery_jobs"); err == nil || err.Error() == "" {
		t.Fatal("limit error missing")
	}
}
