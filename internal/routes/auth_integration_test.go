package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/routes"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const testDatabaseName = "herramientas_auth_integration_test"

func TestMain(m *testing.M) {
	_ = os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	_ = os.Setenv("MONGODB_DATABASE", testDatabaseName)
	_ = os.Setenv("JWT_ACCESS_SECRET", "integration-access-secret")
	_ = os.Setenv("JWT_REFRESH_SECRET", "integration-refresh-secret")
	_ = os.Setenv("JWT_ACCESS_TTL", "15m")
	_ = os.Setenv("JWT_REFRESH_TTL", "24h")

	database.ConnectDB()
	code := m.Run()
	_ = database.GetDatabase().Drop(context.Background())
	database.DisconnectDB()
	os.Exit(code)
}

func TestMongoConnectionSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.GetClient().Ping(ctx, nil); err != nil {
		t.Fatalf("expected mongo ping to succeed, got %v", err)
	}
}

func TestAuthBootstrapAndProtectedRoutes(t *testing.T) {
	cleanupTestDatabase(t)

	server := httptest.NewServer(routes.SetupRouter())
	defer server.Close()

	unauthorizedTools := apiRequest(t, server, "GET", "/herramientas", nil, "")
	assertStatus(t, unauthorizedTools, http.StatusUnauthorized)
	assertCode(t, unauthorizedTools, "unauthorized")

	register := apiRequest(t, server, "POST", "/auth/register", map[string]any{
		"email":           "admin@example.com",
		"password":        "admin123",
		"confirmPassword": "admin123",
	}, "")
	assertStatus(t, register, http.StatusCreated)
	assertString(t, register.Body["user"].(map[string]any)["email"], "admin@example.com")

	secondRegister := apiRequest(t, server, "POST", "/auth/register", map[string]any{
		"email":           "second@example.com",
		"password":        "admin12345",
		"confirmPassword": "admin12345",
	}, "")
	assertStatus(t, secondRegister, http.StatusForbidden)
	assertCode(t, secondRegister, "registration_closed")

	invalidLogin := apiRequest(t, server, "POST", "/auth/login", map[string]any{
		"email":    "admin@example.com",
		"password": "bad-password",
	}, "")
	assertStatus(t, invalidLogin, http.StatusUnauthorized)
	assertCode(t, invalidLogin, "invalid_credentials")

	login := apiRequest(t, server, "POST", "/auth/login", map[string]any{
		"email":    "admin@example.com",
		"password": "admin123",
	}, "")
	assertStatus(t, login, http.StatusOK)
	accessToken := login.mustString(t, "accessToken")
	refreshToken := login.mustString(t, "refreshToken")

	me := apiRequest(t, server, "GET", "/auth/me", nil, accessToken)
	assertStatus(t, me, http.StatusOK)
	roles := me.Body["roles"].([]any)
	if len(roles) != 1 || roles[0] != "administrator" {
		t.Fatalf("expected administrator role, got %#v", roles)
	}

	meMethodNotAllowed := apiRequest(t, server, "POST", "/auth/me", nil, accessToken)
	assertStatus(t, meMethodNotAllowed, http.StatusMethodNotAllowed)

	createTool := apiRequest(t, server, "POST", "/herramientas", map[string]any{
		"code":            "AUTH-001",
		"name":            "Taladro Test",
		"type":            "electric",
		"status":          "active",
		"responsible":     "",
		"assignmentDate":  "",
		"dateMaintenance": "",
		"nextMaintenance": "",
		"location":        "Almacen QA",
		"notes":           "Prueba automatizada",
		"deterioration":   false,
	}, accessToken)
	assertStatus(t, createTool, http.StatusCreated)
	toolID := createTool.mustString(t, "id")

	listTools := apiRequest(t, server, "GET", "/herramientas", nil, accessToken)
	assertStatus(t, listTools, http.StatusOK)
	tools := listTools.BodyArray(t)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	getTool := apiRequest(t, server, "GET", "/herramientas/"+toolID, nil, accessToken)
	assertStatus(t, getTool, http.StatusOK)
	assertString(t, getTool.Body["id"], toolID)
	assertString(t, getTool.Body["name"], "Taladro Test")

	missingToolID := bson.NewObjectID().Hex()
	missingTool := apiRequest(t, server, "GET", "/herramientas/"+missingToolID, nil, accessToken)
	assertStatus(t, missingTool, http.StatusNotFound)

	updatedTool := apiRequest(t, server, "PUT", "/herramientas/"+toolID, map[string]any{
		"id":              toolID,
		"code":            "AUTH-001",
		"name":            "Taladro Test Actualizado",
		"type":            "electric",
		"status":          "assigned",
		"responsible":     "Carlos",
		"assignmentDate":  "2026-03-21",
		"dateMaintenance": "",
		"nextMaintenance": "",
		"location":        "Almacen QA",
		"notes":           "Prueba automatizada",
		"deterioration":   false,
	}, accessToken)
	assertStatus(t, updatedTool, http.StatusOK)
	assertString(t, updatedTool.Body["name"], "Taladro Test Actualizado")

	getUpdatedTool := apiRequest(t, server, "GET", "/herramientas/"+toolID, nil, accessToken)
	assertStatus(t, getUpdatedTool, http.StatusOK)
	assertString(t, getUpdatedTool.Body["responsible"], "Carlos")
	assignmentHistory := getUpdatedTool.Body["assignmentHistory"].([]any)
	if len(assignmentHistory) != 1 {
		t.Fatalf("expected assignment history to contain 1 record, got %d", len(assignmentHistory))
	}

	toolMethodNotAllowed := apiRequest(t, server, "PATCH", "/herramientas/"+toolID, nil, accessToken)
	assertStatus(t, toolMethodNotAllowed, http.StatusMethodNotAllowed)

	deletedTool := apiRequest(t, server, "DELETE", "/herramientas/"+toolID, nil, accessToken)
	assertStatus(t, deletedTool, http.StatusNoContent)

	getDeletedTool := apiRequest(t, server, "GET", "/herramientas/"+toolID, nil, accessToken)
	assertStatus(t, getDeletedTool, http.StatusNotFound)

	missingRoute := apiRequest(t, server, "GET", "/ruta-inexistente", nil, accessToken)
	assertStatus(t, missingRoute, http.StatusNotFound)

	refresh := apiRequest(t, server, "POST", "/auth/refresh", map[string]any{
		"refreshToken": refreshToken,
	}, "")
	assertStatus(t, refresh, http.StatusOK)
	rotatedRefreshToken := refresh.mustString(t, "refreshToken")
	if rotatedRefreshToken == refreshToken {
		t.Fatal("expected rotated refresh token to differ from original")
	}

	oldRefreshReuse := apiRequest(t, server, "POST", "/auth/refresh", map[string]any{
		"refreshToken": refreshToken,
	}, "")
	assertStatus(t, oldRefreshReuse, http.StatusUnauthorized)
	assertCode(t, oldRefreshReuse, "invalid_refresh_token")

	logout := apiRequest(t, server, "POST", "/auth/logout", map[string]any{
		"refreshToken": rotatedRefreshToken,
	}, "")
	assertStatus(t, logout, http.StatusOK)

	refreshAfterLogout := apiRequest(t, server, "POST", "/auth/refresh", map[string]any{
		"refreshToken": rotatedRefreshToken,
	}, "")
	assertStatus(t, refreshAfterLogout, http.StatusUnauthorized)
	assertCode(t, refreshAfterLogout, "invalid_refresh_token")
}

type apiResponse struct {
	StatusCode int
	Body       map[string]any
	RawBody    []byte
}

func (r apiResponse) mustString(t *testing.T, key string) string {
	t.Helper()
	value, ok := r.Body[key].(string)
	if !ok {
		t.Fatalf("expected string for key %s, got %#v", key, r.Body[key])
	}
	return value
}

func (r apiResponse) BodyArray(t *testing.T) []map[string]any {
	t.Helper()
	var body []map[string]any
	if err := json.Unmarshal(r.RawBody, &body); err != nil {
		t.Fatalf("failed to decode array body: %v\nbody: %s", err, string(r.RawBody))
	}
	return body
}

func apiRequest(t *testing.T, server *httptest.Server, method string, path string, payload any, token string) apiResponse {
	t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	decoded := map[string]any{}
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &decoded)
	}

	return apiResponse{
		StatusCode: resp.StatusCode,
		Body:       decoded,
		RawBody:    rawBody,
	}
}

func cleanupTestDatabase(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := database.GetDatabase().Drop(ctx); err != nil {
		t.Fatalf("failed to drop test database: %v", err)
	}
}

func assertStatus(t *testing.T, response apiResponse, expected int) {
	t.Helper()
	if response.StatusCode != expected {
		t.Fatalf("expected status %d, got %d, body=%s", expected, response.StatusCode, string(response.RawBody))
	}
}

func assertCode(t *testing.T, response apiResponse, expected string) {
	t.Helper()
	assertString(t, response.Body["code"], expected)
}

func assertString(t *testing.T, value any, expected string) {
	t.Helper()
	actual, ok := value.(string)
	if !ok {
		t.Fatalf("expected string %q, got %#v", expected, value)
	}
	if actual != expected {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
