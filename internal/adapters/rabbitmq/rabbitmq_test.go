package rabbitmq_test

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rafaelleal24/challenge/internal/adapters/config"
	"github.com/rafaelleal24/challenge/internal/adapters/rabbitmq"
	tcrabbit "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

var (
	testAdapter      *rabbitmq.RabbitMQAdapter
	testAmqpEndpoint string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcrabbit.Run(ctx, "rabbitmq:3-management-alpine")
	if err != nil {
		log.Fatalf("failed to start rabbitmq container: %v", err)
	}

	testAmqpEndpoint, err = container.AmqpURL(ctx)
	if err != nil {
		log.Fatalf("failed to get amqp url: %v", err)
	}

	testAdapter, err = rabbitmq.NewRabbitMQAdapter(config.RabbitMQConfig{
		URL:        testAmqpEndpoint,
		MaxRetries: 2,
		RetryDelay: 100 * time.Millisecond,
		ExchangeConfigs: []config.ExchangeConfig{
			{
				Name:       "exchange.order",
				Type:       "direct",
				Durable:    true,
				AutoDelete: false,
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create rabbitmq adapter: %v", err)
	}

	code := m.Run()

	_ = testAdapter.Close()
	_ = container.Terminate(ctx)

	os.Exit(code)
}

func TestRabbitMQAdapter_HealthCheck(t *testing.T) {
	t.Run("healthy after connection", func(t *testing.T) {
		err := testAdapter.HealthCheck()
		if err != nil {
			t.Fatalf("expected healthy, got %v", err)
		}
	})
}

func TestRabbitMQAdapter_PublishRaw(t *testing.T) {
	ctx := context.Background()

	t.Run("publishes raw message successfully", func(t *testing.T) {
		data := []byte(`{"order_id":"abc123","status":"created"}`)
		err := testAdapter.PublishRaw(ctx, "order.created", "order", data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("published message can be consumed", func(t *testing.T) {
		// Connect a consumer to verify the message actually arrives
		conn, err := amqp.Dial(testAmqpEndpoint)
		if err != nil {
			t.Fatalf("consumer dial failed: %v", err)
		}
		defer conn.Close()

		ch, err := conn.Channel()
		if err != nil {
			t.Fatalf("consumer channel failed: %v", err)
		}
		defer ch.Close()

		q, err := ch.QueueDeclare("test-queue", false, true, false, false, nil)
		if err != nil {
			t.Fatalf("queue declare failed: %v", err)
		}

		err = ch.QueueBind(q.Name, "order.test_consume", "exchange.order", false, nil)
		if err != nil {
			t.Fatalf("queue bind failed: %v", err)
		}

		msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
		if err != nil {
			t.Fatalf("consume failed: %v", err)
		}

		// Publish
		payload := map[string]string{"test": "hello"}
		body, _ := json.Marshal(payload)
		err = testAdapter.PublishRaw(ctx, "order.test_consume", "order", body)
		if err != nil {
			t.Fatalf("publish failed: %v", err)
		}

		// Wait for message
		select {
		case msg := <-msgs:
			var received map[string]string
			if err := json.Unmarshal(msg.Body, &received); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if received["test"] != "hello" {
				t.Fatalf("expected 'hello', got %q", received["test"])
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for message")
		}
	})
}

func TestRabbitMQAdapter_Publish(t *testing.T) {
	ctx := context.Background()

	t.Run("publishes domain event", func(t *testing.T) {
		event := &mockEvent{
			name:       "order.created",
			entityName: "order",
			data:       map[string]string{"order_id": "test123"},
		}

		err := testAdapter.Publish(ctx, event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestRabbitMQAdapter_CloseAndReconnect(t *testing.T) {
	ctx := context.Background()

	t.Run("reconnects after close and publishes successfully", func(t *testing.T) {
		// Create a separate adapter for this test
		adapter, err := rabbitmq.NewRabbitMQAdapter(config.RabbitMQConfig{
			URL:        testAmqpEndpoint,
			MaxRetries: 3,
			RetryDelay: 100 * time.Millisecond,
			ExchangeConfigs: []config.ExchangeConfig{
				{
					Name:       "exchange.order",
					Type:       "direct",
					Durable:    true,
					AutoDelete: false,
				},
			},
		})
		if err != nil {
			t.Fatalf("failed to create adapter: %v", err)
		}
		defer adapter.Close()

		// Publish should work
		err = adapter.PublishRaw(ctx, "order.reconnect_test", "order", []byte(`{"test":"before"}`))
		if err != nil {
			t.Fatalf("initial publish failed: %v", err)
		}
	})

	t.Run("health check fails after close", func(t *testing.T) {
		adapter, err := rabbitmq.NewRabbitMQAdapter(config.RabbitMQConfig{
			URL:        testAmqpEndpoint,
			MaxRetries: 0,
			RetryDelay: 0,
			ExchangeConfigs: []config.ExchangeConfig{
				{
					Name:       "exchange.order",
					Type:       "direct",
					Durable:    true,
					AutoDelete: false,
				},
			},
		})
		if err != nil {
			t.Fatalf("failed to create adapter: %v", err)
		}

		_ = adapter.Close()

		err = adapter.HealthCheck()
		if err == nil {
			t.Fatal("expected health check to fail after close")
		}
	})
}

// mockEvent implements domain.Event for testing
type mockEvent struct {
	name       string
	entityName string
	data       map[string]string
}

func (e *mockEvent) GetName() string       { return e.name }
func (e *mockEvent) GetEntityName() string { return e.entityName }
