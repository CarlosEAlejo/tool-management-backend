package main

import (
	"log"
	"net/http"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/routes"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database.ConnectDB()
	defer database.DisconnectDB()

	r := routes.SetupRouter()

	// Configura CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Cambia esto según sea necesario
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	})

	// Envuelve el enrutador con el middleware CORS
	handler := c.Handler(r)

	log.Println("Servidor iniciado en el puerto 8000")
	log.Fatal(http.ListenAndServe(":8000", handler))
}
