package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	domain "tool_management_backend/internal/domain/tool"
	"tool_management_backend/internal/http/response"
	"tool_management_backend/internal/repository"
	service "tool_management_backend/internal/service/tool"

	"github.com/gorilla/mux"
)

type ToolHandler struct {
	service service.Service
}

func NewToolHandler(service service.Service) *ToolHandler {
	return &ToolHandler{service: service}
}

func (h *ToolHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.Tool
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, response.APIError{
			Message: "Payload invalido",
			Code:    "invalid_json",
		})
		return
	}

	created, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

func (h *ToolHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.Filter{
		Search:      r.URL.Query().Get("search"),
		Status:      r.URL.Query().Get("status"),
		Responsible: r.URL.Query().Get("responsible"),
		Type:        r.URL.Query().Get("type"),
		Location:    r.URL.Query().Get("location"),
	}

	tools, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.writeError(w, err)
		return
	}

	if tools == nil {
		tools = []domain.Tool{}
	}

	response.JSON(w, http.StatusOK, tools)
}

func (h *ToolHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	tool, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, tool)
}

func (h *ToolHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input domain.Tool
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, response.APIError{
			Message: "Payload invalido",
			Code:    "invalid_json",
		})
		return
	}

	updated, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, updated)
}

func (h *ToolHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.service.Delete(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ToolHandler) writeError(w http.ResponseWriter, err error) {
	if validationErr, ok := service.IsValidationError(err); ok {
		response.Error(w, http.StatusBadRequest, response.APIError{
			Message: "Los datos enviados no son validos",
			Code:    "validation_error",
			Details: validationErr.Details,
		})
		return
	}

	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, response.APIError{
			Message: "Herramienta no encontrada",
			Code:    "tool_not_found",
		})
		return
	}

	response.Error(w, http.StatusInternalServerError, response.APIError{
		Message: err.Error(),
		Code:    "internal_error",
	})
}
