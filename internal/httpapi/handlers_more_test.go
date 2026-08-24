package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBooleanQueryParsing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?enabled=true", nil)
	if !boolQuery(req, "enabled", false) {
		t.Fatal("true query parsed as false")
	}
	req = httptest.NewRequest(http.MethodGet, "/?enabled=0", nil)
	if boolQuery(req, "enabled", true) {
		t.Fatal("zero query parsed as true")
	}
}

func TestRequiredHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := requiredHeader(req, "X-ArrayOperator-ID"); err == nil {
		t.Fatal("missing header accepted")
	}
	req.Header.Set("X-ArrayOperator-ID", "op-1")
	if value, err := requiredHeader(req, "X-ArrayOperator-ID"); err != nil || value != "op-1" {
		t.Fatalf("value=%s err=%v", value, err)
	}
}

func TestArrayOperatorCreateMalformedBody(t *testing.T) {
	api := New(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/array_operators", strings.NewReader("{"))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d", res.Code)
	}
}
