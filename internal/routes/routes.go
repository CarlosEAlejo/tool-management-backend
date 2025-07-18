package routes

import (
	"tool_management_backend/internal/handlers"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	// Rutas CRUD
	r.HandleFunc("/herramientas", handlers.CreateHerramienta).Methods("POST")
	r.HandleFunc("/herramientas", handlers.GetHerramientas).Methods("GET")
	r.HandleFunc("/herramientas/{id}", handlers.GetHerramientaByID).Methods("GET")
	r.HandleFunc("/herramientas/{id}", handlers.UpdateHerramienta).Methods("PUT")
	r.HandleFunc("/herramientas/{id}", handlers.DeleteHerramienta).Methods("DELETE")

	return r
}
