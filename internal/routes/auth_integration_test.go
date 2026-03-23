package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/routes"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const testDatabaseName = "herramientas_auth_integration_test"
const refreshCookieName = "tool_management_refresh_token"
const csrfCookieName = "tool_management_csrf_token"
const csrfHeaderName = "X-CSRF-Token"

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

	api := newAPIClient(t)

	unauthorizedTools := api.request(t, server, "GET", "/herramientas", nil, "", true)
	assertStatus(t, unauthorizedTools, http.StatusUnauthorized)
	assertCode(t, unauthorizedTools, "unauthorized")

	register := api.request(t, server, "POST", "/auth/register", map[string]any{
		"email":           "admin@example.com",
		"password":        "admin123",
		"confirmPassword": "admin123",
	}, "", true)
	assertStatus(t, register, http.StatusCreated)
	assertString(t, register.Body["user"].(map[string]any)["email"], "admin@example.com")
	assertCookiePresent(t, register, refreshCookieName, true)
	assertCookiePresent(t, register, csrfCookieName, false)

	secondRegister := api.request(t, server, "POST", "/auth/register", map[string]any{
		"email":           "second@example.com",
		"password":        "admin12345",
		"confirmPassword": "admin12345",
	}, "", true)
	assertStatus(t, secondRegister, http.StatusForbidden)
	assertCode(t, secondRegister, "registration_closed")

	invalidLogin := api.request(t, server, "POST", "/auth/login", map[string]any{
		"email":    "admin@example.com",
		"password": "bad-password",
	}, "", true)
	assertStatus(t, invalidLogin, http.StatusUnauthorized)
	assertCode(t, invalidLogin, "invalid_credentials")

	login := api.request(t, server, "POST", "/auth/login", map[string]any{
		"email":    "admin@example.com",
		"password": "admin123",
	}, "", true)
	assertStatus(t, login, http.StatusOK)
	accessToken := login.mustString(t, "accessToken")
	firstRefreshCookie := assertCookiePresent(t, login, refreshCookieName, true)
	firstCSRFCookie := assertCookiePresent(t, login, csrfCookieName, false)

	me := api.request(t, server, "GET", "/auth/me", nil, accessToken, true)
	assertStatus(t, me, http.StatusOK)
	roles := me.Body["roles"].([]any)
	if len(roles) != 1 || roles[0] != "administrator" {
		t.Fatalf("expected administrator role, got %#v", roles)
	}

	meMethodNotAllowed := api.request(t, server, "POST", "/auth/me", nil, accessToken, true)
	assertStatus(t, meMethodNotAllowed, http.StatusMethodNotAllowed)

	createTool := api.request(t, server, "POST", "/herramientas", map[string]any{
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
	}, accessToken, true)
	assertStatus(t, createTool, http.StatusCreated)
	toolID := createTool.mustString(t, "id")

	listTools := api.request(t, server, "GET", "/herramientas", nil, accessToken, true)
	assertStatus(t, listTools, http.StatusOK)
	tools := listTools.BodyArray(t)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	getTool := api.request(t, server, "GET", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, getTool, http.StatusOK)
	assertString(t, getTool.Body["id"], toolID)
	assertString(t, getTool.Body["name"], "Taladro Test")

	missingToolID := bson.NewObjectID().Hex()
	missingTool := api.request(t, server, "GET", "/herramientas/"+missingToolID, nil, accessToken, true)
	assertStatus(t, missingTool, http.StatusNotFound)

	updatedTool := api.request(t, server, "PUT", "/herramientas/"+toolID, map[string]any{
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
	}, accessToken, true)
	assertStatus(t, updatedTool, http.StatusOK)
	assertString(t, updatedTool.Body["name"], "Taladro Test Actualizado")

	getUpdatedTool := api.request(t, server, "GET", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, getUpdatedTool, http.StatusOK)
	assertString(t, getUpdatedTool.Body["responsible"], "Carlos")
	assignmentHistory := getUpdatedTool.Body["assignmentHistory"].([]any)
	if len(assignmentHistory) != 1 {
		t.Fatalf("expected assignment history to contain 1 record, got %d", len(assignmentHistory))
	}

	toolMethodNotAllowed := api.request(t, server, "PATCH", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, toolMethodNotAllowed, http.StatusMethodNotAllowed)

	deletedTool := api.request(t, server, "DELETE", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, deletedTool, http.StatusNoContent)

	getDeletedTool := api.request(t, server, "GET", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, getDeletedTool, http.StatusNotFound)

	missingRoute := api.request(t, server, "GET", "/ruta-inexistente", nil, accessToken, true)
	assertStatus(t, missingRoute, http.StatusNotFound)

	refreshWithoutCSRF := api.request(t, server, "POST", "/auth/refresh", map[string]any{}, "", false)
	assertStatus(t, refreshWithoutCSRF, http.StatusUnauthorized)
	assertCode(t, refreshWithoutCSRF, "invalid_csrf_token")
	assertCookieCleared(t, refreshWithoutCSRF, refreshCookieName)
	assertCookieCleared(t, refreshWithoutCSRF, csrfCookieName)

	api.setCookie(t, server, firstRefreshCookie)
	api.setCookie(t, server, firstCSRFCookie)
	refresh := api.request(t, server, "POST", "/auth/refresh", map[string]any{}, "", true)
	assertStatus(t, refresh, http.StatusOK)
	secondRefreshCookie := assertCookiePresent(t, refresh, refreshCookieName, true)
	secondCSRFCookie := assertCookiePresent(t, refresh, csrfCookieName, false)
	if secondRefreshCookie.Value == firstRefreshCookie.Value {
		t.Fatal("expected rotated refresh cookie to differ from original")
	}
	if secondCSRFCookie.Value == firstCSRFCookie.Value {
		t.Fatal("expected rotated csrf cookie to differ from original")
	}
	rotatedAccessToken := refresh.mustString(t, "accessToken")

	api.setCookie(t, server, firstRefreshCookie)
	api.setCookie(t, server, firstCSRFCookie)
	oldRefreshReuse := api.request(t, server, "POST", "/auth/refresh", map[string]any{}, "", true)
	assertStatus(t, oldRefreshReuse, http.StatusUnauthorized)
	assertCode(t, oldRefreshReuse, "invalid_refresh_token")
	assertCookieCleared(t, oldRefreshReuse, refreshCookieName)
	assertCookieCleared(t, oldRefreshReuse, csrfCookieName)

	api.setCookie(t, server, secondRefreshCookie)
	api.setCookie(t, server, secondCSRFCookie)
	logout := api.request(t, server, "POST", "/auth/logout", map[string]any{}, "", true)
	assertStatus(t, logout, http.StatusOK)
	assertCookieCleared(t, logout, refreshCookieName)
	assertCookieCleared(t, logout, csrfCookieName)

	api.setCookie(t, server, secondRefreshCookie)
	api.setCookie(t, server, secondCSRFCookie)
	logoutWithoutCSRF := api.request(t, server, "POST", "/auth/logout", map[string]any{}, "", false)
	assertStatus(t, logoutWithoutCSRF, http.StatusUnauthorized)
	assertCode(t, logoutWithoutCSRF, "invalid_csrf_token")

	api.setCookie(t, server, secondRefreshCookie)
	api.setCookie(t, server, secondCSRFCookie)
	refreshAfterLogout := api.request(t, server, "POST", "/auth/refresh", map[string]any{}, "", true)
	assertStatus(t, refreshAfterLogout, http.StatusUnauthorized)
	assertCode(t, refreshAfterLogout, "invalid_refresh_token")
	assertCookieCleared(t, refreshAfterLogout, refreshCookieName)
	assertCookieCleared(t, refreshAfterLogout, csrfCookieName)

	refreshedMe := api.request(t, server, "GET", "/auth/me", nil, rotatedAccessToken, true)
	assertStatus(t, refreshedMe, http.StatusOK)
}

type apiClient struct {
	httpClient *http.Client
}

type apiResponse struct {
	StatusCode int
	Body       map[string]any
	RawBody    []byte
	Cookies    []*http.Cookie
}

func newAPIClient(t *testing.T) *apiClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	return &apiClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Jar:     jar,
		},
	}
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

func (c *apiClient) request(t *testing.T, server *httptest.Server, method string, path string, payload any, token string, includeCSRF bool) apiResponse {
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
	if includeCSRF {
		if csrfToken := c.csrfToken(t, server); csrfToken != "" {
			req.Header.Set(csrfHeaderName, csrfToken)
		}
	}

	resp, err := c.httpClient.Do(req)
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
		Cookies:    resp.Cookies(),
	}
}

func (c *apiClient) csrfToken(t *testing.T, server *httptest.Server) string {
	t.Helper()
	serverURL := parseServerURL(t, server)
	for _, cookie := range c.httpClient.Jar.Cookies(serverURL) {
		if cookie.Name == csrfCookieName {
			return cookie.Value
		}
	}
	return ""
}

func (c *apiClient) setCookie(t *testing.T, server *httptest.Server, cookie *http.Cookie) {
	t.Helper()
	serverURL := parseServerURL(t, server)
	c.httpClient.Jar.SetCookies(serverURL, []*http.Cookie{cookie})
}

func parseServerURL(t *testing.T, server *httptest.Server) *url.URL {
	t.Helper()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server url: %v", err)
	}
	return serverURL
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

func assertCookiePresent(t *testing.T, response apiResponse, name string, expectHTTPOnly bool) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies {
		if cookie.Name == name {
			if cookie.Value == "" {
				t.Fatalf("expected cookie %s to have a value", name)
			}
			if cookie.HttpOnly != expectHTTPOnly {
				t.Fatalf("expected cookie %s HttpOnly=%t, got %t", name, expectHTTPOnly, cookie.HttpOnly)
			}
			return cookie
		}
	}

	t.Fatalf("expected cookie %s to be present", name)
	return nil
}

func assertCookieCleared(t *testing.T, response apiResponse, name string) {
	t.Helper()
	for _, cookie := range response.Cookies {
		if cookie.Name == name {
			if cookie.MaxAge >= 0 && !strings.EqualFold(cookie.Value, "") {
				t.Fatalf("expected cookie %s to be cleared, got maxAge=%d value=%q", name, cookie.MaxAge, cookie.Value)
			}
			return
		}
	}

	t.Fatalf("expected cookie %s clearing instruction", name)
}
