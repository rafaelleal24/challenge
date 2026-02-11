package repository_test

import (
	"context"
	"testing"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/repository"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

func createTestProduct(t *testing.T, repo interface {
	Create(ctx context.Context, product *domain.Product) error
}) *domain.Product {
	t.Helper()
	product := domain.NewProduct("Test Product", "A test description", domain.NewAmountFromCents(2999), 50)
	if err := repo.Create(context.Background(), product); err != nil {
		t.Fatalf("setup: create product failed: %v", err)
	}
	return product
}

func TestProductRepository_Create(t *testing.T) {
	repo := repository.NewProductRepository(testDB)
	ctx := context.Background()

	t.Run("creates product and assigns ID", func(t *testing.T) {
		product := domain.NewProduct("Widget", "A widget", domain.NewAmountFromCents(1500), 100)

		err := repo.Create(ctx, product)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if product.ID == "" {
			t.Fatal("expected product ID to be assigned")
		}
		if len(string(product.ID)) != 24 {
			t.Fatalf("expected 24-char hex ID, got %q", product.ID)
		}
	})
}

func TestProductRepository_GetByID(t *testing.T) {
	repo := repository.NewProductRepository(testDB)
	ctx := context.Background()

	t.Run("returns product by ID", func(t *testing.T) {
		created := createTestProduct(t, repo)

		found, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected id %s, got %s", created.ID, found.ID)
		}
		if found.Name != created.Name {
			t.Fatalf("expected name %q, got %q", created.Name, found.Name)
		}
		if found.Price != created.Price {
			t.Fatalf("expected price %d, got %d", created.Price, found.Price)
		}
		if found.Stock != created.Stock {
			t.Fatalf("expected stock %d, got %d", created.Stock, found.Stock)
		}
	})

	t.Run("returns not found for non-existing ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "aabbccddee112233aabbccdd")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("returns error for invalid ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "bad-id")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindInvalidRequest) {
			t.Fatalf("expected KindInvalidRequest, got %v", err)
		}
	})
}

func TestProductRepository_GetAll(t *testing.T) {
	// Use a fresh database to avoid pollution from other tests
	freshDB := testClient.Database("test_product_getall")
	repo := repository.NewProductRepository(freshDB)
	ctx := context.Background()

	t.Run("returns empty list when no products", func(t *testing.T) {
		products, err := repo.GetAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(products) != 0 {
			t.Fatalf("expected 0 products, got %d", len(products))
		}
	})

	t.Run("returns all created products", func(t *testing.T) {
		p1 := domain.NewProduct("Product 1", "Desc 1", domain.NewAmountFromCents(1000), 10)
		p2 := domain.NewProduct("Product 2", "Desc 2", domain.NewAmountFromCents(2000), 20)
		_ = repo.Create(ctx, p1)
		_ = repo.Create(ctx, p2)

		products, err := repo.GetAll(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(products) != 2 {
			t.Fatalf("expected 2 products, got %d", len(products))
		}
	})
}

func TestProductRepository_DeductStock(t *testing.T) {
	repo := repository.NewProductRepository(testDB)
	ctx := context.Background()

	t.Run("deducts stock successfully", func(t *testing.T) {
		product := domain.NewProduct("Deduct Test", "", domain.NewAmountFromCents(500), 10)
		_ = repo.Create(ctx, product)

		err := repo.DeductStock(ctx, product.ID, 3)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		updated, _ := repo.GetByID(ctx, product.ID)
		if updated.Stock != 7 {
			t.Fatalf("expected stock 7, got %d", updated.Stock)
		}
	})

	t.Run("fails when insufficient stock", func(t *testing.T) {
		product := domain.NewProduct("Low Stock", "", domain.NewAmountFromCents(500), 2)
		_ = repo.Create(ctx, product)

		err := repo.DeductStock(ctx, product.ID, 5)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindUnprocessableEntity) {
			t.Fatalf("expected KindUnprocessableEntity, got %v", err)
		}

		// Stock should remain unchanged
		unchanged, _ := repo.GetByID(ctx, product.ID)
		if unchanged.Stock != 2 {
			t.Fatalf("expected stock 2 (unchanged), got %d", unchanged.Stock)
		}
	})

	t.Run("deducts exact stock to zero", func(t *testing.T) {
		product := domain.NewProduct("Exact Zero", "", domain.NewAmountFromCents(500), 5)
		_ = repo.Create(ctx, product)

		err := repo.DeductStock(ctx, product.ID, 5)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		updated, _ := repo.GetByID(ctx, product.ID)
		if updated.Stock != 0 {
			t.Fatalf("expected stock 0, got %d", updated.Stock)
		}
	})

	t.Run("fails for non-existing product", func(t *testing.T) {
		err := repo.DeductStock(ctx, "aabbccddee112233aabbccdd", 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindUnprocessableEntity) {
			t.Fatalf("expected KindUnprocessableEntity, got %v", err)
		}
	})

	t.Run("fails for invalid ID", func(t *testing.T) {
		err := repo.DeductStock(ctx, "bad-id", 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindInvalidRequest) {
			t.Fatalf("expected KindInvalidRequest, got %v", err)
		}
	})
}
