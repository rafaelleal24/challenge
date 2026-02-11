package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/repository"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

func createTestOrder(t *testing.T, orderRepo interface {
	Create(ctx context.Context, order *domain.Order) error
}, customerID domain.ID) *domain.Order {
	t.Helper()
	items := []domain.OrderItem{
		*domain.NewOrderItem("aabbccddee112233aabbccd1", "Product A", 2, domain.Amount(1000)),
		*domain.NewOrderItem("aabbccddee112233aabbccd2", "Product B", 1, domain.Amount(2000)),
	}
	order := domain.NewOrder(customerID, domain.OrderStatusCreated, items)
	if err := orderRepo.Create(context.Background(), order); err != nil {
		t.Fatalf("setup: create order failed: %v", err)
	}
	return order
}

func TestOrderRepository_Create(t *testing.T) {
	outboxRepo := repository.NewOutboxRepository(testDB)
	orderRepo := repository.NewOrderRepository(testDB, outboxRepo)
	ctx := context.Background()

	t.Run("creates order and assigns IDs", func(t *testing.T) {
		items := []domain.OrderItem{
			*domain.NewOrderItem("aabbccddee112233aabbccd1", "Product A", 2, domain.Amount(1500)),
		}
		order := domain.NewOrder("ccddaabbee112233aabbccdd", domain.OrderStatusCreated, items)

		err := orderRepo.Create(ctx, order)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order.ID == "" {
			t.Fatal("expected order ID to be assigned")
		}
		if len(string(order.ID)) != 24 {
			t.Fatalf("expected 24-char hex order ID, got %q", order.ID)
		}
		for i, item := range order.Items {
			if item.ID == "" {
				t.Fatalf("expected item[%d] ID to be assigned", i)
			}
		}
	})

	t.Run("rejects order with pre-existing ID", func(t *testing.T) {
		items := []domain.OrderItem{
			*domain.NewOrderItem("aabbccddee112233aabbccd1", "Product A", 1, domain.Amount(500)),
		}
		order := domain.NewOrder("ccddaabbee112233aabbccdd", domain.OrderStatusCreated, items)
		order.ID = "aabbccddee112233aabbccdd"

		err := orderRepo.Create(ctx, order)
		if err == nil {
			t.Fatal("expected error for order with existing ID, got nil")
		}
	})

	t.Run("calculates total amount correctly", func(t *testing.T) {
		items := []domain.OrderItem{
			*domain.NewOrderItem("aabbccddee112233aabbccd1", "Product A", 3, domain.Amount(1000)),
			*domain.NewOrderItem("aabbccddee112233aabbccd2", "Product B", 2, domain.Amount(2500)),
		}
		order := domain.NewOrder("ccddaabbee112233aabbccdd", domain.OrderStatusCreated, items)

		_ = orderRepo.Create(ctx, order)

		// 3*1000 + 2*2500 = 3000 + 5000 = 8000
		expectedTotal := domain.Amount(8000)
		if order.TotalAmount != expectedTotal {
			t.Fatalf("expected total %d, got %d", expectedTotal, order.TotalAmount)
		}
	})
}

func TestOrderRepository_GetByID(t *testing.T) {
	outboxRepo := repository.NewOutboxRepository(testDB)
	orderRepo := repository.NewOrderRepository(testDB, outboxRepo)
	ctx := context.Background()
	customerID := domain.ID("ccddaabbee112233aabbccdd")

	t.Run("returns order by ID", func(t *testing.T) {
		created := createTestOrder(t, orderRepo, customerID)

		found, err := orderRepo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected id %s, got %s", created.ID, found.ID)
		}
		if found.CustomerID != customerID {
			t.Fatalf("expected customer id %s, got %s", customerID, found.CustomerID)
		}
		if found.Status != domain.OrderStatusCreated {
			t.Fatalf("expected status %s, got %s", domain.OrderStatusCreated, found.Status)
		}
		if len(found.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(found.Items))
		}
	})

	t.Run("returns not found for non-existing order", func(t *testing.T) {
		_, err := orderRepo.GetByID(ctx, "aabbccddee112233aabb0000")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("returns error for invalid ID", func(t *testing.T) {
		_, err := orderRepo.GetByID(ctx, "bad-id")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindInvalidRequest) {
			t.Fatalf("expected KindInvalidRequest, got %v", err)
		}
	})
}

func TestOrderRepository_GetByCustomerID(t *testing.T) {
	freshDB := testClient.Database("test_order_by_customer")
	outboxRepo := repository.NewOutboxRepository(freshDB)
	orderRepo := repository.NewOrderRepository(freshDB, outboxRepo)
	ctx := context.Background()
	customerID := domain.ID("ccddaabbee112233aabbcc01")

	t.Run("returns empty list when no orders", func(t *testing.T) {
		orders, err := orderRepo.GetByCustomerID(ctx, customerID, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(orders) != 0 {
			t.Fatalf("expected 0 orders, got %d", len(orders))
		}
	})

	t.Run("returns orders for customer", func(t *testing.T) {
		createTestOrder(t, orderRepo, customerID)
		createTestOrder(t, orderRepo, customerID)

		otherCustomer := domain.ID("ccddaabbee112233aabbcc02")
		createTestOrder(t, orderRepo, otherCustomer)

		orders, err := orderRepo.GetByCustomerID(ctx, customerID, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(orders) != 2 {
			t.Fatalf("expected 2 orders for customer, got %d", len(orders))
		}
	})
}

func TestOrderRepository_GetByStatus(t *testing.T) {
	freshDB := testClient.Database("test_order_by_status")
	outboxRepo := repository.NewOutboxRepository(freshDB)
	orderRepo := repository.NewOrderRepository(freshDB, outboxRepo)
	ctx := context.Background()
	customerID := domain.ID("ccddaabbee112233aabbccdd")

	t.Run("returns empty for status with no orders", func(t *testing.T) {
		orders, err := orderRepo.GetByStatus(ctx, domain.OrderStatusShipped, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(orders) != 0 {
			t.Fatalf("expected 0 orders, got %d", len(orders))
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		createTestOrder(t, orderRepo, customerID)

		orders, err := orderRepo.GetByStatus(ctx, domain.OrderStatusCreated, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(orders) < 1 {
			t.Fatal("expected at least 1 order with status 'created'")
		}

		shipped, err := orderRepo.GetByStatus(ctx, domain.OrderStatusShipped, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(shipped) != 0 {
			t.Fatalf("expected 0 shipped orders, got %d", len(shipped))
		}
	})
}

func TestOrderRepository_UpdateStatusWithOutbox(t *testing.T) {
	outboxRepo := repository.NewOutboxRepository(testDB)
	orderRepo := repository.NewOrderRepository(testDB, outboxRepo)
	ctx := context.Background()
	customerID := domain.ID("ccddaabbee112233aabbccdd")

	t.Run("updates status and creates outbox entry", func(t *testing.T) {
		order := createTestOrder(t, orderRepo, customerID)

		event := domain.NewOrderUpdateStatusEvent(
			order.ID, domain.OrderStatusProcessing, domain.OrderStatusCreated, order.CreatedAt, customerID,
		)
		err := orderRepo.UpdateStatusWithOutbox(ctx, order.ID, domain.OrderStatusProcessing, event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		updated, _ := orderRepo.GetByID(ctx, order.ID)
		if updated.Status != domain.OrderStatusProcessing {
			t.Fatalf("expected status %s, got %s", domain.OrderStatusProcessing, updated.Status)
		}

		entries, err := outboxRepo.FetchPending(ctx, 100)
		if err != nil {
			t.Fatalf("expected no error fetching outbox, got %v", err)
		}
		found := false
		for _, e := range entries {
			if e.EventName == "order.update_status" && e.EntityName == "order" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected outbox entry for order.update_status")
		}
	})

	t.Run("returns not found for non-existing order", func(t *testing.T) {
		nonExistingID := domain.ID("aabbccddee112233aabb0000")
		event := domain.NewOrderUpdateStatusEvent(
			nonExistingID, domain.OrderStatusProcessing, domain.OrderStatusCreated, time.Now(), customerID,
		)
		err := orderRepo.UpdateStatusWithOutbox(ctx, nonExistingID, domain.OrderStatusProcessing, event)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})
}

func TestOrderRepository_Delete(t *testing.T) {
	outboxRepo := repository.NewOutboxRepository(testDB)
	orderRepo := repository.NewOrderRepository(testDB, outboxRepo)
	ctx := context.Background()
	customerID := domain.ID("ccddaabbee112233aabbccdd")

	t.Run("deletes existing order", func(t *testing.T) {
		order := createTestOrder(t, orderRepo, customerID)

		err := orderRepo.Delete(ctx, order.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, err = orderRepo.GetByID(ctx, order.ID)
		if err == nil {
			t.Fatal("expected not found after delete")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("returns not found for already deleted", func(t *testing.T) {
		err := orderRepo.Delete(ctx, "aabbccddee112233aabb0000")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})
}
