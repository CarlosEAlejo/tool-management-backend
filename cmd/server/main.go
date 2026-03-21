package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/routes"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsedOrigin.Scheme != "http" {
		return false
	}

	hostname := parsedOrigin.Hostname()
	if hostname != "localhost" && hostname != "127.0.0.1" {
		return false
	}

	port := parsedOrigin.Port()
	if port == "" {
		return false
	}

	return strings.TrimSpace(port) != ""
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database.ConnectDB()
	defer database.DisconnectDB()

	r := routes.SetupRouter()

	c := cors.New(cors.Options{
		AllowOriginFunc: isAllowedOrigin,
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	})

	handler := c.Handler(r)

	log.Println("Servidor iniciado en el puerto 8000")
	log.Fatal(http.ListenAndServe(":8000", handler))
}
