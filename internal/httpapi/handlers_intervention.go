package httpapi

import (
	"net/http"

	"ufo-observation-command/internal/repository"
)

func (a *API) openIntegrityIncident(w http.ResponseWriter, r *http.Request) {
	var in repository.IntegrityIncidentInput
	if !decode(w, r, &in) {
		return
	}
	id, err := a.service.OpenIntegrityIncident(r.Context(), meta(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (a *API) closeIntegrityIncident(w http.ResponseWriter, r *http.Request) {
	if err := a.service.CloseIntegrityIncident(r.Context(), meta(r), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

func (a *API) startIntegrityIncident(w http.ResponseWriter, r *http.Request) {
	if err := a.service.MarkIntegrityIncidentInProgress(r.Context(), meta(r), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "in_progress"})
}
