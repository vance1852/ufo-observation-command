package domain

import "time"

type ComplianceReport struct {
	SurveyMissionID        string
	GeneratedAt            time.Time
	Progress               SurveyMissionProgress
	Expiring               []RecoveryJob
	OpenIntegrityIncidents int
}

func (r ComplianceReport) AtRisk() bool {
	return len(r.Expiring) > 0 || r.OpenIntegrityIncidents > 0 || r.Progress.Rejected > 0
}

func (r ComplianceReport) Status() string {
	if r.AtRisk() {
		return "attention_required"
	}
	if r.Progress.Complete() {
		return "complete"
	}
	return "in_progress"
}
