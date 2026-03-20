package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"tool_management_backend/internal/http/handlers"
	"tool_management_backend/internal/http/router"
	"tool_management_backend/internal/platform/config"
	"tool_management_backend/internal/platform/database"
	mongorepo "tool_management_backend/internal/repository/mongo"
	toolservice "tool_management_backend/internal/service/tool"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		if err := db.Disconnect(disconnectCtx); err != nil {
			log.Printf("mongo disconnect error: %v", err)
		}
	}()

	collection := db.Client().Database(cfg.DatabaseName).Collection(cfg.CollectionName)
	repository := mongorepo.NewToolRepository(collection)
	service := toolservice.New(repository)
	toolHandler := handlers.NewToolHandler(service)
	r := router.New(toolHandler)

	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.FrontendOrigins,
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	})

	handler := c.Handler(r)
	listener, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Servidor listo en el puerto %s", cfg.Port)
	log.Fatal(http.Serve(listener, handler))
}
