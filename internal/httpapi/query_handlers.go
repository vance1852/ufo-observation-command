package httpapi

import (
	"net/http"
	"strconv"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (a *API) listSurveyMissions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	items, total, err := a.service.ListSurveyMissions(r.Context(), repository.SurveyMissionFilter{Status: domain.SurveyMissionStatus(r.URL.Query().Get("status")), Search: r.URL.Query().Get("search"), Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

func (a *API) listSurveyMissionAcousticBuoys(w http.ResponseWriter, r *http.Request) {
	items, err := a.service.ListSurveyMissionAcousticBuoys(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, items)
}

func (a *API) auditHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := a.service.AuditHistory(r.Context(), r.PathValue("object_type"), r.PathValue("object_id"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, items)
}
