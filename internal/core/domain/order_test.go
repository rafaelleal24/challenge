package domain

import (
	"testing"
	"time"
)

func TestOrderStatus_IsValid(t *testing.T) {
	tests := []struct {
		status OrderStatus
		valid  bool
	}{
		{OrderStatusCreated, true},
		{OrderStatusProcessing, true},
		{OrderStatusShipped, true},
		{OrderStatusDelivered, true},
		{OrderStatusCancelled, true},
		{"invalid", false},
		{"", false},
		{"CREATED", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("OrderStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestNewOrderItem(t *testing.T) {
	item := NewOrderItem("prod123", "Widget", 3, NewAmountFromCents(1500))

	if item.ProductID != "prod123" {
		t.Fatalf("expected ProductID 'prod123', got %q", item.ProductID)
	}
	if item.ProductName != "Widget" {
		t.Fatalf("expected ProductName 'Widget', got %q", item.ProductName)
	}
	if item.Quantity != 3 {
		t.Fatalf("expected Quantity 3, got %d", item.Quantity)
	}
	if item.UnitPrice != 1500 {
		t.Fatalf("expected UnitPrice 1500, got %d", item.UnitPrice)
	}
	if item.ID != "" {
		t.Fatalf("expected empty ID, got %q", item.ID)
	}
}

func TestOrderItem_CalculateTotalAmount(t *testing.T) {
	tests := []struct {
		name     string
		price    Amount
		qty      int
		expected Amount
	}{
		{"single item", 1500, 1, 1500},
		{"multiple items", 1500, 3, 4500},
		{"zero quantity", 1500, 0, 0},
		{"zero price", 0, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &OrderItem{UnitPrice: tt.price, Quantity: tt.qty}
			if got := item.CalculateTotalAmount(); got != tt.expected {
				t.Errorf("CalculateTotalAmount() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestCalculateTotalAmount(t *testing.T) {
	tests := []struct {
		name     string
		items    []OrderItem
		expected Amount
	}{
		{
			"single item",
			[]OrderItem{{UnitPrice: 1000, Quantity: 2}},
			2000,
		},
		{
			"multiple items",
			[]OrderItem{
				{UnitPrice: 1000, Quantity: 2},
				{UnitPrice: 500, Quantity: 3},
			},
			3500,
		},
		{
			"empty items",
			[]OrderItem{},
			0,
		},
		{
			"nil items",
			nil,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateTotalAmount(tt.items); got != tt.expected {
				t.Errorf("CalculateTotalAmount() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNewOrder(t *testing.T) {
	items := []OrderItem{
		{ProductID: "p1", ProductName: "A", Quantity: 2, UnitPrice: 1000},
		{ProductID: "p2", ProductName: "B", Quantity: 1, UnitPrice: 500},
	}

	before := time.Now()
	order := NewOrder("cust1", OrderStatusCreated, items)
	after := time.Now()

	if order.CustomerID != "cust1" {
		t.Fatalf("expected CustomerID 'cust1', got %q", order.CustomerID)
	}
	if order.Status != OrderStatusCreated {
		t.Fatalf("expected status 'created', got %q", order.Status)
	}
	if order.ID != "" {
		t.Fatalf("expected empty ID, got %q", order.ID)
	}
	if len(order.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(order.Items))
	}

	// TotalAmount = (1000*2) + (500*1) = 2500
	if order.TotalAmount != 2500 {
		t.Fatalf("expected TotalAmount 2500, got %d", order.TotalAmount)
	}

	if order.CreatedAt.Before(before) || order.CreatedAt.After(after) {
		t.Fatalf("CreatedAt not in expected range")
	}
	if order.UpdatedAt.Before(before) || order.UpdatedAt.After(after) {
		t.Fatalf("UpdatedAt not in expected range")
	}
}

func TestNewOrder_EmptyItems(t *testing.T) {
	order := NewOrder("cust1", OrderStatusCreated, []OrderItem{})

	if order.TotalAmount != 0 {
		t.Fatalf("expected TotalAmount 0 for empty items, got %d", order.TotalAmount)
	}
	if len(order.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(order.Items))
	}
}

func TestNewOrderUpdateStatusEvent(t *testing.T) {
	now := time.Now()
	event := NewOrderUpdateStatusEvent("order1", OrderStatusProcessing, OrderStatusCreated, now, "cust1")

	if event.OrderID != "order1" {
		t.Fatalf("expected OrderID 'order1', got %q", event.OrderID)
	}
	if event.Status != OrderStatusProcessing {
		t.Fatalf("expected Status 'processing', got %q", event.Status)
	}
	if event.OldStatus != OrderStatusCreated {
		t.Fatalf("expected OldStatus 'created', got %q", event.OldStatus)
	}
	if !event.UpdatedAt.Equal(now) {
		t.Fatalf("expected UpdatedAt %v, got %v", now, event.UpdatedAt)
	}
	if event.CustomerID != "cust1" {
		t.Fatalf("expected CustomerID 'cust1', got %q", event.CustomerID)
	}
}

func TestOrderUpdateStatusEvent_GetName(t *testing.T) {
	event := &OrderUpdateStatusEvent{}
	if got := event.GetName(); got != "order.update_status" {
		t.Fatalf("expected 'order.update_status', got %q", got)
	}
}

func TestOrderUpdateStatusEvent_GetEntityName(t *testing.T) {
	event := &OrderUpdateStatusEvent{}
	if got := event.GetEntityName(); got != "order" {
		t.Fatalf("expected 'order', got %q", got)
	}
}
