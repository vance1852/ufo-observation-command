package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ufo-observation-command/internal/console"
	"ufo-observation-command/internal/httpapi"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
)

type consoleResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func performConsoleRequest(t *testing.T, handler http.Handler, token, method, path string, body any) (*httptest.ResponseRecorder, consoleResponse) {
	t.Helper()
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, payload)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var envelope consoleResponse
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("%s %s response is not JSON: %s", method, path, response.Body.String())
	}
	return response, envelope
}

func consoleRequest(t *testing.T, handler http.Handler, token, method, path string, body any, target any) {
	t.Helper()
	response, envelope := performConsoleRequest(t, handler, token, method, path, body)
	if response.Code != http.StatusOK || envelope.Code != 200 || envelope.Message != "success" {
		t.Fatalf("%s %s status=%d response=%+v", method, path, response.Code, envelope)
	}
	if target != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			t.Fatal(err)
		}
	}
}

func loginConsole(t *testing.T, handler http.Handler, username, password string) (string, console.User) {
	t.Helper()
	var login struct {
		Token     string       `json:"token"`
		ExpiresAt time.Time    `json:"expiresAt"`
		User      console.User `json:"user"`
	}
	consoleRequest(t, handler, "", http.MethodPost, "/api/auth/login", map[string]any{"username": username, "password": password}, &login)
	if login.Token == "" || !login.ExpiresAt.After(time.Now()) {
		t.Fatalf("login=%+v", login)
	}
	return login.Token, login.User
}

func TestLoginSessionLifecycleAndRoleAuthorization(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	store := console.NewStore(pool)
	handler := httpapi.New(service.New(repository.NewPostgres(pool)), pool.Ping).WithConsole(store).Handler()

	response, _ := performConsoleRequest(t, handler, "", http.MethodGet, "/api/auth/info", nil)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous info status=%d", response.Code)
	}

	operatorToken, operator := loginConsole(t, handler, "operator", "repair123")
	if operator.Role != console.RoleArrayOperator {
		t.Fatalf("operator role=%s", operator.Role)
	}
	response, _ = performConsoleRequest(t, handler, operatorToken, http.MethodPost, "/api/acoustic_buoy", console.AcousticBuoy{Title: "unauthorized buoy", BuoyClass: 1, IntegrityStatus: 1, Status: 1})
	if response.Code != http.StatusForbidden {
		t.Fatalf("array_operator create acoustic_buoy status=%d", response.Code)
	}

	adminToken, admin := loginConsole(t, handler, "admin", "admin123")
	if admin.Role != console.RoleMissionLead {
		t.Fatalf("admin role=%s", admin.Role)
	}
	var info console.User
	consoleRequest(t, handler, adminToken, http.MethodGet, "/api/auth/info", nil, &info)
	if info.ID != admin.ID {
		t.Fatalf("session user=%+v login user=%+v", info, admin)
	}
	consoleRequest(t, handler, adminToken, http.MethodPost, "/api/auth/logout", nil, nil)
	response, _ = performConsoleRequest(t, handler, adminToken, http.MethodGet, "/api/auth/info", nil)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status=%d", response.Code)
	}

	reviewToken, _ := loginConsole(t, handler, "reviewer", "review123")
	if _, err := pool.Exec(ctx, `UPDATE console_sessions SET created_at=now()-interval '2 hours', expires_at=now()-interval '1 hour' WHERE user_id=(SELECT id FROM console_users WHERE username='reviewer')`); err != nil {
		t.Fatal(err)
	}
	response, _ = performConsoleRequest(t, handler, reviewToken, http.MethodGet, "/api/auth/info", nil)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status=%d", response.Code)
	}
}

func TestAuthenticatedAcousticWorkflowUsesPostgreSQL(t *testing.T) {
	pool, _ := openDatabase(t)
	defer pool.Close()
	store := console.NewStore(pool)
	handler := httpapi.New(service.New(repository.NewPostgres(pool)), pool.Ping).WithConsole(store).Handler()
	token, _ := loginConsole(t, handler, "admin", "admin123")

	acoustic_buoy := console.AcousticBuoy{Title: "contract hydrophone buoy", BuoyClass: 1, SerialNumber: "ABY-TEST-001", ShoreStationCode: "SHORE-T", MooringZone: "trench-T", IntegrityStatus: 1, Status: 1}
	consoleRequest(t, handler, token, http.MethodPost, "/api/acoustic_buoy", acoustic_buoy, &acoustic_buoy)
	if acoustic_buoy.ID == "" {
		t.Fatal("acoustic_buoy id is empty")
	}

	array_operator := console.ArrayOperator{Name: "contract dive operator", SpecialtyLevel: 2, Phone: "13800139999", Skills: "hydrophone recovery", Status: 1}
	consoleRequest(t, handler, token, http.MethodPost, "/api/array_operator", array_operator, &array_operator)
	treatment := console.Treatment{Name: "clock drift calibration", Description: "validate recovered acoustic clock", RiskBudget: 35, DurationMinutes: 30, Status: 1}
	consoleRequest(t, handler, token, http.MethodPost, "/api/treatment", treatment, &treatment)

	scheduledAt := time.Now().UTC().Add(time.Hour).Format("2006-01-02 15:04:05")
	var workOrder console.WorkOrder
	consoleRequest(t, handler, token, http.MethodPost, "/api/work_order", map[string]any{
		"acoustic_buoyId": acoustic_buoy.ID, "treatmentId": treatment.ID, "array_operatorId": array_operator.ID,
		"scheduledAt": scheduledAt, "remark": "接口测试",
	}, &workOrder)
	if workOrder.ID == "" || workOrder.Status != 0 {
		t.Fatalf("work order=%+v", workOrder)
	}
	consoleRequest(t, handler, token, http.MethodPut, "/api/work_order/status", map[string]any{"id": workOrder.ID, "status": 1}, nil)

	packetLoss, temperature, corruption := 0.2, 2.5, 0.1
	record := console.SignalRecoveryReport{AcousticBuoyID: acoustic_buoy.ID, PacketLossPercent: &packetLoss, WaterTemperatureC: &temperature, CorruptionIndex: &corruption, Remark: "recovered segment stable"}
	consoleRequest(t, handler, token, http.MethodPost, "/api/signal_recovery_report", record, &record)
	if record.ID == "" {
		t.Fatal("condition record id is empty")
	}

	var page console.Page[console.WorkOrder]
	consoleRequest(t, handler, token, http.MethodGet, "/api/work_order/page?current=1&size=20", nil, &page)
	if page.Total < 1 {
		t.Fatalf("work order page=%+v", page)
	}
	var stats console.DashboardStats
	consoleRequest(t, handler, token, http.MethodGet, "/api/dashboard/stats", nil, &stats)
	if stats.AcousticBuoyCount < 1 || stats.ArrayOperatorCount < 1 || stats.PendingWorkOrders < 1 {
		t.Fatalf("stats=%+v", stats)
	}
}
