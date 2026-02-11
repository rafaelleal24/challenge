package service

import (
	"context"
	"errors"
	"testing"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/port/mock"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
	"go.uber.org/mock/gomock"
)

func setupCustomerService(t *testing.T) (*CustomerService, *mock.MockCustomerPort) {
	ctrl := gomock.NewController(t)
	customerRepo := mock.NewMockCustomerPort(ctrl)
	svc := NewCustomerService(customerRepo)
	return svc, customerRepo
}

func TestCustomerService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, customerRepo := setupCustomerService(t)
		expectedID := domain.ID("aabbccddee112233aabbccdd")

		customerRepo.EXPECT().
			Create(gomock.Any()).
			Return(expectedID, nil)

		id, err := svc.Create(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != expectedID {
			t.Fatalf("expected id %s, got %s", expectedID, id)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		svc, customerRepo := setupCustomerService(t)
		repoErr := errors.New("db connection failed")

		customerRepo.EXPECT().
			Create(gomock.Any()).
			Return(domain.ID(""), repoErr)

		_, err := svc.Create(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected %v, got %v", repoErr, err)
		}
	})
}

func TestCustomerService_Exists(t *testing.T) {
	t.Run("customer exists", func(t *testing.T) {
		svc, customerRepo := setupCustomerService(t)
		customerID := domain.ID("aabbccddee112233aabbccdd")

		customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(true, nil)

		err := svc.Exists(context.Background(), customerID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("customer not found", func(t *testing.T) {
		svc, customerRepo := setupCustomerService(t)
		customerID := domain.ID("aabbccddee112233aabbccdd")

		customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(false, serviceerrors.NewNotFoundError("entity not found"))

		err := svc.Exists(context.Background(), customerID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("repository error returns nil (logs and swallows)", func(t *testing.T) {
		svc, customerRepo := setupCustomerService(t)
		customerID := domain.ID("aabbccddee112233aabbccdd")

		customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(false, errors.New("unexpected db error"))

		err := svc.Exists(context.Background(), customerID)
		if err != nil {
			t.Fatalf("expected nil (swallowed error), got %v", err)
		}
	})
}
