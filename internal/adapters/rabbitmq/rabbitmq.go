package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rafaelleal24/challenge/internal/adapters/config"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQAdapter struct {
	mu      sync.Mutex
	conn    *amqp.Connection
	channel *amqp.Channel
	config  config.RabbitMQConfig
}

func NewRabbitMQAdapter(cfg config.RabbitMQConfig) (*RabbitMQAdapter, error) {
	adapter := &RabbitMQAdapter{config: cfg, mu: sync.Mutex{}}

	if err := adapter.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return adapter, nil
}

func (r *RabbitMQAdapter) connect() error {
	conn, err := amqp.Dial(r.config.URL)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	for _, ec := range r.config.ExchangeConfigs {
		if err := ch.ExchangeDeclare(ec.Name, ec.Type, ec.Durable, ec.AutoDelete, false, false, nil); err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("failed to declare exchange %s: %w", ec.Name, err)
		}
	}

	r.conn = conn
	r.channel = ch
	return nil
}

func (r *RabbitMQAdapter) reconnect() error {
	if r.channel != nil {
		r.channel.Close()
		r.channel = nil
	}
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	return r.connect()
}

func (r *RabbitMQAdapter) Publish(ctx context.Context, event domain.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		logger.Error(ctx, "failed to marshal event", err, map[string]any{
			"event_name":  event.GetName(),
			"entity_name": event.GetEntityName(),
		})
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return r.publish(ctx, event.GetName(), event.GetEntityName(), body)
}

func (r *RabbitMQAdapter) PublishRaw(ctx context.Context, eventName, entityName string, data []byte) error {
	return r.publish(ctx, eventName, entityName, data)
}

func (r *RabbitMQAdapter) publish(ctx context.Context, eventName, entityName string, body []byte) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
	}

	exchange := fmt.Sprintf("exchange.%s", entityName)
	routingKey := eventName

	var lastErr error
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.config.RetryDelay)
		}

		r.mu.Lock()

		if r.channel == nil {
			if err := r.reconnect(); err != nil {
				r.mu.Unlock()
				lastErr = fmt.Errorf("reconnect failed: %w", err)
				logger.Error(ctx, "publish: reconnect failed", err, map[string]any{
					"attempt": attempt + 1,
				})
				continue
			}
		}

		err := r.channel.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
		if err != nil {
			r.channel = nil
			r.mu.Unlock()
			lastErr = err
			logger.Error(ctx, "publish: failed", err, map[string]any{
				"attempt": attempt + 1,
			})
			continue
		}

		r.mu.Unlock()
		return nil
	}

	return fmt.Errorf("failed to publish after %d attempts: %w", r.config.MaxRetries+1, lastErr)
}

func (r *RabbitMQAdapter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing channel: %w", err))
		}
		r.channel = nil
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing connection: %w", err))
		}
		r.conn = nil
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing RabbitMQ: %v", errs)
	}
	return nil
}

func (r *RabbitMQAdapter) HealthCheck() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
	}
	if r.channel == nil {
		return fmt.Errorf("channel is nil")
	}
	return nil
}
