package domain

import (
	"testing"
	"time"
)

func TestNewProduct(t *testing.T) {
	before := time.Now()
	p := NewProduct("Widget", "A fine widget", NewAmountFromCents(4999), 25)
	after := time.Now()

	if p.Name != "Widget" {
		t.Fatalf("expected name 'Widget', got %q", p.Name)
	}
	if p.Description != "A fine widget" {
		t.Fatalf("expected description 'A fine widget', got %q", p.Description)
	}
	if p.Price != 4999 {
		t.Fatalf("expected price 4999, got %d", p.Price)
	}
	if p.Stock != 25 {
		t.Fatalf("expected stock 25, got %d", p.Stock)
	}
	if p.ID != "" {
		t.Fatalf("expected empty ID, got %q", p.ID)
	}
	if p.CreatedAt.Before(before) || p.CreatedAt.After(after) {
		t.Fatalf("CreatedAt %v not in expected range [%v, %v]", p.CreatedAt, before, after)
	}
	if p.UpdatedAt.Before(before) || p.UpdatedAt.After(after) {
		t.Fatalf("UpdatedAt %v not in expected range [%v, %v]", p.UpdatedAt, before, after)
	}
}
