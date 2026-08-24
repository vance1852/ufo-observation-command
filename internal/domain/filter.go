package domain

import "strings"

type RecoveryJobFilter struct {
	SurveyMissionID string
	Status          RecoveryJobStatus
	Search          string
}

func (f RecoveryJobFilter) Normalize() RecoveryJobFilter {
	f.SurveyMissionID = strings.TrimSpace(f.SurveyMissionID)
	f.Search = strings.TrimSpace(strings.ToLower(f.Search))
	return f
}

func (f RecoveryJobFilter) Matches(s RecoveryJob) bool {
	f = f.Normalize()
	if f.SurveyMissionID != "" && s.SurveyMissionID != f.SurveyMissionID {
		return false
	}
	if f.Status != "" && s.Status != f.Status {
		return false
	}
	if f.Search != "" && !strings.Contains(strings.ToLower(s.TaskCode), f.Search) {
		return false
	}
	return true
}
