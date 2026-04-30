package routes_test

import (
	"net/http"
	"testing"
)

func TestMaintenanceFlowAndValidation(t *testing.T) {
	server := newTestServer(t)
	api := newAPIClient(t)
	accessToken := registerAndLoginAdministrator(t, server, api)

	createTool := createToolViaAPI(t, server, api, accessToken, map[string]any{
		"name": "Taladro QA",
	})
	assertStatus(t, createTool, http.StatusCreated)
	toolID := createTool.mustString(t, "id")

	createWorker := createWorkerViaAPI(t, server, api, accessToken, map[string]any{
		"lastName": "Supervisor",
		"position": "Jefe de brigada",
	})
	assertStatus(t, createWorker, http.StatusCreated)
	workerID := createWorker.mustString(t, "id")

	assignedTool := api.request(t, server, "POST", "/asignaciones", map[string]any{
		"toolId":         toolID,
		"workerId":       workerID,
		"assignmentDate": "2026-03-21",
	}, accessToken, true)
	assertStatus(t, assignedTool, http.StatusCreated)

	maintenanceWhileAssigned := api.request(t, server, "POST", "/mantenimientos", map[string]any{
		"toolId":          toolID,
		"dateMaintenance": "2026-03-22",
		"nextMaintenance": "2026-04-22",
	}, accessToken, true)
	assertStatus(t, maintenanceWhileAssigned, http.StatusBadRequest)
	assertCode(t, maintenanceWhileAssigned, "invalid_tool_status")

	returnAssignedTool := api.request(t, server, "POST", "/asignaciones/"+toolID+"/devolver", map[string]any{}, accessToken, true)
	assertStatus(t, returnAssignedTool, http.StatusOK)

	invalidMaintenanceRange := api.request(t, server, "POST", "/mantenimientos", map[string]any{
		"toolId":          toolID,
		"dateMaintenance": "2026-04-10",
		"nextMaintenance": "2026-04-01",
	}, accessToken, true)
	assertStatus(t, invalidMaintenanceRange, http.StatusBadRequest)
	assertCode(t, invalidMaintenanceRange, "validation_error")

	createMaintenance := api.request(t, server, "POST", "/mantenimientos", map[string]any{
		"toolId":          toolID,
		"dateMaintenance": "2026-04-10",
		"nextMaintenance": "2026-05-10",
	}, accessToken, true)
	assertStatus(t, createMaintenance, http.StatusCreated)

	maintenanceEvents := api.request(t, server, "GET", "/mantenimientos?toolId="+toolID+"&search=scheduled", nil, accessToken, true)
	assertStatus(t, maintenanceEvents, http.StatusOK)
	maintenanceEventList := maintenanceEvents.bodyArray(t)
	if len(maintenanceEventList) != 1 {
		t.Fatalf("expected filtered maintenance history to return 1 event, got %d", len(maintenanceEventList))
	}

	invalidMaintenanceFilter := api.request(t, server, "GET", "/mantenimientos?to=not-a-date", nil, accessToken, true)
	assertStatus(t, invalidMaintenanceFilter, http.StatusBadRequest)
	assertCode(t, invalidMaintenanceFilter, "validation_error")

	updateToolWhileInMaintenance := api.request(t, server, "PUT", "/herramientas/"+toolID, map[string]any{
		"code":              "AUTH-001",
		"name":              "Taladro QA",
		"type":              "electric",
		"status":            "maintenance",
		"responsibleId":     "",
		"responsible":       "",
		"assignmentDate":    "",
		"dateMaintenance":   "2026-04-10",
		"nextMaintenance":   "2026-05-10",
		"purchaseDate":      "",
		"price":             0,
		"location":          "Patio temporal",
		"notes":             "Actualizado en mantenimiento",
		"deterioration":     false,
		"assignmentHistory": []any{},
		"maintenanceRecord": []any{},
	}, accessToken, true)
	assertStatus(t, updateToolWhileInMaintenance, http.StatusOK)
	assertString(t, updateToolWhileInMaintenance.Body["location"], "Patio temporal")

	completeMaintenance := api.request(t, server, "POST", "/mantenimientos/"+toolID+"/finalizar", map[string]any{}, accessToken, true)
	assertStatus(t, completeMaintenance, http.StatusOK)

	completeMaintenanceAgain := api.request(t, server, "POST", "/mantenimientos/"+toolID+"/finalizar", map[string]any{}, accessToken, true)
	assertStatus(t, completeMaintenanceAgain, http.StatusBadRequest)
	assertCode(t, completeMaintenanceAgain, "invalid_tool_status")

	allMaintenanceEvents := api.request(t, server, "GET", "/mantenimientos?toolId="+toolID, nil, accessToken, true)
	assertStatus(t, allMaintenanceEvents, http.StatusOK)
	if len(allMaintenanceEvents.bodyArray(t)) != 2 {
		t.Fatalf("expected maintenance history to contain 2 events, got %d", len(allMaintenanceEvents.bodyArray(t)))
	}
}
