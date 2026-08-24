package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"ufo-observation-command/internal/httpapi"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
	"github.com/google/uuid"
)

func TestAuditHTTPQueryFiltersAndPaginates(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	repo := repository.NewPostgres(pool)
	objectID := uuid.NewString()
	for index, action := range []string{"complete", "complete", "accept"} {
		if err := repo.WriteAudit(ctx, repository.AuditInput{
			RequestID:  fmt.Sprintf("audit-query-%d", index),
			ObjectType: "acoustic_buoy_task",
			ObjectID:   objectID,
			Action:     action,
			Outcome:    "success",
		}); err != nil {
			t.Fatal(err)
		}
	}
	handler := httpapi.New(service.New(repo), pool.Ping).Handler()
	now := time.Now().UTC()
	path := "/v1/audit?from=" + url.QueryEscape(now.Add(-time.Hour).Format(time.RFC3339)) +
		"&to=" + url.QueryEscape(now.Add(time.Hour).Format(time.RFC3339)) +
		"&object_type=acoustic_buoy_task&object_id=" + objectID + "&action=complete&limit=1&offset=0"
	var response struct {
		Data service.AuditPage `json:"data"`
	}
	apiRequest(t, handler, http.MethodGet, path, nil, "", http.StatusOK, &response)
	if len(response.Data.Items) != 1 || response.Data.Total != 2 || response.Data.Limit != 1 {
		t.Fatalf("page=%+v", response.Data)
	}
}

func TestAuditHTTPQueryRejectsInvalidRange(t *testing.T) {
	pool, _ := openDatabase(t)
	defer pool.Close()
	handler := httpapi.New(service.New(repository.NewPostgres(pool)), pool.Ping).Handler()
	apiRequest(t, handler, http.MethodGet, "/v1/audit?from=bad&to=also-bad", nil, "", http.StatusConflict, nil)
}

func TestAuditHTTPQueryReturnsEmptyArray(t *testing.T) {
	pool, _ := openDatabase(t)
	defer pool.Close()
	handler := httpapi.New(service.New(repository.NewPostgres(pool)), pool.Ping).Handler()
	now := time.Now().UTC()
	path := "/v1/audit?from=" + url.QueryEscape(now.Add(-time.Hour).Format(time.RFC3339)) +
		"&to=" + url.QueryEscape(now.Add(time.Hour).Format(time.RFC3339))
	var response struct {
		Data service.AuditPage `json:"data"`
	}
	apiRequest(t, handler, http.MethodGet, path, nil, "", http.StatusOK, &response)
	if response.Data.Items == nil || len(response.Data.Items) != 0 || response.Data.Total != 0 {
		t.Fatalf("page=%+v", response.Data)
	}
}
