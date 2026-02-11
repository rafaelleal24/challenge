package repository_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testDB *mongo.Database
var testClient *mongo.Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:7", mongodb.WithReplicaSet("rs0"))
	if err != nil {
		log.Fatalf("failed to start mongodb container: %v", err)
	}

	endpoint, err := container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	clientOpts := options.Client().
		ApplyURI(endpoint).
		SetDirect(true).
		SetConnectTimeout(30 * time.Second).
		SetServerSelectionTimeout(30 * time.Second)

	testClient, err = mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("failed to connect to mongodb: %v", err)
	}

	if err := testClient.Ping(ctx, nil); err != nil {
		log.Fatalf("failed to ping mongodb: %v", err)
	}

	testDB = testClient.Database("test_db")

	code := m.Run()

	_ = testClient.Disconnect(ctx)
	_ = container.Terminate(ctx)

	os.Exit(code)
}
