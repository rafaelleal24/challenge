package service

import (
	"context"
	"errors"
	"testing"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/dto"
	"github.com/rafaelleal24/challenge/internal/core/port/mock"
	"go.uber.org/mock/gomock"
)

func setupProductService(t *testing.T) (*ProductService, *mock.MockProductPort) {
	ctrl := gomock.NewController(t)
	productRepo := mock.NewMockProductPort(ctrl)
	svc := NewProductService(productRepo)
	return svc, productRepo
}

func TestProductService_CreateProduct(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, productRepo := setupProductService(t)
		req := &dto.CreateProductRequest{
			Name:        "Test Product",
			Description: "A test product",
			Price:       2999,
			Stock:       50,
		}

		productRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p *domain.Product) error {
				p.ID = domain.ID("aabbccddee112233aabbccdd")
				return nil
			})

		product, err := svc.CreateProduct(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if product == nil {
			t.Fatal("expected product, got nil")
		}
		if product.Name != req.Name {
			t.Fatalf("expected name %q, got %q", req.Name, product.Name)
		}
		if product.Description != req.Description {
			t.Fatalf("expected description %q, got %q", req.Description, product.Description)
		}
		if int(product.Price) != req.Price {
			t.Fatalf("expected price %d, got %d", req.Price, product.Price)
		}
		if product.Stock != req.Stock {
			t.Fatalf("expected stock %d, got %d", req.Stock, product.Stock)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		svc, productRepo := setupProductService(t)
		req := &dto.CreateProductRequest{
			Name:  "Test Product",
			Price: 2999,
			Stock: 10,
		}

		productRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(errors.New("insert failed"))

		product, err := svc.CreateProduct(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if product != nil {
			t.Fatal("expected nil product on error")
		}
	})
}

func TestProductService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, productRepo := setupProductService(t)
		productID := domain.ID("aabbccddee112233aabbccdd")
		expected := &domain.Product{
			ID:    productID,
			Name:  "Test Product",
			Price: domain.Amount(2999),
			Stock: 50,
		}

		productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(expected, nil)

		product, err := svc.GetByID(context.Background(), productID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if product.ID != expected.ID {
			t.Fatalf("expected product id %s, got %s", expected.ID, product.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, productRepo := setupProductService(t)
		productID := domain.ID("aabbccddee112233aabbccdd")

		productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(nil, errors.New("not found"))

		product, err := svc.GetByID(context.Background(), productID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if product != nil {
			t.Fatal("expected nil product")
		}
	})
}

func TestProductService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, productRepo := setupProductService(t)
		expected := []*domain.Product{
			{ID: domain.ID("aabbccddee112233aabbccd1"), Name: "Product 1"},
			{ID: domain.ID("aabbccddee112233aabbccd2"), Name: "Product 2"},
		}

		productRepo.EXPECT().
			GetAll(gomock.Any()).
			Return(expected, nil)

		products, err := svc.GetAll(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(products) != 2 {
			t.Fatalf("expected 2 products, got %d", len(products))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		svc, productRepo := setupProductService(t)

		productRepo.EXPECT().
			GetAll(gomock.Any()).
			Return([]*domain.Product{}, nil)

		products, err := svc.GetAll(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(products) != 0 {
			t.Fatalf("expected 0 products, got %d", len(products))
		}
	})

	t.Run("repository error", func(t *testing.T) {
		svc, productRepo := setupProductService(t)

		productRepo.EXPECT().
			GetAll(gomock.Any()).
			Return(nil, errors.New("db error"))

		_, err := svc.GetAll(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestProductService_DeductStock(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, productRepo := setupProductService(t)
		productID := domain.ID("aabbccddee112233aabbccdd")

		productRepo.EXPECT().
			DeductStock(gomock.Any(), productID, 5).
			Return(nil)

		err := svc.DeductStock(context.Background(), productID, 5)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("insufficient stock", func(t *testing.T) {
		svc, productRepo := setupProductService(t)
		productID := domain.ID("aabbccddee112233aabbccdd")

		productRepo.EXPECT().
			DeductStock(gomock.Any(), productID, 999).
			Return(errors.New("insufficient stock"))

		err := svc.DeductStock(context.Background(), productID, 999)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
