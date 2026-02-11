package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/rafaelleal24/challenge/internal/adapters/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewConnection(config config.MongoConfig) (*mongo.Client, error) {

	clientOpts := options.Client().
		ApplyURI(config.URI).
		SetTimeout(config.Timeout).
		SetConnectTimeout(config.ConnectTimeout).
		SetServerSelectionTimeout(config.ServerSelectionTimeout).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize)

	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

func Disconnect(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if client == nil {
		return nil
	}

	if err := client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	return nil
}
