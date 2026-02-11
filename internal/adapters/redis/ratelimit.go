package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/rafaelleal24/challenge/internal/adapters/http/middleware"
)

var rateLimitScript = goredis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
	redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)

type RateLimiter struct {
	client *Client
}

func NewRateLimiter(client *Client) middleware.RateLimiter {
	return &RateLimiter{client: client}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	count, err := rateLimitScript.Run(ctx, r.client.rdb, []string{redisKey}, int(window.Seconds())).Int()
	if err != nil {
		return false, err
	}
	return count <= limit, nil
}
