package database

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var client *mongo.Client

func ConnectDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err = mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}
	log.Println("Conectado a MongoDB!")
}

func DisconnectDB() {
	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Disconnect(ctx); err != nil {
		log.Fatal(err)
	}
}

func GetClient() *mongo.Client {
	return client
}

func GetDatabaseName() string {
	name := strings.TrimSpace(os.Getenv("MONGODB_DATABASE"))
	if name == "" {
		return "herramientas"
	}
	return name
}

func GetDatabase() *mongo.Database {
	return client.Database(GetDatabaseName())
}
