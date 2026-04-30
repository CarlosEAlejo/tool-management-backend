package routes_test

import (
	"net/http"
	"testing"
)

func TestWorkersCRUDAndSearch(t *testing.T) {
	server := newTestServer(t)
	api := newAPIClient(t)
	accessToken := registerAndLoginAdministrator(t, server, api)

	createWorker := createWorkerViaAPI(t, server, api, accessToken, nil)
	assertStatus(t, createWorker, http.StatusCreated)
	workerID := createWorker.mustString(t, "id")

	listWorkers := api.request(t, server, "GET", "/trabajadores?search=carlos", nil, accessToken, true)
	assertStatus(t, listWorkers, http.StatusOK)
	if len(listWorkers.bodyArray(t)) != 1 {
		t.Fatalf("expected worker search to return 1 item, got %d", len(listWorkers.bodyArray(t)))
	}

	updateWorker := api.request(t, server, "PUT", "/trabajadores/"+workerID, map[string]any{
		"firstName": "Carlos",
		"lastName":  "Supervisor",
		"position":  "Jefe de brigada",
		"email":     "carlos.supervisor@example.com",
		"phone":     "5554321",
		"notes":     "Actualizado",
	}, accessToken, true)
	assertStatus(t, updateWorker, http.StatusOK)
	assertString(t, updateWorker.Body["lastName"], "Supervisor")
	assertString(t, updateWorker.Body["position"], "Jefe de brigada")

	getWorker := api.request(t, server, "GET", "/trabajadores/"+workerID, nil, accessToken, true)
	assertStatus(t, getWorker, http.StatusOK)
	assertString(t, getWorker.Body["email"], "carlos.supervisor@example.com")

	deleteWorker := api.request(t, server, "DELETE", "/trabajadores/"+workerID, nil, accessToken, true)
	assertStatus(t, deleteWorker, http.StatusNoContent)

	getDeletedWorker := api.request(t, server, "GET", "/trabajadores/"+workerID, nil, accessToken, true)
	assertStatus(t, getDeletedWorker, http.StatusNotFound)
}
