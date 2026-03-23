package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func CreateHerramienta(w http.ResponseWriter, r *http.Request) {
	var herramienta models.Herramienta
	if err := json.NewDecoder(r.Body).Decode(&herramienta); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	herramienta.ID = bson.NewObjectID()

	collection := database.GetDatabase().Collection("herramientas")
	_, err := collection.InsertOne(context.TODO(), herramienta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(herramienta)
}

func GetHerramientas(w http.ResponseWriter, r *http.Request) {
	filter := buildHerramientaFilter(r.URL.Query())

	collection := database.GetDatabase().Collection("herramientas")
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		sendErrorResponse(w, "Error al buscar herramientas: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var herramientas []models.Herramienta
	if err := cursor.All(context.TODO(), &herramientas); err != nil {
		sendErrorResponse(w, "Error al decodificar resultados: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if herramientas == nil {
		herramientas = []models.Herramienta{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(herramientas)
}

func buildHerramientaFilter(queryParams url.Values) bson.M {
	filter := bson.M{}

	exactFilters := []string{"code", "name", "type", "status", "responsible", "location"}
	for _, field := range exactFilters {
		if value := queryParams.Get(field); value != "" {
			filter[field] = value
		}
	}

	if search := queryParams.Get("search"); search != "" {
		orFilters := []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"code": bson.M{"$regex": search, "$options": "i"}},
			{"responsible": bson.M{"$regex": search, "$options": "i"}},
		}

		if len(filter) > 0 {
			filter = bson.M{
				"$and": []bson.M{
					filter,
					{"$or": orFilters},
				},
			}
		} else {
			filter["$or"] = orFilters
		}
	}

	return filter
}

func UpdateHerramienta(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	var herramienta models.Herramienta
	if err := json.NewDecoder(r.Body).Decode(&herramienta); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentHerramienta, err := getCurrentHerramienta(id)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if herramienta.Status == "assigned" {
		herramienta = processAssignmentHistory(currentHerramienta, herramienta)
	}

	if herramienta.Status == "maintenance" {
		herramienta = processMaintenanceHistory(currentHerramienta, herramienta)
	}

	if err := updateHerramientaInDB(id, herramienta); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(herramienta)
}

func getCurrentHerramienta(id bson.ObjectID) (models.Herramienta, error) {
	collection := database.GetDatabase().Collection("herramientas")
	var herramienta models.Herramienta
	err := collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&herramienta)
	return herramienta, err
}

func processAssignmentHistory(current, newHerramienta models.Herramienta) models.Herramienta {
	newHistory := models.AssignmentHistory{
		Responsible:    newHerramienta.Responsible,
		AssignmentDate: newHerramienta.AssignmentDate,
	}

	if shouldAddNewHistory(current.AssignmentHistory, newHistory) {
		newHerramienta.AssignmentHistory = append(current.AssignmentHistory, newHistory)
		if len(newHerramienta.AssignmentHistory) > 10 {
			newHerramienta.AssignmentHistory = newHerramienta.AssignmentHistory[len(newHerramienta.AssignmentHistory)-10:]
		}
	} else {
		newHerramienta.AssignmentHistory = current.AssignmentHistory
	}

	return newHerramienta
}

func shouldAddNewHistory(history []models.AssignmentHistory, newHistory models.AssignmentHistory) bool {
	if len(history) == 0 {
		return true
	}

	lastHistory := history[len(history)-1]
	return !(lastHistory.Responsible == newHistory.Responsible &&
		lastHistory.AssignmentDate == newHistory.AssignmentDate)
}

func processMaintenanceHistory(current, newHerramienta models.Herramienta) models.Herramienta {
	newMaintenanceRecord := models.MaintenanceHistory{
		DateMaintenance: newHerramienta.DateMaintenance,
		NextMaintenance: newHerramienta.NextMaintenance,
	}
	if shouldAddNewMaintenanceRecord(current.MaintenanceRecord, newMaintenanceRecord) {
		newHerramienta.MaintenanceRecord = append(current.MaintenanceRecord, newMaintenanceRecord)
		if len(newHerramienta.MaintenanceRecord) > 10 {
			newHerramienta.MaintenanceRecord = newHerramienta.MaintenanceRecord[len(newHerramienta.MaintenanceRecord)-10:]
		}
	} else {
		newHerramienta.MaintenanceRecord = current.MaintenanceRecord
	}
	return newHerramienta
}

func shouldAddNewMaintenanceRecord(records []models.MaintenanceHistory, newRecord models.MaintenanceHistory) bool {
	if len(records) == 0 {
		return true
	}
	lastRecord := records[len(records)-1]
	return !(lastRecord.DateMaintenance == newRecord.DateMaintenance &&
		lastRecord.NextMaintenance == newRecord.NextMaintenance)
}

func updateHerramientaInDB(id bson.ObjectID, herramienta models.Herramienta) error {
	collection := database.GetDatabase().Collection("herramientas")
	_, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": herramienta},
	)
	return err
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	http.Error(w, message, statusCode)
}

func DeleteHerramienta(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := database.GetDatabase().Collection("herramientas")
	result, err := collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Herramienta no encontrada", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func GetHerramientaByID(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := database.GetDatabase().Collection("herramientas")
	var herramienta models.Herramienta
	err = collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&herramienta)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Herramienta no encontrada", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(herramienta)
}
