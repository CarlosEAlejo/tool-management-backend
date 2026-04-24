package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func CreateTrabajador(w http.ResponseWriter, r *http.Request) {
	var trabajador models.Trabajador
	if err := json.NewDecoder(r.Body).Decode(&trabajador); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	trabajador = normalizeTrabajadorPayload(trabajador)
	if err := validateTrabajador(trabajador); err != nil {
		WriteJSONError(w, "validation_error", err.Error(), http.StatusBadRequest)
		return
	}

	trabajador.ID = bson.NewObjectID()
	if _, err := trabajadoresCollection().InsertOne(context.TODO(), trabajador); err != nil {
		WriteJSONError(w, "worker_create_failed", "No se pudo crear el trabajador.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusCreated, trabajador)
}

func GetTrabajadores(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("search"))
	filter := bson.M{}
	if query != "" {
		filter["$or"] = []bson.M{
			{"firstName": bson.M{"$regex": query, "$options": "i"}},
			{"lastName": bson.M{"$regex": query, "$options": "i"}},
			{"position": bson.M{"$regex": query, "$options": "i"}},
			{"email": bson.M{"$regex": query, "$options": "i"}},
		}
	}

	cursor, err := trabajadoresCollection().Find(context.TODO(), filter)
	if err != nil {
		WriteJSONError(w, "worker_list_failed", "No se pudo obtener la lista de trabajadores.", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	trabajadores := []models.Trabajador{}
	if err := cursor.All(context.TODO(), &trabajadores); err != nil {
		WriteJSONError(w, "worker_list_failed", "No se pudo obtener la lista de trabajadores.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, trabajadores)
}

func GetTrabajadorByID(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		WriteJSONError(w, "invalid_id", "El identificador no es valido.", http.StatusBadRequest)
		return
	}

	var trabajador models.Trabajador
	if err := trabajadoresCollection().FindOne(context.TODO(), bson.M{"_id": id}).Decode(&trabajador); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			WriteJSONError(w, "worker_not_found", "El trabajador no existe.", http.StatusNotFound)
			return
		}
		WriteJSONError(w, "worker_read_failed", "No se pudo obtener el trabajador.", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, trabajador)
}

func UpdateTrabajador(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		WriteJSONError(w, "invalid_id", "El identificador no es valido.", http.StatusBadRequest)
		return
	}

	var trabajador models.Trabajador
	if err := json.NewDecoder(r.Body).Decode(&trabajador); err != nil {
		WriteJSONError(w, "invalid_json", "Los datos enviados no tienen un formato valido.", http.StatusBadRequest)
		return
	}

	trabajador = normalizeTrabajadorPayload(trabajador)
	if err := validateTrabajador(trabajador); err != nil {
		WriteJSONError(w, "validation_error", err.Error(), http.StatusBadRequest)
		return
	}

	result, err := trabajadoresCollection().UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"firstName": trabajador.FirstName,
			"lastName":  trabajador.LastName,
			"position":  trabajador.Position,
			"email":     trabajador.Email,
			"phone":     trabajador.Phone,
			"notes":     trabajador.Notes,
		}},
	)
	if err != nil {
		WriteJSONError(w, "worker_update_failed", "No se pudo actualizar el trabajador.", http.StatusInternalServerError)
		return
	}
	if result.MatchedCount == 0 {
		WriteJSONError(w, "worker_not_found", "El trabajador no existe.", http.StatusNotFound)
		return
	}

	trabajador.ID = id
	WriteJSON(w, http.StatusOK, trabajador)
}

func DeleteTrabajador(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		WriteJSONError(w, "invalid_id", "El identificador no es valido.", http.StatusBadRequest)
		return
	}

	result, err := trabajadoresCollection().DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		WriteJSONError(w, "worker_delete_failed", "No se pudo eliminar el trabajador.", http.StatusInternalServerError)
		return
	}
	if result.DeletedCount == 0 {
		WriteJSONError(w, "worker_not_found", "El trabajador no existe.", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func trabajadoresCollection() *mongo.Collection {
	return database.GetDatabase().Collection("trabajadores")
}

func normalizeTrabajadorPayload(trabajador models.Trabajador) models.Trabajador {
	trabajador.FirstName = strings.TrimSpace(trabajador.FirstName)
	trabajador.LastName = strings.TrimSpace(trabajador.LastName)
	trabajador.Position = strings.TrimSpace(trabajador.Position)
	trabajador.Email = strings.TrimSpace(strings.ToLower(trabajador.Email))
	trabajador.Phone = strings.TrimSpace(trabajador.Phone)
	trabajador.Notes = strings.TrimSpace(trabajador.Notes)
	return trabajador
}

func validateTrabajador(trabajador models.Trabajador) error {
	if trabajador.FirstName == "" {
		return errors.New("El nombre es obligatorio")
	}
	if trabajador.LastName == "" {
		return errors.New("Los apellidos son obligatorios")
	}
	if trabajador.Position == "" {
		return errors.New("El cargo es obligatorio")
	}
	return nil
}
