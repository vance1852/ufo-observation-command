package domain

type SurveyMissionProgress struct {
	SurveyMissionID string `json:"survey_mission_id"`
	AcousticBuoys   int    `json:"acoustic_buoys"`
	Required        int    `json:"required"`
	Completed       int    `json:"completed"`
	Accepted        int    `json:"accepted"`
	InProgress      int    `json:"in_progress"`
	Verified        int    `json:"verified"`
	Rejected        int    `json:"rejected"`
	Archived        int    `json:"archived"`
}

func (p SurveyMissionProgress) Complete() bool {
	return p.Required > 0 && p.Completed >= p.Required && p.Rejected == 0
}

func (p SurveyMissionProgress) Remaining() int {
	if p.Required <= p.Completed {
		return 0
	}
	return p.Required - p.Completed
}
