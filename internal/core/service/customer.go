package service

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/logger"
	"github.com/rafaelleal24/challenge/internal/core/port"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

type CustomerService struct {
	customerRepository port.CustomerPort
}

func NewCustomerService(customerRepository port.CustomerPort) *CustomerService {
	return &CustomerService{customerRepository: customerRepository}
}

func (s *CustomerService) Create(ctx context.Context) (domain.ID, error) {
	return s.customerRepository.Create(ctx)
}

func (s *CustomerService) Exists(ctx context.Context, id domain.ID) error {
	_, err := s.customerRepository.Exists(ctx, id)
	if err != nil {
		if serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			return serviceerrors.NewNotFoundError("customer not found")
		}
		logger.Error(ctx, "customer: exists failed", err, map[string]any{
			"customer_id": id,
		})
		return nil
	}

	return nil
}
