package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/dto"
	"github.com/rafaelleal24/challenge/internal/core/logger"
	"github.com/rafaelleal24/challenge/internal/core/port"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
	"github.com/rafaelleal24/challenge/internal/core/utils"
)

const (
	ORDER_MAX_ITEMS = 100
	orderCacheTTL   = 15 * time.Minute
)

type OrderService struct {
	orderRepository port.OrderPort
	productService  *ProductService
	customerService *CustomerService
	orderCache      port.CachePort[domain.Order]
	idempotency     *IdempotencyService[domain.Order]
	txManager       port.TransactionManager
}

func (s *OrderService) getCacheKey(orderID domain.ID) string {
	return fmt.Sprintf("order:%s", orderID)
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID domain.ID) (*domain.Order, error) {
	cached, err := s.orderCache.Get(ctx, s.getCacheKey(orderID))
	if err != nil {
		logger.Error(ctx, "cache: get order failed", err, map[string]any{
			"order_id": orderID,
		})
	}
	if cached != nil {
		logger.Info(ctx, "order found in cache", map[string]any{
			"order_id": orderID,
		})
		return cached, nil
	}

	order, err := s.orderRepository.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := s.orderCache.Set(ctx, s.getCacheKey(orderID), order, orderCacheTTL); err != nil {
		logger.Error(ctx, "cache: set order failed", err, map[string]any{
			"order_id": orderID,
		})
	}

	return order, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID domain.ID, status domain.OrderStatus) error {
	if !status.IsValid() {
		return serviceerrors.NewInvalidRequestError("invalid status")
	}

	order, err := s.orderRepository.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == status {
		return serviceerrors.NewUnprocessableEntityError("order already has this status")
	}

	event := domain.NewOrderUpdateStatusEvent(orderID, status, order.Status, time.Now(), order.CustomerID)
	if err := s.orderRepository.UpdateStatusWithOutbox(ctx, orderID, status, event); err != nil {
		return err
	}

	order.Status = status
	order.UpdatedAt = time.Now()
	if err := s.orderCache.Set(ctx, s.getCacheKey(orderID), order, orderCacheTTL); err != nil {
		logger.Error(ctx, "cache: update order failed", err, map[string]any{
			"order_id": orderID,
		})
	}

	logger.Info(ctx, "Order status updated", map[string]any{
		"order_id":   orderID,
		"old_status": order.Status,
		"new_status": status,
	})

	return nil
}

func (s *OrderService) getOrderItems(ctx context.Context, dtoItems []dto.OrderItem) ([]domain.OrderItem, error) {
	items := make([]domain.OrderItem, len(dtoItems))
	for i, item := range dtoItems {
		product, err := s.productService.GetByID(ctx, item.ProductID)
		if err != nil {
			return nil, err
		}
		items[i] = *domain.NewOrderItem(item.ProductID, product.Name, item.Quantity, product.Price)
	}
	return items, nil
}

func (s *OrderService) processOrder(ctx context.Context, request *dto.CreateOrderRequest) (*domain.Order, error) {
	if len(request.Items) > ORDER_MAX_ITEMS {
		return nil, serviceerrors.NewUnprocessableEntityError("order items limit exceeded")
	}

	if err := s.customerService.Exists(ctx, request.CustomerID); err != nil {
		return nil, err
	}

	items, err := s.getOrderItems(ctx, request.Items)
	if err != nil {
		return nil, err
	}

	order := domain.NewOrder(request.CustomerID, domain.OrderStatusCreated, items)

	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, item := range order.Items {
			if err := s.productService.DeductStock(txCtx, item.ProductID, item.Quantity); err != nil {
				return err
			}
		}
		return s.orderRepository.Create(txCtx, order)
	})
	if err != nil {
		logger.Error(ctx, "transaction: create order failed", err, map[string]any{
			"order_id": order.ID,
		})
		return nil, err
	}

	logger.Info(ctx, "Order created successfully", map[string]any{
		"order_id": order.ID,
	})
	return order, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, idempotencyKey string, request *dto.CreateOrderRequest) (*domain.Order, error) {
	if idempotencyKey == "" {
		return s.processOrder(ctx, request)
	}

	payloadHash := utils.HashJSON(request)

	existing, err := s.idempotency.Claim(ctx, idempotencyKey, payloadHash)
	if err != nil {
		logger.Error(ctx, "idempotency: claim failed", err, map[string]any{
			"idempotency_key": idempotencyKey,
		})
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	order, err := s.processOrder(ctx, request)
	if err != nil {
		s.idempotency.Release(ctx, idempotencyKey)
		logger.Error(ctx, "idempotency: release failed", err, map[string]any{
			"idempotency_key": idempotencyKey,
		})
		return nil, err
	}

	s.idempotency.Complete(ctx, idempotencyKey, payloadHash, order)

	return order, nil
}

func NewOrderService(
	orderRepository port.OrderPort,
	productService *ProductService,
	customerService *CustomerService,
	orderCache port.CachePort[domain.Order],
	idempotency *IdempotencyService[domain.Order],
	txManager port.TransactionManager,
) *OrderService {
	return &OrderService{
		orderRepository: orderRepository,
		productService:  productService,
		customerService: customerService,
		orderCache:      orderCache,
		idempotency:     idempotency,
		txManager:       txManager,
	}
}
