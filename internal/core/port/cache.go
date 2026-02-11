package port

import (
	"context"
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock
type CachePort[T any] interface {
	Get(ctx context.Context, key string) (*T, error)
	Set(ctx context.Context, key string, value *T, ttl time.Duration) error
	SetNX(ctx context.Context, key string, value *T, ttl time.Duration) (bool, error)
	Del(ctx context.Context, key string) error
}
