package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ufo-observation-command/internal/audit"
	"ufo-observation-command/internal/domain"
	"github.com/google/uuid"
)

func (a *API) queryAudit(w http.ResponseWriter, r *http.Request) {
	from, err := requiredTimeQuery(r, "from")
	if err != nil {
		writeError(w, err)
		return
	}
	to, err := requiredTimeQuery(r, "to")
	if err != nil {
		writeError(w, err)
		return
	}
	limit, err := boundedIntQuery(r, "limit", 50, 0, 200)
	if err != nil {
		writeError(w, err)
		return
	}
	offset, err := boundedIntQuery(r, "offset", 0, -1, int(^uint(0)>>1))
	if err != nil {
		writeError(w, err)
		return
	}
	objectID := r.URL.Query().Get("object_id")
	if objectID != "" {
		if _, err := uuid.Parse(objectID); err != nil {
			writeError(w, fmt.Errorf("invalid audit object id: %w", domain.ErrConflict))
			return
		}
	}
	filter := audit.Filter{
		ObjectType: r.URL.Query().Get("object_type"),
		ObjectID:   objectID,
		Action:     r.URL.Query().Get("action"),
		Outcome:    r.URL.Query().Get("outcome"),
	}
	page, err := a.service.QueryAudit(r.Context(), filter, from, to, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, page)
}

func requiredTimeQuery(r *http.Request, name string) (time.Time, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required: %w", name, domain.ErrConflict)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339: %w", name, domain.ErrConflict)
	}
	return parsed.UTC(), nil
}

func boundedIntQuery(r *http.Request, name string, fallback, minimum, maximum int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= minimum || parsed > maximum {
		return 0, fmt.Errorf("%s is out of range: %w", name, domain.ErrConflict)
	}
	return parsed, nil
}
