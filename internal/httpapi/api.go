package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ufo-observation-command/internal/console"
	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
	"github.com/google/uuid"
)

type API struct {
	service      *service.Service
	ready        func(context.Context) error
	consoleStore *console.Store
}

func New(svc *service.Service, ready func(context.Context) error) *API {
	return &API{service: svc, ready: ready}
}

func (a *API) WithConsole(store *console.Store) *API {
	a.consoleStore = store
	return a
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /readyz", a.readyz)
	mux.HandleFunc("POST /v1/cases", a.createSurveyMission)
	mux.HandleFunc("POST /v1/cases/{id}/schedule", a.scheduleSurveyMission)
	mux.HandleFunc("POST /v1/cases/{id}/activate", a.activateSurveyMission)
	mux.HandleFunc("POST /v1/cases/{id}/close", a.closeSurveyMission)
	mux.HandleFunc("GET /v1/cases/{id}/progress", a.survey_missionProgress)
	mux.HandleFunc("GET /v1/cases", a.listSurveyMissions)
	mux.HandleFunc("GET /v1/cases/{id}/acoustic_buoys", a.listSurveyMissionAcousticBuoys)
	mux.HandleFunc("GET /v1/cases/{id}/report", a.complianceReport)
	mux.HandleFunc("POST /v1/array_operators", a.createArrayOperator)
	mux.HandleFunc("GET /v1/array_operators", a.listArrayOperators)
	mux.HandleFunc("POST /v1/array_operators/{id}/rename", a.renameArrayOperator)
	mux.HandleFunc("POST /v1/assignments", a.createAssignment)
	mux.HandleFunc("POST /v1/assignments/{id}/advance", a.advanceAssignment)
	mux.HandleFunc("POST /v1/recovery_jobs", a.createRecoveryJob)
	mux.HandleFunc("POST /v1/recovery_jobs/{id}/complete", a.completeRecoveryJob)
	mux.HandleFunc("POST /v1/recovery_jobs/{id}/activation", a.transferRecoveryJob)
	mux.HandleFunc("POST /v1/recovery_jobs/{id}/accept", a.acceptRecoveryJob)
	mux.HandleFunc("POST /v1/recovery_jobs/{id}/archive", a.archiveRecoveryJob)
	mux.HandleFunc("GET /v1/recovery_jobs", a.listRecoveryJobs)
	mux.HandleFunc("POST /v1/dive_windows", a.createDiveWindow)
	mux.HandleFunc("POST /v1/dive_windows/{id}/start", a.startDiveWindow)
	mux.HandleFunc("POST /v1/dive_windows/{id}/complete", a.completeDiveWindow)
	mux.HandleFunc("POST /v1/dive_windows/{id}/cancel", a.cancelDiveWindow)
	mux.HandleFunc("POST /v1/signal_recovery_report", a.submitSignalRecoveryReport)
	mux.HandleFunc("POST /v1/signal_recovery_report/{id}/review", a.reviewSignalRecoveryReport)
	mux.HandleFunc("POST /v1/integrity_incidents", a.openIntegrityIncident)
	mux.HandleFunc("POST /v1/integrity_incidents/{id}/start", a.startIntegrityIncident)
	mux.HandleFunc("POST /v1/integrity_incidents/{id}/close", a.closeIntegrityIncident)
	mux.HandleFunc("GET /v1/audit", a.queryAudit)
	mux.HandleFunc("GET /v1/audit/{object_type}/{object_id}", a.auditHistory)
	if a.consoleStore != nil {
		a.registerConsoleRoutes(mux)
	}
	return requestMiddleware(mux)
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	if a.ready != nil {
		if err := a.ready(r.Context()); err != nil {
			writeError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type survey_missionRequest struct {
	Code          string                         `json:"code"`
	Name          string                         `json:"name"`
	Timezone      string                         `json:"timezone"`
	StartsAt      time.Time                      `json:"starts_at"`
	EndsAt        time.Time                      `json:"ends_at"`
	CreatedBy     string                         `json:"created_by"`
	AcousticBuoys []repository.AcousticBuoyInput `json:"acoustic_buoys"`
}

func (a *API) createSurveyMission(w http.ResponseWriter, r *http.Request) {
	var in survey_missionRequest
	if !decode(w, r, &in) {
		return
	}
	result, err := a.service.CreateSurveyMission(r.Context(), meta(r), service.CreateSurveyMissionRequest{Code: in.Code, Name: in.Name, Timezone: in.Timezone, StartsAt: in.StartsAt, EndsAt: in.EndsAt, CreatedBy: in.CreatedBy, AcousticBuoys: in.AcousticBuoys})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

type versionRequest struct {
	Version int64 `json:"version"`
}

func (a *API) scheduleSurveyMission(w http.ResponseWriter, r *http.Request) {
	a.advanceSurveyMission(w, r, domain.SurveyMissionScheduled)
}
func (a *API) activateSurveyMission(w http.ResponseWriter, r *http.Request) {
	a.advanceSurveyMission(w, r, domain.SurveyMissionActive)
}
func (a *API) closeSurveyMission(w http.ResponseWriter, r *http.Request) {
	a.advanceSurveyMission(w, r, domain.SurveyMissionClosed)
}

func (a *API) advanceSurveyMission(w http.ResponseWriter, r *http.Request, next domain.SurveyMissionStatus) {
	var in versionRequest
	if !decode(w, r, &in) {
		return
	}
	var err error
	switch next {
	case domain.SurveyMissionScheduled:
		err = a.service.ScheduleSurveyMission(r.Context(), meta(r), r.PathValue("id"), in.Version)
	case domain.SurveyMissionActive:
		err = a.service.ActivateSurveyMission(r.Context(), meta(r), r.PathValue("id"), in.Version)
	case domain.SurveyMissionClosed:
		err = a.service.CloseSurveyMission(r.Context(), meta(r), r.PathValue("id"), in.Version)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(next)})
}

func (a *API) createRecoveryJob(w http.ResponseWriter, r *http.Request) {
	var in repository.RecoveryJobInput
	if !decode(w, r, &in) {
		return
	}
	result, err := a.service.CreateRecoveryJob(r.Context(), meta(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *API) completeRecoveryJob(w http.ResponseWriter, r *http.Request) {
	a.moveRecoveryJob(w, r, domain.RecoveryJobCompleted)
}

func (a *API) archiveRecoveryJob(w http.ResponseWriter, r *http.Request) {
	a.moveRecoveryJob(w, r, domain.RecoveryJobArchived)
}

func (a *API) moveRecoveryJob(w http.ResponseWriter, r *http.Request, next domain.RecoveryJobStatus) {
	var in versionRequest
	if !decode(w, r, &in) {
		return
	}
	var err error
	if next == domain.RecoveryJobCompleted {
		err = a.service.CompleteRecoveryJob(r.Context(), meta(r), r.PathValue("id"), in.Version)
	} else {
		err = a.service.ArchiveRecoveryJob(r.Context(), meta(r), r.PathValue("id"), in.Version)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(next)})
}

func (a *API) transferRecoveryJob(w http.ResponseWriter, r *http.Request) {
	a.activation(w, r, domain.RecoveryJobActivationPending)
}

func (a *API) acceptRecoveryJob(w http.ResponseWriter, r *http.Request) {
	a.activation(w, r, domain.RecoveryJobAccepted)
}

type activationRequest struct {
	From       *string   `json:"from_operator"`
	To         string    `json:"to_operator"`
	Location   string    `json:"location"`
	RecordedAt time.Time `json:"recorded_at"`
	Note       string    `json:"note"`
	Version    int64     `json:"version"`
}

func (a *API) activation(w http.ResponseWriter, r *http.Request, next domain.RecoveryJobStatus) {
	var in activationRequest
	if !decode(w, r, &in) {
		return
	}
	if in.RecordedAt.IsZero() {
		in.RecordedAt = time.Now().UTC()
	}
	input := repository.ActivationInput{RecoveryJobID: r.PathValue("id"), From: in.From, To: in.To, Location: in.Location, RecordedAt: in.RecordedAt, Note: in.Note}
	var err error
	if next == domain.RecoveryJobActivationPending {
		err = a.service.ActivationChecked(r.Context(), meta(r), input, in.Version)
	} else {
		err = a.service.AcceptChecked(r.Context(), meta(r), input, in.Version)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(next)})
}

func (a *API) listRecoveryJobs(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := a.service.ListRecoveryJobs(r.Context(), offset, limit, r.URL.Query().Get("survey_mission_id"), domain.RecoveryJobStatus(r.URL.Query().Get("status")))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

type dive_windowRequest struct {
	repository.DiveWindowInput
	RecoveryJobIDs []string `json:"recovery_job_ids"`
}

func (a *API) createDiveWindow(w http.ResponseWriter, r *http.Request) {
	var in dive_windowRequest
	if !decode(w, r, &in) {
		return
	}
	id, err := a.service.CreateDiveWindow(r.Context(), meta(r), in.DiveWindowInput, append([]string(nil), in.RecoveryJobIDs...))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (a *API) submitSignalRecoveryReport(w http.ResponseWriter, r *http.Request) {
	var in repository.SignalRecoveryReportInput
	if !decode(w, r, &in) {
		return
	}
	id, err := a.service.SubmitSignalRecoveryReport(r.Context(), meta(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

type reviewRequest struct {
	RecoveryJobID               string `json:"recovery_job_id"`
	Accepted                    bool   `json:"accepted"`
	SignalRecoveryReportVersion int64  `json:"signal_recovery_report_version"`
	RecoveryJobVersion          int64  `json:"task_version"`
}

func (a *API) reviewSignalRecoveryReport(w http.ResponseWriter, r *http.Request) {
	var in reviewRequest
	if !decode(w, r, &in) {
		return
	}
	err := a.service.ReviewSignalRecoveryReport(r.Context(), meta(r), r.PathValue("id"), in.RecoveryJobID, in.Accepted, in.SignalRecoveryReportVersion, in.RecoveryJobVersion)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reviewed"})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, fmt.Errorf("invalid json: %w", domain.ErrConflict))
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, fmt.Errorf("request body must contain one json value: %w", domain.ErrConflict))
		return false
	}
	return true
}

func meta(r *http.Request) service.RequestMeta {
	array_operator := strings.TrimSpace(r.Header.Get("X-ArrayOperator-ID"))
	var array_operatorID *string
	if _, err := uuid.Parse(array_operator); err == nil {
		array_operatorID = &array_operator
	}
	return service.RequestMeta{RequestID: requestID(r.Context()), ArrayOperatorID: array_operatorID}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInvalidTransition):
		status, code = http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrCapacityExceeded):
		status, code = http.StatusUnprocessableEntity, "capacity_exceeded"
	case errors.Is(err, domain.ErrExpired):
		status, code = http.StatusUnprocessableEntity, "expired"
	}
	writeJSON(w, status, map[string]string{"code": code, "message": err.Error(), "request_id": requestIDFromWriter(w)})
}

func requestIDFromWriter(w http.ResponseWriter) string { return w.Header().Get("X-Request-ID") }
