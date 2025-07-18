package main

import (
	"log"
	"net/http"

	// "os"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/routes"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database.ConnectDB()
	defer database.DisconnectDB()

	r := routes.SetupRouter()

	log.Println("Servidor iniciado en el puerto 8000")
	log.Fatal(http.ListenAndServe(":8000", r))
}
