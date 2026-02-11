package port

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/core/domain"
)

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock

type OrderPort interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id domain.ID) (*domain.Order, error)
	GetByCustomerID(ctx context.Context, customerID domain.ID, limit, offset int64) ([]*domain.Order, error)
	GetByStatus(ctx context.Context, status domain.OrderStatus, limit, offset int64) ([]*domain.Order, error)
	UpdateStatusWithOutbox(ctx context.Context, id domain.ID, status domain.OrderStatus, event domain.Event) error
	Delete(ctx context.Context, id domain.ID) error
}
