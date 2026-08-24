package domain

import (
	"fmt"
	"strings"
)

type ArrayOperatorRole string

const (
	RoleAcousticBuoyOperator ArrayOperatorRole = "acoustic_buoy_operator"
	RoleInstallationOperator ArrayOperatorRole = "signal_recovery_report_array_operator"
	RoleQualityReviewer      ArrayOperatorRole = "quality_reviewer"
	RoleSafetySupervisor     ArrayOperatorRole = "safety_supervisor"
)

type ArrayOperator struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Role ArrayOperatorRole `json:"role"`
}

func (o ArrayOperator) Validate() error {
	if strings.TrimSpace(o.ID) == "" || strings.TrimSpace(o.Name) == "" {
		return fmt.Errorf("array_operator identity is required: %w", ErrConflict)
	}
	switch o.Role {
	case RoleAcousticBuoyOperator, RoleInstallationOperator, RoleQualityReviewer, RoleSafetySupervisor:
		return nil
	default:
		return fmt.Errorf("unknown array_operator role: %w", ErrConflict)
	}
}

func (o ArrayOperator) CanRecordSignalRecoveryReport() bool {
	if err := o.Validate(); err != nil {
		return false
	}
	switch o.Role {
	case RoleInstallationOperator, RoleSafetySupervisor:
		return true
	default:
		return false
	}
}

func (o ArrayOperator) Can(action string) bool {
	switch action {
	case "complete", "activation":
		return o.Role == RoleAcousticBuoyOperator || o.Role == RoleSafetySupervisor
	case "record_signal_recovery_report":
		return o.Role == RoleInstallationOperator || o.Role == RoleSafetySupervisor
	case "review_signal_recovery_report":
		return o.Role == RoleQualityReviewer || o.Role == RoleSafetySupervisor
	case "close_survey_mission", "archive":
		return o.Role == RoleSafetySupervisor
	default:
		return false
	}
}
