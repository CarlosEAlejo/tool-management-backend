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

func UpdateHerramienta(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var herramienta models.Herramienta
	if err := json.NewDecoder(r.Body).Decode(&herramienta); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := database.GetClient().Database("herramientas").Collection("herramientas")
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": herramienta},
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(herramienta)
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
