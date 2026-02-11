package repository_test

import (
	"context"
	"testing"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/repository"
	"github.com/rafaelleal24/challenge/internal/adapters/outbox"
)

func TestOutboxRepository_Insert(t *testing.T) {
	freshDB := testClient.Database("test_outbox_insert")
	repo := repository.NewOutboxRepository(freshDB)
	ctx := context.Background()

	t.Run("inserts entry successfully", func(t *testing.T) {
		entry := outbox.Entry{
			EventName:  "order.created",
			EntityName: "order",
			EventData:  []byte(`{"order_id":"123"}`),
		}

		err := repo.Insert(ctx, entry)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestOutboxRepository_FetchPending(t *testing.T) {
	freshDB := testClient.Database("test_outbox_fetch")
	repo := repository.NewOutboxRepository(freshDB)
	ctx := context.Background()

	t.Run("returns empty when no entries", func(t *testing.T) {
		entries, err := repo.FetchPending(ctx, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("fetches inserted entries", func(t *testing.T) {
		_ = repo.Insert(ctx, outbox.Entry{EventName: "evt.1", EntityName: "entity", EventData: []byte(`{}`)})
		_ = repo.Insert(ctx, outbox.Entry{EventName: "evt.2", EntityName: "entity", EventData: []byte(`{}`)})

		entries, err := repo.FetchPending(ctx, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
		// Each entry should have an ID
		for i, e := range entries {
			if e.ID == "" {
				t.Fatalf("entry[%d] has empty ID", i)
			}
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		entries, err := repo.FetchPending(ctx, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry (limit=1), got %d", len(entries))
		}
	})
}

func TestOutboxRepository_Delete(t *testing.T) {
	freshDB := testClient.Database("test_outbox_delete")
	repo := repository.NewOutboxRepository(freshDB)
	ctx := context.Background()

	t.Run("deletes entry by ID", func(t *testing.T) {
		_ = repo.Insert(ctx, outbox.Entry{EventName: "evt.del", EntityName: "entity", EventData: []byte(`{}`)})

		entries, _ := repo.FetchPending(ctx, 10)
		if len(entries) == 0 {
			t.Fatal("setup: expected at least 1 entry")
		}

		err := repo.Delete(ctx, entries[0].ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		remaining, _ := repo.FetchPending(ctx, 10)
		if len(remaining) != 0 {
			t.Fatalf("expected 0 entries after delete, got %d", len(remaining))
		}
	})

	t.Run("returns error for invalid ID", func(t *testing.T) {
		err := repo.Delete(ctx, "bad-id")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
