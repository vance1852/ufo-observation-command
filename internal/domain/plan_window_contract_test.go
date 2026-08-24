package domain

import (
	"testing"
	"time"
)

func TestCollectionRequiresAnOpenSurveyMissionWindow(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	survey_mission := SurveyMission{Status: SurveyMissionActive, StartsAt: now.Add(-2 * time.Hour), EndsAt: now.Add(-time.Hour)}
	if SurveyMissionExecutionAllowed(survey_mission, now) {
		t.Fatal("collection allowed after survey_mission window ended")
	}
}
