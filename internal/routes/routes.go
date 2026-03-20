package routes

import (
	"net/http"

	"tool_management_backend/internal/handlers"
	"tool_management_backend/internal/middleware"
	"tool_management_backend/internal/models"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/auth/register", handlers.Register).Methods("POST")
	r.HandleFunc("/auth/login", handlers.Login).Methods("POST")
	r.Handle("/auth/me", middleware.RequireAuth(http.HandlerFunc(handlers.Me))).Methods("GET")
	r.HandleFunc("/auth/refresh", handlers.Refresh).Methods("POST")
	r.HandleFunc("/auth/logout", handlers.Logout).Methods("POST")

	tools := r.PathPrefix("/herramientas").Subrouter()
	tools.Use(middleware.RequireAuth)
	tools.Use(middleware.RequireRoles(models.RoleAdministrator))
	tools.HandleFunc("", handlers.CreateHerramienta).Methods("POST")
	tools.HandleFunc("", handlers.GetHerramientas).Methods("GET")
	tools.HandleFunc("/{id}", handlers.GetHerramientaByID).Methods("GET")
	tools.HandleFunc("/{id}", handlers.UpdateHerramienta).Methods("PUT")
	tools.HandleFunc("/{id}", handlers.DeleteHerramienta).Methods("DELETE")

	return r
}
