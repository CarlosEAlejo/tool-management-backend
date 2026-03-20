package database

import (
	"context"
	"log"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func ConnectDB() {
	var err error
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Conectado a MongoDB!")
}

func DisconnectDB() {
	if client == nil {
		return
	}
	if err := client.Disconnect(context.TODO()); err != nil {
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
