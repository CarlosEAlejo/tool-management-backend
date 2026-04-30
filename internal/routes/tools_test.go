package routes_test

import (
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestToolsCRUDAndEditRestrictions(t *testing.T) {
	server := newTestServer(t)
	api := newAPIClient(t)
	accessToken := registerAndLoginAdministrator(t, server, api)

	createTool := createToolViaAPI(t, server, api, accessToken, nil)
	assertStatus(t, createTool, http.StatusCreated)
	toolID := createTool.mustString(t, "id")

	filteredTools := api.request(t, server, "GET", "/herramientas?search=taladro&status=active", nil, accessToken, true)
	assertStatus(t, filteredTools, http.StatusOK)
	if len(filteredTools.bodyArray(t)) != 1 {
		t.Fatalf("expected filtered tools list to return 1 tool, got %d", len(filteredTools.bodyArray(t)))
	}

	listTools := api.request(t, server, "GET", "/herramientas", nil, accessToken, true)
	assertStatus(t, listTools, http.StatusOK)
	if len(listTools.bodyArray(t)) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(listTools.bodyArray(t)))
	}

	updateTool := api.request(t, server, "PUT", "/herramientas/"+toolID, map[string]any{
		"code":              "AUTH-001",
		"name":              "Taladro QA",
		"type":              "electric",
		"status":            "active",
		"responsibleId":     "",
		"responsible":       "",
		"assignmentDate":    "",
		"dateMaintenance":   "",
		"nextMaintenance":   "",
		"purchaseDate":      "",
		"price":             0,
		"location":          "Deposito Central",
		"notes":             "Actualizado por integration test",
		"deterioration":     false,
		"assignmentHistory": []any{},
		"maintenanceRecord": []any{},
	}, accessToken, true)
	assertStatus(t, updateTool, http.StatusOK)
	assertString(t, updateTool.Body["name"], "Taladro QA")
	assertString(t, updateTool.Body["location"], "Deposito Central")

	restrictedToolUpdate := api.request(t, server, "PUT", "/herramientas/"+toolID, map[string]any{
		"code":              "AUTH-001",
		"name":              "Taladro QA",
		"type":              "electric",
		"status":            "active",
		"responsibleId":     "worker-x",
		"responsible":       "Manipulado",
		"assignmentDate":    "2026-03-10",
		"dateMaintenance":   "",
		"nextMaintenance":   "",
		"purchaseDate":      "",
		"price":             0,
		"location":          "Deposito Central",
		"notes":             "Intento invalido",
		"deterioration":     false,
		"assignmentHistory": []any{},
		"maintenanceRecord": []any{},
	}, accessToken, true)
	assertStatus(t, restrictedToolUpdate, http.StatusBadRequest)
	assertBodyContains(t, restrictedToolUpdate, "No puedes modificar responsable o fecha de asignacion")

	getTool := api.request(t, server, "GET", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, getTool, http.StatusOK)
	assertString(t, getTool.Body["id"], toolID)
	assertString(t, getTool.Body["name"], "Taladro QA")

	missingToolID := bson.NewObjectID().Hex()
	missingTool := api.request(t, server, "GET", "/herramientas/"+missingToolID, nil, accessToken, true)
	assertStatus(t, missingTool, http.StatusNotFound)

	toolMethodNotAllowed := api.request(t, server, "PATCH", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, toolMethodNotAllowed, http.StatusMethodNotAllowed)

	deletedTool := api.request(t, server, "DELETE", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, deletedTool, http.StatusNoContent)

	getDeletedTool := api.request(t, server, "GET", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, getDeletedTool, http.StatusNotFound)
}
