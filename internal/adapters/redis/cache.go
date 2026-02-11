package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/rafaelleal24/challenge/internal/core/port"
)

type Cache[T any] struct {
	client *Client
	prefix string
}

func NewCache[T any](client *Client, prefix string) port.CachePort[T] {
	return &Cache[T]{client: client, prefix: prefix}
}

func (c *Cache[T]) key(id string) string {
	return fmt.Sprintf("%s:%s", c.prefix, id)
}

func (c *Cache[T]) Get(ctx context.Context, id string) (*T, error) {
	data, err := c.client.Get(ctx, c.key(id))
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var value T
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return nil, err
	}
	return &value, nil
}

func (c *Cache[T]) Set(ctx context.Context, id string, value *T, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(id), string(data), ttl)
}

func (c *Cache[T]) SetNX(ctx context.Context, id string, value *T, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return c.client.SetNX(ctx, c.key(id), string(data), ttl)
}

func (c *Cache[T]) Del(ctx context.Context, id string) error {
	return c.client.Del(ctx, c.key(id))
}
