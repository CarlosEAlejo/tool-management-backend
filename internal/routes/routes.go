package routes

import (
	"net/http"

	"tool_management_backend/internal/handlers"
	"tool_management_backend/internal/middleware"
	"tool_management_backend/internal/models"
)

func SetupRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", handlers.Register)
	mux.HandleFunc("POST /auth/login", handlers.Login)
	mux.Handle("GET /auth/me", middleware.RequireAuth(http.HandlerFunc(handlers.Me)))
	mux.HandleFunc("POST /auth/refresh", handlers.Refresh)
	mux.HandleFunc("POST /auth/logout", handlers.Logout)

	toolMiddlewares := []func(http.Handler) http.Handler{
		middleware.RequireAuth,
		middleware.RequireRoles(models.RoleAdministrator),
	}

	handleWithMiddleware(mux, "POST /herramientas", http.HandlerFunc(handlers.CreateHerramienta), toolMiddlewares...)
	handleWithMiddleware(mux, "GET /herramientas", http.HandlerFunc(handlers.GetHerramientas), toolMiddlewares...)
	handleWithMiddleware(mux, "GET /herramientas/{id}", http.HandlerFunc(handlers.GetHerramientaByID), toolMiddlewares...)
	handleWithMiddleware(mux, "PUT /herramientas/{id}", http.HandlerFunc(handlers.UpdateHerramienta), toolMiddlewares...)
	handleWithMiddleware(mux, "DELETE /herramientas/{id}", http.HandlerFunc(handlers.DeleteHerramienta), toolMiddlewares...)

	return mux
}

func handleWithMiddleware(mux *http.ServeMux, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) {
	mux.Handle(pattern, applyMiddlewares(handler, middlewares...))
}

func applyMiddlewares(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}
