package router

import (
	"net/http"
	"tool_management_backend/internal/http/handlers"
	"tool_management_backend/internal/http/middleware"

	"github.com/gorilla/mux"
)

func New(toolHandler *handlers.ToolHandler) http.Handler {
	r := mux.NewRouter()

	r.Use(middleware.Recovery)
	r.Use(middleware.Logging)

	r.HandleFunc("/herramientas", toolHandler.Create).Methods(http.MethodPost)
	r.HandleFunc("/herramientas", toolHandler.List).Methods(http.MethodGet)
	r.HandleFunc("/herramientas/{id}", toolHandler.GetByID).Methods(http.MethodGet)
	r.HandleFunc("/herramientas/{id}", toolHandler.Update).Methods(http.MethodPut)
	r.HandleFunc("/herramientas/{id}", toolHandler.Delete).Methods(http.MethodDelete)

	return r
}
