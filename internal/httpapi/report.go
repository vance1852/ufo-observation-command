package httpapi

import "net/http"

func (a *API) complianceReport(w http.ResponseWriter, r *http.Request) {
	report, err := a.service.ComplianceReport(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, report)
}
