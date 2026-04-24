package routes

import (
	"net/http"
	"strings"

	"tool_management_backend/internal/handlers"
	"tool_management_backend/internal/middleware"
	"tool_management_backend/internal/models"
)

func SetupRouter() http.Handler {
	mux := http.NewServeMux()

	handleFuncWithAPIPrefix(mux, "POST /auth/register", handlers.Register)
	handleFuncWithAPIPrefix(mux, "POST /auth/login", handlers.Login)
	handleWithMiddlewareAndAPIPrefix(mux, "GET /auth/me", http.HandlerFunc(handlers.Me), middleware.RequireAuth)
	handleFuncWithAPIPrefix(mux, "POST /auth/refresh", handlers.Refresh)
	handleFuncWithAPIPrefix(mux, "POST /auth/logout", handlers.Logout)

	adminMiddlewares := []func(http.Handler) http.Handler{
		middleware.RequireAuth,
		middleware.RequireRoles(models.RoleAdministrator),
	}

	handleWithMiddlewareAndAPIPrefix(mux, "POST /herramientas", http.HandlerFunc(handlers.CreateHerramienta), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "GET /herramientas", http.HandlerFunc(handlers.GetHerramientas), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "GET /herramientas/{id}", http.HandlerFunc(handlers.GetHerramientaByID), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "PUT /herramientas/{id}", http.HandlerFunc(handlers.UpdateHerramienta), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "DELETE /herramientas/{id}", http.HandlerFunc(handlers.DeleteHerramienta), adminMiddlewares...)

	handleWithMiddlewareAndAPIPrefix(mux, "POST /trabajadores", http.HandlerFunc(handlers.CreateTrabajador), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "GET /trabajadores", http.HandlerFunc(handlers.GetTrabajadores), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "GET /trabajadores/{id}", http.HandlerFunc(handlers.GetTrabajadorByID), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "PUT /trabajadores/{id}", http.HandlerFunc(handlers.UpdateTrabajador), adminMiddlewares...)
	handleWithMiddlewareAndAPIPrefix(mux, "DELETE /trabajadores/{id}", http.HandlerFunc(handlers.DeleteTrabajador), adminMiddlewares...)

	return mux
}

func handleWithMiddleware(mux *http.ServeMux, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) {
	mux.Handle(pattern, applyMiddlewares(handler, middlewares...))
}

func handleFuncWithAPIPrefix(mux *http.ServeMux, pattern string, handler http.HandlerFunc) {
	mux.HandleFunc(pattern, handler)
	mux.HandleFunc(withAPIPrefix(pattern), handler)
}

func handleWithMiddlewareAndAPIPrefix(mux *http.ServeMux, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) {
	handleWithMiddleware(mux, pattern, handler, middlewares...)
	handleWithMiddleware(mux, withAPIPrefix(pattern), handler, middlewares...)
}

func withAPIPrefix(pattern string) string {
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) != 2 {
		if strings.HasPrefix(pattern, "/api/") {
			return pattern
		}
		return "/api" + pattern
	}

	method := strings.TrimSpace(parts[0])
	path := strings.TrimSpace(parts[1])
	if strings.HasPrefix(path, "/api/") {
		return pattern
	}

	return method + " /api" + path
}

func applyMiddlewares(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}
