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

type maintenancePayload struct {
	ToolID          string `json:"toolId"`
	DateMaintenance string `json:"dateMaintenance"`
	NextMaintenance string `json:"nextMaintenance"`
}

func CreateMaintenance(w http.ResponseWriter, r *http.Request) {
	var payload maintenancePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	payload.ToolID = strings.TrimSpace(payload.ToolID)
	payload.DateMaintenance = strings.TrimSpace(payload.DateMaintenance)
	payload.NextMaintenance = strings.TrimSpace(payload.NextMaintenance)

	if payload.ToolID == "" || payload.DateMaintenance == "" || payload.NextMaintenance == "" {
		WriteJSONError(w, "validation_error", "La herramienta y las fechas de mantenimiento son obligatorias.", http.StatusBadRequest)
		return
	}

	dateMaintenance, err := time.Parse("2006-01-02", payload.DateMaintenance)
	if err != nil {
		WriteJSONError(w, "validation_error", "La fecha de mantenimiento debe tener formato YYYY-MM-DD.", http.StatusBadRequest)
		return
	}
	nextMaintenance, err := time.Parse("2006-01-02", payload.NextMaintenance)
	if err != nil {
		WriteJSONError(w, "validation_error", "La fecha proxima de mantenimiento debe tener formato YYYY-MM-DD.", http.StatusBadRequest)
		return
	}
	if nextMaintenance.Before(dateMaintenance) {
		WriteJSONError(w, "validation_error", "La fecha proxima de mantenimiento no puede ser anterior a la fecha de mantenimiento.", http.StatusBadRequest)
		return
	}

	toolObjectID, err := bson.ObjectIDFromHex(payload.ToolID)
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

	if tool.Status != "active" && tool.Status != "maintenance" {
		WriteJSONError(w, "invalid_tool_status", "Solo se pueden programar mantenimientos para herramientas disponibles o ya en mantenimiento.", http.StatusBadRequest)
		return
	}

	updatedTool := tool
	updatedTool.Status = "maintenance"
	updatedTool.DateMaintenance = payload.DateMaintenance
	updatedTool.NextMaintenance = payload.NextMaintenance
	updatedTool = processMaintenanceHistory(tool, updatedTool)

	if err := updateHerramientaInDB(toolObjectID, updatedTool); err != nil {
		WriteJSONError(w, "tool_update_failed", "No se pudo actualizar la herramienta.", http.StatusInternalServerError)
		return
	}

	event := models.MaintenanceEvent{
		ID:              bson.NewObjectID(),
		ToolID:          tool.ID.Hex(),
		ToolCode:        tool.Code,
		ToolName:        tool.Name,
		Action:          models.MaintenanceActionScheduled,
		DateMaintenance: payload.DateMaintenance,
		NextMaintenance: payload.NextMaintenance,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	if err := insertMaintenanceEvent(event); err != nil {
		WriteJSONError(w, "maintenance_log_failed", "No se pudo registrar el evento de mantenimiento.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{
		"tool":  updatedTool,
		"event": event,
	})
}

func CompleteMaintenance(w http.ResponseWriter, r *http.Request) {
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

	if tool.Status != "maintenance" {
		WriteJSONError(w, "invalid_tool_status", "La herramienta no se encuentra en mantenimiento.", http.StatusBadRequest)
		return
	}

	updatedTool := tool
	updatedTool.Status = "active"
	updatedTool.DateMaintenance = ""
	updatedTool.NextMaintenance = ""

	if err := updateHerramientaInDB(toolObjectID, updatedTool); err != nil {
		WriteJSONError(w, "tool_update_failed", "No se pudo actualizar la herramienta.", http.StatusInternalServerError)
		return
	}

	event := models.MaintenanceEvent{
		ID:              bson.NewObjectID(),
		ToolID:          tool.ID.Hex(),
		ToolCode:        tool.Code,
		ToolName:        tool.Name,
		Action:          models.MaintenanceActionCompleted,
		DateMaintenance: tool.DateMaintenance,
		NextMaintenance: tool.NextMaintenance,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	if err := insertMaintenanceEvent(event); err != nil {
		WriteJSONError(w, "maintenance_log_failed", "No se pudo registrar el cierre del mantenimiento.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"tool":  updatedTool,
		"event": event,
	})
}

func ListMaintenanceEvents(w http.ResponseWriter, r *http.Request) {
	filter, err := buildMaintenanceEventFilter(r.URL.Query())
	if err != nil {
		WriteJSONError(w, "validation_error", err.Error(), http.StatusBadRequest)
		return
	}

	cursor, err := maintenanceEventsCollection().Find(context.TODO(), filter)
	if err != nil {
		WriteJSONError(w, "maintenance_list_failed", "No se pudo obtener el historial de mantenimientos.", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	events := []models.MaintenanceEvent{}
	if err := cursor.All(context.TODO(), &events); err != nil {
		WriteJSONError(w, "maintenance_list_failed", "No se pudo obtener el historial de mantenimientos.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

func buildMaintenanceEventFilter(query url.Values) (bson.M, error) {
	filter := bson.M{}
	andFilters := make([]bson.M, 0)

	if toolID := strings.TrimSpace(query.Get("toolId")); toolID != "" {
		andFilters = append(andFilters, bson.M{"toolId": toolID})
	}

	if search := strings.TrimSpace(query.Get("search")); search != "" {
		andFilters = append(andFilters, bson.M{
			"$or": []bson.M{
				{"toolCode": bson.M{"$regex": search, "$options": "i"}},
				{"toolName": bson.M{"$regex": search, "$options": "i"}},
				{"action": bson.M{"$regex": search, "$options": "i"}},
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

func insertMaintenanceEvent(event models.MaintenanceEvent) error {
	_, err := maintenanceEventsCollection().InsertOne(context.TODO(), event)
	return err
}

func maintenanceEventsCollection() *mongo.Collection {
	return database.GetDatabase().Collection("mantenimientos")
}
