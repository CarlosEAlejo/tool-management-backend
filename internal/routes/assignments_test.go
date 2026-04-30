package routes_test

import (
	"net/http"
	"testing"
)

func TestAssignmentsFlowAndFilters(t *testing.T) {
	server := newTestServer(t)
	api := newAPIClient(t)
	accessToken := registerAndLoginAdministrator(t, server, api)

	createTool := createToolViaAPI(t, server, api, accessToken, map[string]any{
		"name": "Taladro QA",
	})
	assertStatus(t, createTool, http.StatusCreated)
	toolID := createTool.mustString(t, "id")

	firstWorker := createWorkerViaAPI(t, server, api, accessToken, map[string]any{
		"lastName": "Supervisor",
		"position": "Jefe de brigada",
		"email":    "carlos.supervisor@example.com",
	})
	assertStatus(t, firstWorker, http.StatusCreated)
	firstWorkerID := firstWorker.mustString(t, "id")

	secondWorker := createWorkerViaAPI(t, server, api, accessToken, map[string]any{
		"firstName": "Maria",
		"lastName":  "Lopez",
		"position":  "Operaria",
		"email":     "maria.lopez@example.com",
		"phone":     "5556789",
	})
	assertStatus(t, secondWorker, http.StatusCreated)
	secondWorkerID := secondWorker.mustString(t, "id")

	assignedTool := api.request(t, server, "POST", "/asignaciones", map[string]any{
		"toolId":         toolID,
		"workerId":       firstWorkerID,
		"assignmentDate": "2026-03-21",
	}, accessToken, true)
	assertStatus(t, assignedTool, http.StatusCreated)

	getAssignedTool := api.request(t, server, "GET", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, getAssignedTool, http.StatusOK)
	assertString(t, getAssignedTool.Body["responsible"], "Carlos Supervisor")
	assignmentHistory := getAssignedTool.Body["assignmentHistory"].([]any)
	if len(assignmentHistory) != 1 {
		t.Fatalf("expected assignment history to contain 1 record, got %d", len(assignmentHistory))
	}

	reassignTool := api.request(t, server, "POST", "/asignaciones", map[string]any{
		"toolId":         toolID,
		"workerId":       secondWorkerID,
		"assignmentDate": "2026-03-22",
	}, accessToken, true)
	assertStatus(t, reassignTool, http.StatusCreated)

	assignmentEvents := api.request(t, server, "GET", "/asignaciones?toolId="+toolID+"&search=maria", nil, accessToken, true)
	assertStatus(t, assignmentEvents, http.StatusOK)
	assignmentEventList := assignmentEvents.bodyArray(t)
	if len(assignmentEventList) != 1 {
		t.Fatalf("expected filtered assignment history to return 1 event, got %d", len(assignmentEventList))
	}
	assertString(t, assignmentEventList[0]["workerName"], "Maria Lopez")

	invalidAssignmentFilter := api.request(t, server, "GET", "/asignaciones?from=2026-99-99", nil, accessToken, true)
	assertStatus(t, invalidAssignmentFilter, http.StatusBadRequest)
	assertCode(t, invalidAssignmentFilter, "validation_error")

	getReassignedTool := api.request(t, server, "GET", "/herramientas/"+toolID, nil, accessToken, true)
	assertStatus(t, getReassignedTool, http.StatusOK)
	assertString(t, getReassignedTool.Body["responsible"], "Maria Lopez")
	reassignedHistory := getReassignedTool.Body["assignmentHistory"].([]any)
	if len(reassignedHistory) != 2 {
		t.Fatalf("expected assignment history to contain 2 records after reassignment, got %d", len(reassignedHistory))
	}

	returnAssignedTool := api.request(t, server, "POST", "/asignaciones/"+toolID+"/devolver", map[string]any{}, accessToken, true)
	assertStatus(t, returnAssignedTool, http.StatusOK)

	returnAlreadyActiveTool := api.request(t, server, "POST", "/asignaciones/"+toolID+"/devolver", map[string]any{}, accessToken, true)
	assertStatus(t, returnAlreadyActiveTool, http.StatusBadRequest)
	assertCode(t, returnAlreadyActiveTool, "invalid_tool_status")
}
