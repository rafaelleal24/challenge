package redis_test

import (
	"context"
	"testing"
	"time"

	adaptredis "github.com/rafaelleal24/challenge/internal/adapters/redis"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := adaptredis.NewRateLimiter(testClient)
	ctx := context.Background()

	t.Run("allows requests under limit", func(t *testing.T) {
		key := "rate-test-under"
		for i := 0; i < 3; i++ {
			allowed, err := rl.Allow(ctx, key, 5, 1*time.Minute)
			if err != nil {
				t.Fatalf("request %d: expected no error, got %v", i, err)
			}
			if !allowed {
				t.Fatalf("request %d: expected to be allowed", i)
			}
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		key := "rate-test-over"
		limit := 2
		for i := 0; i < limit; i++ {
			_, _ = rl.Allow(ctx, key, limit, 1*time.Minute)
		}

		allowed, err := rl.Allow(ctx, key, limit, 1*time.Minute)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if allowed {
			t.Fatal("expected request to be blocked (over limit)")
		}
	})

	t.Run("window expires and resets count", func(t *testing.T) {
		key := "rate-test-expire"
		limit := 1
		window := 2 * time.Second

		allowed, _ := rl.Allow(ctx, key, limit, window)
		if !allowed {
			t.Fatal("first request should be allowed")
		}

		allowed, _ = rl.Allow(ctx, key, limit, window)
		if allowed {
			t.Fatal("second request should be blocked")
		}

		time.Sleep(3 * time.Second)

		allowed, _ = rl.Allow(ctx, key, limit, window)
		if !allowed {
			t.Fatal("request after window should be allowed again")
		}
	})
}
