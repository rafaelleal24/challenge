package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

func RateLimit(limiter RateLimiter, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("%s:%s:%s", c.Request.Method, c.FullPath(), c.ClientIP())

		allowed, err := limiter.Allow(c.Request.Context(), key, limit, window)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}
