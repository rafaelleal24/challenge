package port

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/core/domain"
)

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock

type ProductPort interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id domain.ID) (*domain.Product, error)
	GetAll(ctx context.Context) ([]*domain.Product, error)
	DeductStock(ctx context.Context, id domain.ID, quantity int) error
}
