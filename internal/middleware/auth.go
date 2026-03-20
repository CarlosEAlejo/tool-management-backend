package middleware

import (
	"net/http"
	"strings"

	"tool_management_backend/internal/auth"
	"tool_management_backend/internal/handlers"
)

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(authorization, "Bearer ") {
			handlers.WriteJSONError(w, "unauthorized", "Debes iniciar sesion.", http.StatusUnauthorized)
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
		user, err := auth.NewService().ParseAccessToken(r.Context(), token)
		if err != nil {
			handlers.WriteJSONError(w, "unauthorized", "La sesion no es valida.", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, handlers.WithAuthenticatedUser(r, user))
	})
}

func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := handlers.GetAuthenticatedUser(r)
			if !ok {
				handlers.WriteJSONError(w, "unauthorized", "Debes iniciar sesion.", http.StatusUnauthorized)
				return
			}

			for _, role := range user.Roles {
				for _, allowedRole := range allowedRoles {
					if role == allowedRole {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			handlers.WriteJSONError(w, "forbidden", "No tienes permisos para acceder a este recurso.", http.StatusForbidden)
		})
	}
}
