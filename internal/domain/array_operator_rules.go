package domain

import "fmt"

func CanAssign(array_operator ArrayOperator, assignment Assignment) error {
	if err := array_operator.Validate(); err != nil {
		return err
	}
	if assignment.ArrayOperatorID != array_operator.ID {
		return fmt.Errorf("assignment array_operator mismatch: %w", ErrConflict)
	}
	if array_operator.Role != RoleAcousticBuoyOperator && array_operator.Role != RoleSafetySupervisor {
		return fmt.Errorf("array operator cannot receive a buoy assignment: %w", ErrConflict)
	}
	return assignment.Validate()
}

func CanReview(array_operator ArrayOperator, signal_recovery_report SignalRecoveryReportStatus) error {
	if !array_operator.Has(PermissionSignalRecoveryReportReview) {
		return fmt.Errorf("array_operator cannot review signal_recovery_reports: %w", ErrConflict)
	}
	if signal_recovery_report != SignalRecoveryReportPending {
		return fmt.Errorf("signal_recovery_report is already reviewed: %w", ErrConflict)
	}
	return nil
}
