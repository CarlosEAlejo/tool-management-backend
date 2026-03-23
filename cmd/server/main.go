package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
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

func serverAddress() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8000"
	}

	if strings.HasPrefix(port, ":") {
		return port
	}

	return ":" + port
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not loaded; using system environment")
	}

	database.ConnectDB()
	defer database.DisconnectDB()

	r := routes.SetupRouter()

	c := cors.New(cors.Options{
		AllowOriginFunc:  isAllowedOrigin,
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-Token"},
	})

	addr := serverAddress()
	handler := c.Handler(r)

	log.Printf("Servidor iniciado en %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
