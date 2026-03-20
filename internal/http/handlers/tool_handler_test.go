package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "tool_management_backend/internal/domain/tool"
	"tool_management_backend/internal/http/handlers"
	"tool_management_backend/internal/repository"

	"github.com/gorilla/mux"
)

type mockService struct {
	createFn func(ctx context.Context, input domain.Tool) (domain.Tool, error)
	listFn   func(ctx context.Context, filter domain.Filter) ([]domain.Tool, error)
	getFn    func(ctx context.Context, id string) (domain.Tool, error)
	updateFn func(ctx context.Context, id string, input domain.Tool) (domain.Tool, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m mockService) Create(ctx context.Context, input domain.Tool) (domain.Tool, error) {
	return m.createFn(ctx, input)
}
func (m mockService) List(ctx context.Context, filter domain.Filter) ([]domain.Tool, error) {
	return m.listFn(ctx, filter)
}
func (m mockService) GetByID(ctx context.Context, id string) (domain.Tool, error) {
	return m.getFn(ctx, id)
}
func (m mockService) Update(ctx context.Context, id string, input domain.Tool) (domain.Tool, error) {
	return m.updateFn(ctx, id, input)
}
func (m mockService) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func TestCreateReturnsValidationError(t *testing.T) {
	handler := handlers.NewToolHandler(mockService{
		createFn: func(ctx context.Context, input domain.Tool) (domain.Tool, error) {
			return domain.Tool{}, domain.ValidationError{Details: domain.ValidationErrors{"name": "required"}}
		},
		listFn:   defaultList,
		getFn:    defaultGet,
		updateFn: defaultUpdate,
		deleteFn: defaultDelete,
	})

	req := httptest.NewRequest(http.MethodPost, "/herramientas", bytes.NewBufferString(`{"code":"T-1"}`))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListReturnsTools(t *testing.T) {
	handler := handlers.NewToolHandler(mockService{
		createFn: defaultCreate,
		listFn: func(ctx context.Context, filter domain.Filter) ([]domain.Tool, error) {
			return []domain.Tool{{ID: "1", Name: "Taladro"}}, nil
		},
		getFn:    defaultGet,
		updateFn: defaultUpdate,
		deleteFn: defaultDelete,
	})

	req := httptest.NewRequest(http.MethodGet, "/herramientas?status=active", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload []domain.Tool
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(payload) != 1 || payload[0].Name != "Taladro" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestGetByIDReturnsNotFound(t *testing.T) {
	handler := handlers.NewToolHandler(mockService{
		createFn: defaultCreate,
		listFn:   defaultList,
		getFn: func(ctx context.Context, id string) (domain.Tool, error) {
			return domain.Tool{}, repository.ErrNotFound
		},
		updateFn: defaultUpdate,
		deleteFn: defaultDelete,
	})

	req := httptest.NewRequest(http.MethodGet, "/herramientas/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	rec := httptest.NewRecorder()

	handler.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeletePropagatesInternalError(t *testing.T) {
	handler := handlers.NewToolHandler(mockService{
		createFn: defaultCreate,
		listFn:   defaultList,
		getFn:    defaultGet,
		updateFn: defaultUpdate,
		deleteFn: func(ctx context.Context, id string) error {
			return errors.New("boom")
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/herramientas/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func defaultCreate(ctx context.Context, input domain.Tool) (domain.Tool, error) {
	return domain.Tool{}, nil
}
func defaultList(ctx context.Context, filter domain.Filter) ([]domain.Tool, error) {
	return []domain.Tool{}, nil
}
func defaultGet(ctx context.Context, id string) (domain.Tool, error) { return domain.Tool{}, nil }
func defaultUpdate(ctx context.Context, id string, input domain.Tool) (domain.Tool, error) {
	return domain.Tool{}, nil
}
func defaultDelete(ctx context.Context, id string) error { return nil }
