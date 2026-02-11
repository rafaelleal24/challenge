package repository_test

import (
	"context"
	"testing"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/repository"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

func TestCustomerRepository_Create(t *testing.T) {
	repo := repository.NewCustomerRepository(testDB)
	ctx := context.Background()

	t.Run("creates customer and returns valid ID", func(t *testing.T) {
		id, err := repo.Create(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id == "" {
			t.Fatal("expected non-empty ID")
		}
		if len(string(id)) != 24 {
			t.Fatalf("expected 24-char hex ID, got %q (len=%d)", id, len(string(id)))
		}
	})
}

func TestCustomerRepository_Exists(t *testing.T) {
	repo := repository.NewCustomerRepository(testDB)
	ctx := context.Background()

	t.Run("returns true for existing customer", func(t *testing.T) {
		id, err := repo.Create(ctx)
		if err != nil {
			t.Fatalf("setup: create failed: %v", err)
		}

		exists, err := repo.Exists(ctx, id)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !exists {
			t.Fatal("expected customer to exist")
		}
	})

	t.Run("returns error for non-existing customer", func(t *testing.T) {
		_, err := repo.Exists(ctx, "aabbccddee112233aabbccdd")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("returns error for invalid ID format", func(t *testing.T) {
		_, err := repo.Exists(ctx, "invalid-id")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindInvalidRequest) {
			t.Fatalf("expected KindInvalidRequest, got %v", err)
		}
	})
}
