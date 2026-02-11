package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rafaelleal24/challenge/internal/core/logger"
	"github.com/rafaelleal24/challenge/internal/core/port"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

type IdempotencyStatus string

const (
	IdempotencyProcessing IdempotencyStatus = "processing"
	IdempotencyCompleted  IdempotencyStatus = "completed"
)

type IdempotencyEntry[T any] struct {
	Status      IdempotencyStatus `json:"status"`
	PayloadHash string            `json:"payload_hash"`
	Result      *T                `json:"result,omitempty"`
}

type IdempotencyService[T any] struct {
	cache        port.CachePort[IdempotencyEntry[T]]
	ttl          time.Duration
	pollInterval time.Duration
	pollTimeout  time.Duration
}

func NewIdempotencyService[T any](
	cache port.CachePort[IdempotencyEntry[T]],
	ttl time.Duration,
	pollInterval time.Duration,
	pollTimeout time.Duration,
) *IdempotencyService[T] {
	return &IdempotencyService[T]{
		cache:        cache,
		ttl:          ttl,
		pollInterval: pollInterval,
		pollTimeout:  pollTimeout,
	}
}

func (s *IdempotencyService[T]) Claim(ctx context.Context, key, payloadHash string) (*T, error) {
	claimed, err := s.cache.SetNX(ctx, key, &IdempotencyEntry[T]{
		Status:      IdempotencyProcessing,
		PayloadHash: payloadHash,
	}, s.ttl)
	if err != nil {
		return nil, fmt.Errorf("idempotency claim failed: %w", err)
	}

	if claimed {
		return nil, nil
	}

	return s.waitForCompletion(ctx, key, payloadHash)
}

func (s *IdempotencyService[T]) Complete(ctx context.Context, key, payloadHash string, result *T) {
	err := s.cache.Set(ctx, key, &IdempotencyEntry[T]{
		Status:      IdempotencyCompleted,
		PayloadHash: payloadHash,
		Result:      result,
	}, s.ttl)
	if err != nil {
		logger.Error(ctx, "idempotency: complete failed", err, map[string]any{
			"idempotency_key": key,
			"payload_hash":    payloadHash,
		})
	}
}

func (s *IdempotencyService[T]) Release(ctx context.Context, key string) {
	if err := s.cache.Del(ctx, key); err != nil {
		logger.Error(ctx, "idempotency: release failed", err, map[string]any{
			"idempotency_key": key,
		})
	}
}

func (s *IdempotencyService[T]) checkEntry(ctx context.Context, key, payloadHash string) (*T, error) {
	entry, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if entry == nil {
		return nil, serviceerrors.NewConflictError("previous request failed, retry with the same key")
	}
	if entry.PayloadHash != payloadHash {
		return nil, serviceerrors.NewUnprocessableEntityError("idempotency key already used with a different payload")
	}
	if entry.Status == IdempotencyCompleted {
		return entry.Result, nil
	}
	return nil, nil
}

func (s *IdempotencyService[T]) waitForCompletion(ctx context.Context, key, payloadHash string) (*T, error) {
	result, err := s.checkEntry(ctx, key, payloadHash)
	if result != nil || err != nil {
		return result, err
	}

	timeout := time.After(s.pollTimeout)
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, serviceerrors.NewConflictError("idempotency key still being processed, timed out")
		case <-ticker.C:
			result, err := s.checkEntry(ctx, key, payloadHash)
			if result != nil || err != nil {
				return result, err
			}
		}
	}
}
