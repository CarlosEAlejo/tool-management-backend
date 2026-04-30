package routes_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"tool_management_backend/internal/database"
)

func TestMongoConnectionSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.GetClient().Ping(ctx, nil); err != nil {
		t.Fatalf("expected mongo ping to succeed, got %v", err)
	}
}

func TestAuthLifecycleAndSessionHardening(t *testing.T) {
	server := newTestServer(t)
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
	assertStatus(t, secondRegister, http.StatusCreated)
	secondRoles := secondRegister.Body["user"].(map[string]any)["roles"].([]any)
	if len(secondRoles) != 1 || secondRoles[0] != "administrator" {
		t.Fatalf("expected second registered user to be administrator, got %#v", secondRoles)
	}

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
