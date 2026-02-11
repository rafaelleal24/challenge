package service

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/dto"
	"github.com/rafaelleal24/challenge/internal/core/logger"
	"github.com/rafaelleal24/challenge/internal/core/port"
)

type ProductService struct {
	productRepository port.ProductPort
}

func NewProductService(productRepository port.ProductPort) *ProductService {
	return &ProductService{productRepository: productRepository}
}

func (s *ProductService) CreateProduct(ctx context.Context, request *dto.CreateProductRequest) (*domain.Product, error) {
	product := domain.NewProduct(request.Name, request.Description, domain.NewAmountFromCents(request.Price), request.Stock)

	if err := s.productRepository.Create(ctx, product); err != nil {
		logger.Error(ctx, "product: create failed", err, map[string]any{
			"name":        request.Name,
			"description": request.Description,
			"price":       request.Price,
			"stock":       request.Stock,
		})
		return nil, err
	}

	logger.Info(ctx, "Product created", map[string]any{"product_id": product.ID})
	return product, nil
}

func (s *ProductService) GetByID(ctx context.Context, id domain.ID) (*domain.Product, error) {
	return s.productRepository.GetByID(ctx, id)
}

func (s *ProductService) GetAll(ctx context.Context) ([]*domain.Product, error) {
	return s.productRepository.GetAll(ctx)
}

func (s *ProductService) DeductStock(ctx context.Context, id domain.ID, quantity int) error {
	return s.productRepository.DeductStock(ctx, id, quantity)
}
