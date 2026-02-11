package redis_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/rafaelleal24/challenge/internal/adapters/config"
	adaptredis "github.com/rafaelleal24/challenge/internal/adapters/redis"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var testClient *adaptredis.Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		log.Fatalf("failed to start redis container: %v", err)
	}

	endpoint, err := container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	testClient, err = adaptredis.NewConnection(config.RedisConfig{
		URL:      endpoint,
		Password: "",
		DB:       0,
	})
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	code := m.Run()

	_ = testClient.Close()
	_ = container.Terminate(ctx)

	os.Exit(code)
}
