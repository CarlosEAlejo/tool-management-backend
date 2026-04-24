package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tool_management_backend/internal/auth"
)

type authRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var payload authRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	service := auth.NewService()
	result, err := service.Register(r.Context(), payload.Email, payload.Password, payload.ConfirmPassword)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailAlreadyUsed):
			WriteJSONError(w, "email_already_used", "Ese correo ya esta registrado.", http.StatusConflict)
		default:
			WriteJSONError(w, "validation_error", normalizeAuthError(err), http.StatusBadRequest)
		}
		return
	}

	setRefreshTokenCookie(w, r, result.RefreshToken, service.RefreshCookieTTL())
	setCSRFCookie(w, r, result.CSRFToken, service.RefreshCookieTTL())
	WriteJSON(w, http.StatusCreated, result.AuthResult)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var payload authRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	service := auth.NewService()
	result, err := service.Login(r.Context(), payload.Email, payload.Password)
	if err != nil {
		WriteJSONError(w, "invalid_credentials", "Correo o contrasena incorrectos.", http.StatusUnauthorized)
		return
	}

	setRefreshTokenCookie(w, r, result.RefreshToken, service.RefreshCookieTTL())
	setCSRFCookie(w, r, result.CSRFToken, service.RefreshCookieTTL())
	WriteJSON(w, http.StatusOK, result.AuthResult)
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := getRefreshTokenFromCookie(r)
	csrfCookieToken := getCSRFTokenFromCookie(r)
	csrfHeaderToken := getCSRFTokenFromHeader(r)
	if refreshToken == "" || csrfCookieToken == "" || csrfHeaderToken == "" || csrfCookieToken != csrfHeaderToken {
		clearAuthCookies(w, r)
		WriteJSONError(w, "invalid_csrf_token", "La solicitud de sesion no es valida.", http.StatusUnauthorized)
		return
	}

	service := auth.NewService()
	result, err := service.Refresh(r.Context(), refreshToken, csrfHeaderToken)
	if err != nil {
		clearAuthCookies(w, r)
		if errors.Is(err, auth.ErrInvalidCSRFToken) {
			WriteJSONError(w, "invalid_csrf_token", "La solicitud de sesion no es valida.", http.StatusUnauthorized)
			return
		}
		WriteJSONError(w, "invalid_refresh_token", "La sesion ya no es valida.", http.StatusUnauthorized)
		return
	}

	setRefreshTokenCookie(w, r, result.RefreshToken, service.RefreshCookieTTL())
	setCSRFCookie(w, r, result.CSRFToken, service.RefreshCookieTTL())
	WriteJSON(w, http.StatusOK, result.AuthResult)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := getRefreshTokenFromCookie(r)
	csrfCookieToken := getCSRFTokenFromCookie(r)
	csrfHeaderToken := getCSRFTokenFromHeader(r)
	if csrfCookieToken == "" || csrfHeaderToken == "" || csrfCookieToken != csrfHeaderToken {
		clearAuthCookies(w, r)
		WriteJSONError(w, "invalid_csrf_token", "La solicitud de sesion no es valida.", http.StatusUnauthorized)
		return
	}

	if err := auth.NewService().Logout(r.Context(), refreshToken, csrfHeaderToken); err != nil {
		clearAuthCookies(w, r)
		if errors.Is(err, auth.ErrInvalidCSRFToken) {
			WriteJSONError(w, "invalid_csrf_token", "La solicitud de sesion no es valida.", http.StatusUnauthorized)
			return
		}
		WriteJSONError(w, "logout_failed", "No se pudo cerrar la sesion.", http.StatusInternalServerError)
		return
	}

	clearAuthCookies(w, r)
	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func Me(w http.ResponseWriter, r *http.Request) {
	user, ok := GetAuthenticatedUser(r)
	if !ok {
		WriteJSONError(w, "unauthorized", "Debes iniciar sesion.", http.StatusUnauthorized)
		return
	}

	WriteJSON(w, http.StatusOK, user.Public())
}

func normalizeAuthError(err error) string {
	message := strings.TrimSpace(err.Error())
	switch message {
	case "invalid email":
		return "Debes escribir un correo valido."
	case "password must be at least 8 characters":
		return "La contrasena debe tener al menos 8 caracteres."
	case "passwords do not match":
		return "Las contrasenas no coinciden."
	default:
		return "No se pudo procesar la solicitud."
	}
}


