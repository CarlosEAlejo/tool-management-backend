package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateHerramienta(w http.ResponseWriter, r *http.Request) {
	var herramienta models.Herramienta
	if err := json.NewDecoder(r.Body).Decode(&herramienta); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	herramienta.ID = primitive.NewObjectID()

	collection := database.GetClient().Database("herramientas").Collection("herramientas")
	_, err := collection.InsertOne(context.TODO(), herramienta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(herramienta)
}

func GetHerramientas(w http.ResponseWriter, r *http.Request) {
	query := bson.M{}

	// Add filters from query parameters
	if status := r.URL.Query().Get("status"); status != "" {
		query["status"] = status
	}

	if responsible := r.URL.Query().Get("responsible"); responsible != "" {
		query["responsible"] = bson.M{"$regex": primitive.Regex{Pattern: responsible, Options: "i"}}
	}

	collection := database.GetClient().Database("herramientas").Collection("herramientas")
	cursor, err := collection.Find(context.TODO(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO()) // Asegúrate de cerrar el cursor

	var herramientas []models.Herramienta
	if err = cursor.All(context.TODO(), &herramientas); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(herramientas)
}

// UpdateHerramienta maneja la actualización de una herramienta
func UpdateHerramienta(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	var herramienta models.Herramienta
	if err := json.NewDecoder(r.Body).Decode(&herramienta); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Obtener la herramienta actual
	currentHerramienta, err := getCurrentHerramienta(id)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Procesar el historial si es necesario
	if herramienta.Status == "assigned" {
		herramienta = processAssignmentHistory(currentHerramienta, herramienta)
	}

	// Procesar historial de mantenimientos si es necesario
	if herramienta.Status == "maintenance" {
		herramienta = processMaintenanceHistory(currentHerramienta, herramienta)
	}

	// Actualizar en la base de datos
	if err := updateHerramientaInDB(id, herramienta); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(herramienta)
}

// getCurrentHerramienta obtiene la herramienta actual desde la base de datos
func getCurrentHerramienta(id primitive.ObjectID) (models.Herramienta, error) {
	collection := database.GetClient().Database("herramientas").Collection("herramientas")
	var herramienta models.Herramienta
	err := collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&herramienta)
	return herramienta, err
}

// processAssignmentHistory procesa el historial de asignaciones
func processAssignmentHistory(current, newHerramienta models.Herramienta) models.Herramienta {
	newHistory := models.AssignmentHistory{
		Responsible:    newHerramienta.Responsible,
		AssignmentDate: newHerramienta.AssignmentDate,
	}

	// Verificar si hay que agregar el nuevo historial
	if shouldAddNewHistory(current.AssignmentHistory, newHistory) {
		newHerramienta.AssignmentHistory = append(current.AssignmentHistory, newHistory)

		// Limitar el historial a los últimos 10 registros
		if len(newHerramienta.AssignmentHistory) > 10 {
			newHerramienta.AssignmentHistory = newHerramienta.AssignmentHistory[len(newHerramienta.AssignmentHistory)-10:]
		}
	} else {
		newHerramienta.AssignmentHistory = current.AssignmentHistory
	}

	return newHerramienta
}

// shouldAddNewHistory determina si se debe agregar un nuevo registro al historial
func shouldAddNewHistory(history []models.AssignmentHistory, newHistory models.AssignmentHistory) bool {
	// Si no hay historial, agregar el primer registro
	if len(history) == 0 {
		return true
	}

	lastHistory := history[len(history)-1]
	// Solo agregar si es diferente al último registro
	return !(lastHistory.Responsible == newHistory.Responsible &&
		lastHistory.AssignmentDate == newHistory.AssignmentDate)
}

// Nuevo método para procesar historial de mantenimientos
func processMaintenanceHistory(current, newHerramienta models.Herramienta) models.Herramienta {
	newMaintenanceRecord := models.MaintenanceHistory{
		DateMaintenance: newHerramienta.DateMaintenance,
		NextMaintenance: newHerramienta.NextMaintenance,
	}
	if shouldAddNewMaintenanceRecord(current.MaintenanceRecord, newMaintenanceRecord) {
		newHerramienta.MaintenanceRecord = append(current.MaintenanceRecord, newMaintenanceRecord)

		// Limitar a los últimos 10 registros
		if len(newHerramienta.MaintenanceRecord) > 10 {
			newHerramienta.MaintenanceRecord = newHerramienta.MaintenanceRecord[len(newHerramienta.MaintenanceRecord)-10:]
		}
	} else {
		newHerramienta.MaintenanceRecord = current.MaintenanceRecord
	}
	return newHerramienta
}

// Determina si se debe añadir nuevo registro de mantenimiento
func shouldAddNewMaintenanceRecord(records []models.MaintenanceHistory, newRecord models.MaintenanceHistory) bool {
	if len(records) == 0 {
		return true
	}
	lastRecord := records[len(records)-1]
	return !(lastRecord.DateMaintenance == newRecord.DateMaintenance &&
		lastRecord.NextMaintenance == newRecord.NextMaintenance)
}

// updateHerramientaInDB actualiza la herramienta en la base de datos
func updateHerramientaInDB(id primitive.ObjectID, herramienta models.Herramienta) error {
	collection := database.GetClient().Database("herramientas").Collection("herramientas")
	_, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": herramienta},
	)
	return err
}

// sendErrorResponse envía una respuesta de error
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	http.Error(w, message, statusCode)
}

func DeleteHerramienta(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := database.GetClient().Database("herramientas").Collection("herramientas")
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
	params := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := database.GetClient().Database("herramientas").Collection("herramientas")
	var herramienta models.Herramienta
	err = collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&herramienta)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Herramienta no encontrada", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(herramienta)
}
