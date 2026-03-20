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

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var payload authRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	result, err := auth.NewService().Register(r.Context(), payload.Email, payload.Password, payload.ConfirmPassword)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrRegistrationClosed):
			WriteJSONError(w, "registration_closed", "El registro ya no esta disponible.", http.StatusForbidden)
		case errors.Is(err, auth.ErrEmailAlreadyUsed):
			WriteJSONError(w, "email_already_used", "Ese correo ya esta registrado.", http.StatusConflict)
		default:
			WriteJSONError(w, "validation_error", normalizeAuthError(err), http.StatusBadRequest)
		}
		return
	}

	WriteJSON(w, http.StatusCreated, result)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var payload authRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	result, err := auth.NewService().Login(r.Context(), payload.Email, payload.Password)
	if err != nil {
		WriteJSONError(w, "invalid_credentials", "Correo o contrasena incorrectos.", http.StatusUnauthorized)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	var payload refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	result, err := auth.NewService().Refresh(r.Context(), payload.RefreshToken)
	if err != nil {
		WriteJSONError(w, "invalid_refresh_token", "La sesion ya no es valida.", http.StatusUnauthorized)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	var payload refreshRequest
	_ = json.NewDecoder(r.Body).Decode(&payload)

	if err := auth.NewService().Logout(r.Context(), payload.RefreshToken); err != nil {
		WriteJSONError(w, "logout_failed", "No se pudo cerrar la sesion.", http.StatusInternalServerError)
		return
	}

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
