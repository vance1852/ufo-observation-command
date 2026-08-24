package domain

import "testing"

func TestSignalRecoveryReportRecordingRequiresInstallationOperatorRole(t *testing.T) {
	tests := []struct {
		role    ArrayOperatorRole
		allowed bool
	}{
		{role: RoleAcousticBuoyOperator, allowed: false},
		{role: RoleQualityReviewer, allowed: false},
		{role: RoleInstallationOperator, allowed: true},
		{role: RoleSafetySupervisor, allowed: true},
	}
	for _, test := range tests {
		array_operator := ArrayOperator{ID: "array_operator", Name: "ArrayOperator", Role: test.role}
		if actual := array_operator.CanRecordSignalRecoveryReport(); actual != test.allowed {
			t.Fatalf("role=%s allowed=%v", test.role, actual)
		}
	}
}
