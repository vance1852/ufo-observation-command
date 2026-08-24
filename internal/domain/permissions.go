package domain

type Permission string

const (
	PermissionSurveyMissionWrite         Permission = "survey_mission:write"
	PermissionRecoveryJobComplete        Permission = "recovery_job:complete"
	PermissionSignalRecoveryReportRecord Permission = "signal_recovery_report:record"
	PermissionSignalRecoveryReportReview Permission = "signal_recovery_report:review"
	PermissionIntegrityIncidentClose     Permission = "integrity_incident:close"
)

func (o ArrayOperator) Permissions() []Permission {
	switch o.Role {
	case RoleAcousticBuoyOperator:
		return []Permission{PermissionRecoveryJobComplete}
	case RoleInstallationOperator:
		return []Permission{PermissionSignalRecoveryReportRecord}
	case RoleQualityReviewer:
		return []Permission{PermissionSignalRecoveryReportReview}
	case RoleSafetySupervisor:
		return []Permission{PermissionSurveyMissionWrite, PermissionRecoveryJobComplete, PermissionSignalRecoveryReportRecord, PermissionSignalRecoveryReportReview, PermissionIntegrityIncidentClose}
	default:
		return nil
	}
}

func (o ArrayOperator) Has(permission Permission) bool {
	for _, item := range o.Permissions() {
		if item == permission {
			return true
		}
	}
	return false
}
