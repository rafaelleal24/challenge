package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelleal24/challenge/internal/core/port/mock"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
	"go.uber.org/mock/gomock"
)

type testPayload struct {
	Value string `json:"value"`
}

func setupIdempotencyService(t *testing.T) (*IdempotencyService[testPayload], *mock.MockCachePort[IdempotencyEntry[testPayload]]) {
	ctrl := gomock.NewController(t)
	cache := mock.NewMockCachePort[IdempotencyEntry[testPayload]](ctrl)
	svc := NewIdempotencyService[testPayload](cache, 15*time.Minute, 50*time.Millisecond, 500*time.Millisecond)
	return svc, cache
}

func TestIdempotencyService_Claim(t *testing.T) {
	t.Run("first request - claims successfully", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"
		hash := "hash123"

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(true, nil)

		result, err := svc.Claim(context.Background(), key, hash)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result != nil {
			t.Fatal("expected nil result for first claim")
		}
	})

	t.Run("duplicate request - completed - returns result", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"
		hash := "hash123"
		expectedResult := &testPayload{Value: "order-result"}

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(false, nil)

		cache.EXPECT().
			Get(gomock.Any(), key).
			Return(&IdempotencyEntry[testPayload]{
				Status:      IdempotencyCompleted,
				PayloadHash: hash,
				Result:      expectedResult,
			}, nil)

		result, err := svc.Claim(context.Background(), key, hash)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if result.Value != expectedResult.Value {
			t.Fatalf("expected %q, got %q", expectedResult.Value, result.Value)
		}
	})

	t.Run("duplicate request - different payload hash", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(false, nil)

		cache.EXPECT().
			Get(gomock.Any(), key).
			Return(&IdempotencyEntry[testPayload]{
				Status:      IdempotencyCompleted,
				PayloadHash: "different-hash",
				Result:      &testPayload{Value: "old"},
			}, nil)

		_, err := svc.Claim(context.Background(), key, "new-hash")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindUnprocessableEntity) {
			t.Fatalf("expected KindUnprocessableEntity, got %v", err)
		}
	})

	t.Run("duplicate request - entry disappeared (previous failed)", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"
		hash := "hash123"

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(false, nil)

		cache.EXPECT().
			Get(gomock.Any(), key).
			Return(nil, nil)

		_, err := svc.Claim(context.Background(), key, hash)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindConflict) {
			t.Fatalf("expected KindConflict, got %v", err)
		}
	})

	t.Run("SetNX cache error", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"
		hash := "hash123"

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(false, errors.New("redis down"))

		_, err := svc.Claim(context.Background(), key, hash)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("duplicate request - still processing - polls and completes", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"
		hash := "hash123"
		expectedResult := &testPayload{Value: "order-result"}

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(false, nil)

		firstCall := cache.EXPECT().
			Get(gomock.Any(), key).
			Return(&IdempotencyEntry[testPayload]{
				Status:      IdempotencyProcessing,
				PayloadHash: hash,
			}, nil)

		secondCall := cache.EXPECT().
			Get(gomock.Any(), key).
			Return(&IdempotencyEntry[testPayload]{
				Status:      IdempotencyProcessing,
				PayloadHash: hash,
			}, nil).
			After(firstCall)

		cache.EXPECT().
			Get(gomock.Any(), key).
			Return(&IdempotencyEntry[testPayload]{
				Status:      IdempotencyCompleted,
				PayloadHash: hash,
				Result:      expectedResult,
			}, nil).
			After(secondCall)

		result, err := svc.Claim(context.Background(), key, hash)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if result.Value != expectedResult.Value {
			t.Fatalf("expected %q, got %q", expectedResult.Value, result.Value)
		}
	})

	t.Run("duplicate request - still processing - times out", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cache := mock.NewMockCachePort[IdempotencyEntry[testPayload]](ctrl)
		svc := NewIdempotencyService[testPayload](cache, 15*time.Minute, 20*time.Millisecond, 80*time.Millisecond)

		key := "idem-key-1"
		hash := "hash123"

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(false, nil)

		// Always returns processing
		cache.EXPECT().
			Get(gomock.Any(), key).
			Return(&IdempotencyEntry[testPayload]{
				Status:      IdempotencyProcessing,
				PayloadHash: hash,
			}, nil).
			AnyTimes()

		_, err := svc.Claim(context.Background(), key, hash)
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindConflict) {
			t.Fatalf("expected KindConflict, got %v", err)
		}
	})

	t.Run("duplicate request - context cancelled while polling", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cache := mock.NewMockCachePort[IdempotencyEntry[testPayload]](ctrl)
		svc := NewIdempotencyService[testPayload](cache, 15*time.Minute, 50*time.Millisecond, 5*time.Second)

		key := "idem-key-1"
		hash := "hash123"

		cache.EXPECT().
			SetNX(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			Return(false, nil)

		cache.EXPECT().
			Get(gomock.Any(), key).
			Return(&IdempotencyEntry[testPayload]{
				Status:      IdempotencyProcessing,
				PayloadHash: hash,
			}, nil).
			AnyTimes()

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(30 * time.Millisecond)
			cancel()
		}()

		_, err := svc.Claim(ctx, key, hash)
		if err == nil {
			t.Fatal("expected context error, got nil")
		}
	})
}

func TestIdempotencyService_Complete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"
		hash := "hash123"
		result := &testPayload{Value: "order-result"}

		cache.EXPECT().
			Set(gomock.Any(), key, gomock.Any(), 15*time.Minute).
			DoAndReturn(func(_ context.Context, _ string, entry *IdempotencyEntry[testPayload], _ time.Duration) error {
				if entry.Status != IdempotencyCompleted {
					t.Fatalf("expected status %q, got %q", IdempotencyCompleted, entry.Status)
				}
				if entry.PayloadHash != hash {
					t.Fatalf("expected hash %q, got %q", hash, entry.PayloadHash)
				}
				if entry.Result.Value != result.Value {
					t.Fatalf("expected result %q, got %q", result.Value, entry.Result.Value)
				}
				return nil
			})

		svc.Complete(context.Background(), key, hash, result)
	})

	t.Run("cache error is logged but does not panic", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)

		cache.EXPECT().
			Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("redis error"))

		// Should not panic
		svc.Complete(context.Background(), "key", "hash", &testPayload{})
	})
}

func TestIdempotencyService_Release(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)
		key := "idem-key-1"

		cache.EXPECT().
			Del(gomock.Any(), key).
			Return(nil)

		svc.Release(context.Background(), key)
	})

	t.Run("cache error is logged but does not panic", func(t *testing.T) {
		svc, cache := setupIdempotencyService(t)

		cache.EXPECT().
			Del(gomock.Any(), gomock.Any()).
			Return(errors.New("redis error"))

		// Should not panic
		svc.Release(context.Background(), "key")
	})
}
