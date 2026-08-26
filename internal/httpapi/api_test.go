package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthIncludesAliveStatus(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"alive"`) {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if res.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id missing")
	}
}

func TestReadyReportsDependencyFailure(t *testing.T) {
	api := New(nil, func(context.Context) error { return errors.New("database unavailable") })
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestMalformedJSONUsesConflictResponse(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/cases", strings.NewReader("{"))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "conflict") {
		t.Fatalf("body=%s", res.Body.String())
	}
}

func TestUnknownRouteReturnsNotFound(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestUnknownJSONFieldIsRejected(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/array_operators", strings.NewReader(`{"name":"AcousticBuoyOperator","role":"acoustic_buoy_operator","unexpected":true}`))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestMultipleJSONValuesAreRejected(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/array_operators", strings.NewReader(`{"name":"AcousticBuoyOperator","role":"acoustic_buoy_operator"} {}`))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
