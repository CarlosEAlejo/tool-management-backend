package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type assignmentPayload struct {
	ToolID         string `json:"toolId"`
	WorkerID       string `json:"workerId"`
	AssignmentDate string `json:"assignmentDate"`
}

func CreateAssignment(w http.ResponseWriter, r *http.Request) {
	var payload assignmentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	payload.ToolID = strings.TrimSpace(payload.ToolID)
	payload.WorkerID = strings.TrimSpace(payload.WorkerID)
	payload.AssignmentDate = normalizeAssignmentDate(payload.AssignmentDate)

	if payload.ToolID == "" || payload.WorkerID == "" {
		WriteJSONError(w, "validation_error", "La herramienta y el trabajador son obligatorios.", http.StatusBadRequest)
		return
	}

	toolObjectID, err := bson.ObjectIDFromHex(payload.ToolID)
	if err != nil {
		WriteJSONError(w, "invalid_tool_id", "El identificador de herramienta no es valido.", http.StatusBadRequest)
		return
	}

	workerObjectID, err := bson.ObjectIDFromHex(payload.WorkerID)
	if err != nil {
		WriteJSONError(w, "invalid_worker_id", "El identificador de trabajador no es valido.", http.StatusBadRequest)
		return
	}

	tool, err := getCurrentHerramienta(toolObjectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			WriteJSONError(w, "tool_not_found", "La herramienta seleccionada no existe.", http.StatusNotFound)
			return
		}
		WriteJSONError(w, "tool_read_failed", "No se pudo obtener la herramienta.", http.StatusInternalServerError)
		return
	}

	if tool.Status != "active" && tool.Status != "assigned" {
		WriteJSONError(w, "invalid_tool_status", "Solo se pueden asignar herramientas disponibles o ya asignadas.", http.StatusBadRequest)
		return
	}

	worker, err := getTrabajadorByObjectID(workerObjectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			WriteJSONError(w, "worker_not_found", "El trabajador seleccionado no existe.", http.StatusNotFound)
			return
		}
		WriteJSONError(w, "worker_read_failed", "No se pudo obtener el trabajador.", http.StatusInternalServerError)
		return
	}

	workerName := strings.TrimSpace(strings.TrimSpace(worker.FirstName) + " " + strings.TrimSpace(worker.LastName))
	updatedTool := tool
	updatedTool.Status = "assigned"
	updatedTool.ResponsibleID = worker.ID.Hex()
	updatedTool.Responsible = workerName
	updatedTool.AssignmentDate = payload.AssignmentDate
	updatedTool = processAssignmentHistory(tool, updatedTool)

	if err := updateHerramientaInDB(toolObjectID, updatedTool); err != nil {
		WriteJSONError(w, "tool_update_failed", "No se pudo actualizar la herramienta.", http.StatusInternalServerError)
		return
	}

	event := models.AssignmentEvent{
		ID:             bson.NewObjectID(),
		ToolID:         tool.ID.Hex(),
		ToolCode:       tool.Code,
		ToolName:       tool.Name,
		WorkerID:       worker.ID.Hex(),
		WorkerName:     workerName,
		Action:         models.AssignmentActionAssigned,
		AssignmentDate: payload.AssignmentDate,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	if err := insertAssignmentEvent(event); err != nil {
		WriteJSONError(w, "assignment_log_failed", "No se pudo registrar el evento de asignacion.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{
		"tool":  updatedTool,
		"event": event,
	})
}

func ReturnAssignment(w http.ResponseWriter, r *http.Request) {
	toolHexID := strings.TrimSpace(r.PathValue("toolId"))
	toolObjectID, err := bson.ObjectIDFromHex(toolHexID)
	if err != nil {
		WriteJSONError(w, "invalid_tool_id", "El identificador de herramienta no es valido.", http.StatusBadRequest)
		return
	}

	tool, err := getCurrentHerramienta(toolObjectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			WriteJSONError(w, "tool_not_found", "La herramienta seleccionada no existe.", http.StatusNotFound)
			return
		}
		WriteJSONError(w, "tool_read_failed", "No se pudo obtener la herramienta.", http.StatusInternalServerError)
		return
	}

	if tool.Status != "assigned" {
		WriteJSONError(w, "invalid_tool_status", "La herramienta no se encuentra asignada.", http.StatusBadRequest)
		return
	}

	assignmentDate := normalizeAssignmentDate("")
	event := models.AssignmentEvent{
		ID:             bson.NewObjectID(),
		ToolID:         tool.ID.Hex(),
		ToolCode:       tool.Code,
		ToolName:       tool.Name,
		WorkerID:       tool.ResponsibleID,
		WorkerName:     tool.Responsible,
		Action:         models.AssignmentActionReturned,
		AssignmentDate: assignmentDate,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	updatedTool := tool
	updatedTool.Status = "active"
	updatedTool.ResponsibleID = ""
	updatedTool.Responsible = ""
	updatedTool.AssignmentDate = ""

	if err := updateHerramientaInDB(toolObjectID, updatedTool); err != nil {
		WriteJSONError(w, "tool_update_failed", "No se pudo actualizar la herramienta.", http.StatusInternalServerError)
		return
	}

	if err := insertAssignmentEvent(event); err != nil {
		WriteJSONError(w, "assignment_log_failed", "No se pudo registrar el evento de devolucion.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"tool":  updatedTool,
		"event": event,
	})
}

func ListAssignmentEvents(w http.ResponseWriter, r *http.Request) {
	filter, err := buildAssignmentEventFilter(r.URL.Query())
	if err != nil {
		WriteJSONError(w, "validation_error", err.Error(), http.StatusBadRequest)
		return
	}

	cursor, err := assignmentEventsCollection().Find(context.TODO(), filter)
	if err != nil {
		WriteJSONError(w, "assignment_list_failed", "No se pudo obtener el historial de asignaciones.", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	events := []models.AssignmentEvent{}
	if err := cursor.All(context.TODO(), &events); err != nil {
		WriteJSONError(w, "assignment_list_failed", "No se pudo obtener el historial de asignaciones.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

func buildAssignmentEventFilter(query url.Values) (bson.M, error) {
	filter := bson.M{}
	andFilters := make([]bson.M, 0)

	if toolID := strings.TrimSpace(query.Get("toolId")); toolID != "" {
		andFilters = append(andFilters, bson.M{"toolId": toolID})
	}

	if workerID := strings.TrimSpace(query.Get("workerId")); workerID != "" {
		andFilters = append(andFilters, bson.M{"workerId": workerID})
	}

	if search := strings.TrimSpace(query.Get("search")); search != "" {
		andFilters = append(andFilters, bson.M{
			"$or": []bson.M{
				{"toolCode": bson.M{"$regex": search, "$options": "i"}},
				{"toolName": bson.M{"$regex": search, "$options": "i"}},
				{"workerName": bson.M{"$regex": search, "$options": "i"}},
			},
		})
	}

	createdAtFilter := bson.M{}
	if from := strings.TrimSpace(query.Get("from")); from != "" {
		parsed, err := time.Parse("2006-01-02", from)
		if err != nil {
			return nil, errors.New("La fecha 'from' no es valida. Usa YYYY-MM-DD")
		}
		createdAtFilter["$gte"] = parsed.UTC().Format(time.RFC3339)
	}
	if to := strings.TrimSpace(query.Get("to")); to != "" {
		parsed, err := time.Parse("2006-01-02", to)
		if err != nil {
			return nil, errors.New("La fecha 'to' no es valida. Usa YYYY-MM-DD")
		}
		createdAtFilter["$lte"] = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second).UTC().Format(time.RFC3339)
	}
	if len(createdAtFilter) > 0 {
		andFilters = append(andFilters, bson.M{"createdAt": createdAtFilter})
	}

	if len(andFilters) == 1 {
		return andFilters[0], nil
	}
	if len(andFilters) > 1 {
		filter["$and"] = andFilters
	}

	return filter, nil
}

func normalizeAssignmentDate(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return time.Now().Format("2006-01-02")
	}

	if _, err := time.Parse("2006-01-02", trimmed); err == nil {
		return trimmed
	}

	return time.Now().Format("2006-01-02")
}

func insertAssignmentEvent(event models.AssignmentEvent) error {
	_, err := assignmentEventsCollection().InsertOne(context.TODO(), event)
	return err
}

func assignmentEventsCollection() *mongo.Collection {
	return database.GetDatabase().Collection("asignaciones")
}

func getTrabajadorByObjectID(id bson.ObjectID) (models.Trabajador, error) {
	var trabajador models.Trabajador
	err := trabajadoresCollection().FindOne(context.TODO(), bson.M{"_id": id}).Decode(&trabajador)
	return trabajador, err
}
