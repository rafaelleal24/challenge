package redis_test

import (
	"context"
	"testing"
	"time"

	adaptredis "github.com/rafaelleal24/challenge/internal/adapters/redis"
)

type testCacheItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestCache_SetAndGet(t *testing.T) {
	cache := adaptredis.NewCache[testCacheItem](testClient, "test-cache")
	ctx := context.Background()

	t.Run("set and get value", func(t *testing.T) {
		item := &testCacheItem{Name: "widget", Value: 42}
		err := cache.Set(ctx, "item-1", item, 1*time.Minute)
		if err != nil {
			t.Fatalf("expected no error on set, got %v", err)
		}

		got, err := cache.Get(ctx, "item-1")
		if err != nil {
			t.Fatalf("expected no error on get, got %v", err)
		}
		if got == nil {
			t.Fatal("expected item, got nil")
		}
		if got.Name != item.Name {
			t.Fatalf("expected name %q, got %q", item.Name, got.Name)
		}
		if got.Value != item.Value {
			t.Fatalf("expected value %d, got %d", item.Value, got.Value)
		}
	})

	t.Run("get returns nil for missing key", func(t *testing.T) {
		got, err := cache.Get(ctx, "nonexistent-key")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("ttl expires value", func(t *testing.T) {
		item := &testCacheItem{Name: "ephemeral", Value: 1}
		err := cache.Set(ctx, "ttl-item", item, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		got, err := cache.Get(ctx, "ttl-item")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil (expired), got %+v", got)
		}
	})
}

func TestCache_SetNX(t *testing.T) {
	cache := adaptredis.NewCache[testCacheItem](testClient, "test-setnx")
	ctx := context.Background()

	t.Run("first SetNX succeeds", func(t *testing.T) {
		item := &testCacheItem{Name: "first", Value: 1}
		ok, err := cache.SetNX(ctx, "nx-key", item, 1*time.Minute)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !ok {
			t.Fatal("expected SetNX to succeed (first write)")
		}
	})

	t.Run("second SetNX fails (key exists)", func(t *testing.T) {
		item := &testCacheItem{Name: "second", Value: 2}
		ok, err := cache.SetNX(ctx, "nx-key", item, 1*time.Minute)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ok {
			t.Fatal("expected SetNX to fail (key already exists)")
		}

		got, _ := cache.Get(ctx, "nx-key")
		if got == nil {
			t.Fatal("expected original item")
		}
		if got.Name != "first" {
			t.Fatalf("expected original name 'first', got %q", got.Name)
		}
	})
}

func TestCache_Del(t *testing.T) {
	cache := adaptredis.NewCache[testCacheItem](testClient, "test-del")
	ctx := context.Background()

	t.Run("deletes existing key", func(t *testing.T) {
		item := &testCacheItem{Name: "to-delete", Value: 99}
		_ = cache.Set(ctx, "del-key", item, 1*time.Minute)

		err := cache.Del(ctx, "del-key")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		got, _ := cache.Get(ctx, "del-key")
		if got != nil {
			t.Fatalf("expected nil after delete, got %+v", got)
		}
	})

	t.Run("delete non-existing key does not error", func(t *testing.T) {
		err := cache.Del(ctx, "nonexistent-del-key")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
