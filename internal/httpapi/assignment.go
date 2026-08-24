package httpapi

import (
	"net/http"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

type assignmentRequest struct {
	SurveyMissionID string    `json:"survey_mission_id"`
	AcousticBuoyID  string    `json:"acoustic_buoy_id"`
	ArrayOperatorID string    `json:"array_operator_id"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
}

func (a *API) createAssignment(w http.ResponseWriter, r *http.Request) {
	var in assignmentRequest
	if !decode(w, r, &in) {
		return
	}
	array_operator, err := a.service.LoadArrayOperator(r.Context(), in.ArrayOperatorID)
	if err != nil {
		writeError(w, err)
		return
	}
	assignment := repository.NewAssignment(in.SurveyMissionID, in.AcousticBuoyID, in.ArrayOperatorID, in.StartsAt, in.EndsAt)
	if err := a.service.AssignAcousticBuoy(r.Context(), meta(r), assignment, array_operator); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, assignment)
}

type assignmentAdvanceRequest struct {
	Status  string `json:"status"`
	Version int64  `json:"version"`
}

func (a *API) advanceAssignment(w http.ResponseWriter, r *http.Request) {
	var in assignmentAdvanceRequest
	if !decode(w, r, &in) {
		return
	}
	if in.Status != "active" && in.Status != "completed" && in.Status != "cancelled" {
		writeError(w, domain.ErrConflict)
		return
	}
	if err := a.service.AdvanceAssignment(r.Context(), meta(r), r.PathValue("id"), in.Status, in.Version); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": in.Status})
}
