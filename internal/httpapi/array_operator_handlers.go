package httpapi

import (
	"net/http"

	"ufo-observation-command/internal/domain"
)

func (a *API) listArrayOperators(w http.ResponseWriter, r *http.Request) {
	items, total, err := a.service.ListArrayOperators(r.Context(), domain.ArrayOperatorRole(r.URL.Query().Get("role")), 50, 0)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

type renameArrayOperatorRequest struct {
	Name string `json:"name"`
}

func (a *API) renameArrayOperator(w http.ResponseWriter, r *http.Request) {
	var in renameArrayOperatorRequest
	if !decode(w, r, &in) {
		return
	}
	if err := a.service.RenameArrayOperator(r.Context(), meta(r), r.PathValue("id"), in.Name); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "renamed"})
}
