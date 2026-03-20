package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	client *mongo.Client
}

func Connect(ctx context.Context, uri string) (*MongoClient, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return &MongoClient{client: client}, nil
}

func (m *MongoClient) Disconnect(ctx context.Context) error {
	if m == nil || m.client == nil {
		return nil
	}

	return m.client.Disconnect(ctx)
}

func (m *MongoClient) Client() *mongo.Client {
	return m.client
}
