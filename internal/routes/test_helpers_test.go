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
)

const testDatabaseName = "herramientas_auth_integration_test"
const refreshCookieName = "tool_management_refresh_token"
const csrfCookieName = "tool_management_csrf_token"
const csrfHeaderName = "X-CSRF-Token"

type apiClient struct {
	httpClient *http.Client
}

type apiResponse struct {
	StatusCode int
	Body       map[string]any
	RawBody    []byte
	Cookies    []*http.Cookie
}

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

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cleanupTestDatabase(t)
	server := httptest.NewServer(routes.SetupRouter())
	t.Cleanup(server.Close)
	return server
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

func (r apiResponse) bodyArray(t *testing.T) []map[string]any {
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

func assertBodyContains(t *testing.T, response apiResponse, expected string) {
	t.Helper()
	if !strings.Contains(string(response.RawBody), expected) {
		t.Fatalf("expected body to contain %q, got %s", expected, string(response.RawBody))
	}
}

func registerAndLoginAdministrator(t *testing.T, server *httptest.Server, api *apiClient) string {
	t.Helper()

	register := api.request(t, server, "POST", "/auth/register", map[string]any{
		"email":           "admin@example.com",
		"password":        "admin123",
		"confirmPassword": "admin123",
	}, "", true)
	assertStatus(t, register, http.StatusCreated)

	login := api.request(t, server, "POST", "/auth/login", map[string]any{
		"email":    "admin@example.com",
		"password": "admin123",
	}, "", true)
	assertStatus(t, login, http.StatusOK)
	return login.mustString(t, "accessToken")
}

func createToolViaAPI(t *testing.T, server *httptest.Server, api *apiClient, accessToken string, overrides map[string]any) apiResponse {
	t.Helper()
	payload := map[string]any{
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
	}
	for key, value := range overrides {
		payload[key] = value
	}
	return api.request(t, server, "POST", "/herramientas", payload, accessToken, true)
}

func createWorkerViaAPI(t *testing.T, server *httptest.Server, api *apiClient, accessToken string, overrides map[string]any) apiResponse {
	t.Helper()
	payload := map[string]any{
		"firstName": "Carlos",
		"lastName":  "Operador",
		"position":  "Tecnico",
		"email":     "carlos.operador@example.com",
		"phone":     "5551234",
		"notes":     "",
	}
	for key, value := range overrides {
		payload[key] = value
	}
	return api.request(t, server, "POST", "/trabajadores", payload, accessToken, true)
}
