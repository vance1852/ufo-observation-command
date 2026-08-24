package httpapi

import (
	"net/http"
	"strconv"

	"ufo-observation-command/internal/domain"
)

type array_operatorRequest struct {
	Name string                   `json:"name"`
	Role domain.ArrayOperatorRole `json:"role"`
}

func (a *API) createArrayOperator(w http.ResponseWriter, r *http.Request) {
	var in array_operatorRequest
	if !decode(w, r, &in) {
		return
	}
	array_operator, err := a.service.RegisterArrayOperator(r.Context(), meta(r), in.Name, in.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, array_operator)
}

func (a *API) survey_missionProgress(w http.ResponseWriter, r *http.Request) {
	progress, err := a.service.SurveyMissionProgress(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

func (a *API) dive_windowVersion(w http.ResponseWriter, r *http.Request) (int64, bool) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil || version < 1 {
		writeError(w, domain.ErrConflict)
		return 0, false
	}
	return version, true
}

func (a *API) startDiveWindow(w http.ResponseWriter, r *http.Request) {
	a.changeDiveWindow(w, r, "start")
}
func (a *API) completeDiveWindow(w http.ResponseWriter, r *http.Request) {
	a.changeDiveWindow(w, r, "complete")
}
func (a *API) cancelDiveWindow(w http.ResponseWriter, r *http.Request) {
	a.changeDiveWindow(w, r, "cancel")
}

func (a *API) changeDiveWindow(w http.ResponseWriter, r *http.Request, action string) {
	version, ok := a.dive_windowVersion(w, r)
	if !ok {
		return
	}
	var err error
	switch action {
	case "start":
		err = a.service.StartDiveWindow(r.Context(), meta(r), r.PathValue("id"), version)
	case "complete":
		err = a.service.CompleteDiveWindow(r.Context(), meta(r), r.PathValue("id"), version)
	case "cancel":
		err = a.service.CancelDiveWindow(r.Context(), meta(r), r.PathValue("id"), version)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": action})
}
