package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"tool_management_backend/internal/models"
)

type contextKey string

const authenticatedUserKey contextKey = "authenticated-user"

func WithAuthenticatedUser(r *http.Request, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedUserKey, user)
	return r.WithContext(ctx)
}

func GetAuthenticatedUser(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(authenticatedUserKey).(*models.User)
	return user, ok
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteJSONError(w http.ResponseWriter, code string, message string, status int) {
	WriteJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
	})
}
