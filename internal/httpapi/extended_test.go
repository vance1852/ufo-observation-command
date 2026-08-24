package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDIsPreserved(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "client-request")
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Header().Get("X-Request-ID") != "client-request" {
		t.Fatalf("request id=%s", res.Header().Get("X-Request-ID"))
	}
}

func TestMalformedBodyDoesNotInvokeService(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/recovery_jobs", strings.NewReader("not-json"))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestDiveWindowVersionQueryRejectsMissingValue(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/dive_windows/b1/start", nil)
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestAssignmentHandlerRejectsUnknownStatusBeforeService(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/assignments/a1/advance", strings.NewReader(`{"status":"unknown","version":1}`))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
