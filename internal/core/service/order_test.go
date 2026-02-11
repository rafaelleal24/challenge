package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/dto"
	"github.com/rafaelleal24/challenge/internal/core/port/mock"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
	"github.com/rafaelleal24/challenge/internal/core/utils"
	"go.uber.org/mock/gomock"
)

type orderMocks struct {
	orderRepo    *mock.MockOrderPort
	productSvc   *ProductService
	productRepo  *mock.MockProductPort
	customerSvc  *CustomerService
	customerRepo *mock.MockCustomerPort
	orderCache   *mock.MockCachePort[domain.Order]
	idemCache    *mock.MockCachePort[IdempotencyEntry[domain.Order]]
	txManager    *mock.MockTransactionManager
}

func setupOrderService(t *testing.T) (*OrderService, *orderMocks) {
	ctrl := gomock.NewController(t)

	orderRepo := mock.NewMockOrderPort(ctrl)
	productRepo := mock.NewMockProductPort(ctrl)
	customerRepo := mock.NewMockCustomerPort(ctrl)
	orderCache := mock.NewMockCachePort[domain.Order](ctrl)
	idemCache := mock.NewMockCachePort[IdempotencyEntry[domain.Order]](ctrl)
	txManager := mock.NewMockTransactionManager(ctrl)

	productSvc := NewProductService(productRepo)
	customerSvc := NewCustomerService(customerRepo)
	idemSvc := NewIdempotencyService[domain.Order](idemCache, 15*time.Minute, 50*time.Millisecond, 500*time.Millisecond)

	svc := NewOrderService(orderRepo, productSvc, customerSvc, orderCache, idemSvc, txManager)

	return svc, &orderMocks{
		orderRepo:    orderRepo,
		productSvc:   productSvc,
		productRepo:  productRepo,
		customerSvc:  customerSvc,
		customerRepo: customerRepo,
		orderCache:   orderCache,
		idemCache:    idemCache,
		txManager:    txManager,
	}
}

func TestOrderService_GetOrderByID(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		cachedOrder := &domain.Order{
			ID:     orderID,
			Status: domain.OrderStatusCreated,
		}

		m.orderCache.EXPECT().
			Get(gomock.Any(), "order:"+string(orderID)).
			Return(cachedOrder, nil)

		order, err := svc.GetOrderByID(context.Background(), orderID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order.ID != orderID {
			t.Fatalf("expected order id %s, got %s", orderID, order.ID)
		}
	})

	t.Run("cache miss - fetches from repo and caches", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		repoOrder := &domain.Order{
			ID:     orderID,
			Status: domain.OrderStatusCreated,
		}

		m.orderCache.EXPECT().
			Get(gomock.Any(), "order:"+string(orderID)).
			Return(nil, nil)

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(repoOrder, nil)

		m.orderCache.EXPECT().
			Set(gomock.Any(), "order:"+string(orderID), repoOrder, orderCacheTTL).
			Return(nil)

		order, err := svc.GetOrderByID(context.Background(), orderID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order.ID != orderID {
			t.Fatalf("expected order id %s, got %s", orderID, order.ID)
		}
	})

	t.Run("cache error - still fetches from repo", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		repoOrder := &domain.Order{ID: orderID}

		m.orderCache.EXPECT().
			Get(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("redis error"))

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(repoOrder, nil)

		m.orderCache.EXPECT().
			Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		order, err := svc.GetOrderByID(context.Background(), orderID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order == nil {
			t.Fatal("expected order, got nil")
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")

		m.orderCache.EXPECT().
			Get(gomock.Any(), gomock.Any()).
			Return(nil, nil)

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(nil, serviceerrors.NewNotFoundError("order not found"))

		_, err := svc.GetOrderByID(context.Background(), orderID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("cache set error is swallowed", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		repoOrder := &domain.Order{ID: orderID}

		m.orderCache.EXPECT().
			Get(gomock.Any(), gomock.Any()).
			Return(nil, nil)

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(repoOrder, nil)

		m.orderCache.EXPECT().
			Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("cache set failed"))

		order, err := svc.GetOrderByID(context.Background(), orderID)
		if err != nil {
			t.Fatalf("expected no error (cache set failure is non-fatal), got %v", err)
		}
		if order == nil {
			t.Fatal("expected order, got nil")
		}
	})
}

// --- UpdateOrderStatus ---

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		customerID := domain.ID("ccddaabbee112233aabbccdd")
		existingOrder := &domain.Order{
			ID:         orderID,
			CustomerID: customerID,
			Status:     domain.OrderStatusCreated,
			Items:      []domain.OrderItem{},
		}

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(existingOrder, nil)

		m.orderRepo.EXPECT().
			UpdateStatusWithOutbox(gomock.Any(), orderID, domain.OrderStatusProcessing, gomock.Any()).
			Return(nil)

		m.orderCache.EXPECT().
			Set(gomock.Any(), "order:"+string(orderID), gomock.Any(), orderCacheTTL).
			Return(nil)

		err := svc.UpdateOrderStatus(context.Background(), orderID, domain.OrderStatusProcessing)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		svc, _ := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")

		err := svc.UpdateOrderStatus(context.Background(), orderID, domain.OrderStatus("invalid"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindInvalidRequest) {
			t.Fatalf("expected KindInvalidRequest, got %v", err)
		}
	})

	t.Run("same status", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		existingOrder := &domain.Order{
			ID:     orderID,
			Status: domain.OrderStatusProcessing,
		}

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(existingOrder, nil)

		err := svc.UpdateOrderStatus(context.Background(), orderID, domain.OrderStatusProcessing)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindUnprocessableEntity) {
			t.Fatalf("expected KindUnprocessableEntity, got %v", err)
		}
	})

	t.Run("order not found", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(nil, serviceerrors.NewNotFoundError("order not found"))

		err := svc.UpdateOrderStatus(context.Background(), orderID, domain.OrderStatusProcessing)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("update repo error", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		existingOrder := &domain.Order{
			ID:     orderID,
			Status: domain.OrderStatusCreated,
		}

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(existingOrder, nil)

		m.orderRepo.EXPECT().
			UpdateStatusWithOutbox(gomock.Any(), orderID, domain.OrderStatusProcessing, gomock.Any()).
			Return(errors.New("db error"))

		err := svc.UpdateOrderStatus(context.Background(), orderID, domain.OrderStatusProcessing)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("cache set error is swallowed on update", func(t *testing.T) {
		svc, m := setupOrderService(t)
		orderID := domain.ID("aabbccddee112233aabbccdd")
		existingOrder := &domain.Order{
			ID:     orderID,
			Status: domain.OrderStatusCreated,
		}

		m.orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(existingOrder, nil)

		m.orderRepo.EXPECT().
			UpdateStatusWithOutbox(gomock.Any(), orderID, domain.OrderStatusShipped, gomock.Any()).
			Return(nil)

		m.orderCache.EXPECT().
			Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("cache error"))

		err := svc.UpdateOrderStatus(context.Background(), orderID, domain.OrderStatusShipped)
		if err != nil {
			t.Fatalf("expected no error (cache failure non-fatal), got %v", err)
		}
	})
}

// --- CreateOrder (processOrder) ---

func TestOrderService_CreateOrder(t *testing.T) {
	customerID := domain.ID("ccddaabbee112233aabbccdd")
	productID := domain.ID("aabbccddee112233aabbccd1")

	validRequest := &dto.CreateOrderRequest{
		CustomerID: customerID,
		Items: []dto.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	product := &domain.Product{
		ID:    productID,
		Name:  "Test Product",
		Price: domain.Amount(2999),
		Stock: 50,
	}

	t.Run("success without idempotency key", func(t *testing.T) {
		svc, m := setupOrderService(t)

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(true, nil)

		m.productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(product, nil)

		m.txManager.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		m.productRepo.EXPECT().
			DeductStock(gomock.Any(), productID, 2).
			Return(nil)

		m.orderRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, order *domain.Order) error {
				order.ID = domain.ID("aabbccddee112233aabbccdd")
				return nil
			})

		order, err := svc.CreateOrder(context.Background(), "", validRequest)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order == nil {
			t.Fatal("expected order, got nil")
		}
		if order.CustomerID != customerID {
			t.Fatalf("expected customer id %s, got %s", customerID, order.CustomerID)
		}
		if len(order.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(order.Items))
		}
		expectedTotal := domain.Amount(2999).Multiply(2)
		if order.TotalAmount != expectedTotal {
			t.Fatalf("expected total %d, got %d", expectedTotal, order.TotalAmount)
		}
	})

	t.Run("too many items", func(t *testing.T) {
		svc, _ := setupOrderService(t)

		items := make([]dto.OrderItem, ORDER_MAX_ITEMS+1)
		for i := range items {
			items[i] = dto.OrderItem{ProductID: productID, Quantity: 1}
		}
		req := &dto.CreateOrderRequest{
			CustomerID: customerID,
			Items:      items,
		}

		_, err := svc.CreateOrder(context.Background(), "", req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindUnprocessableEntity) {
			t.Fatalf("expected KindUnprocessableEntity, got %v", err)
		}
	})

	t.Run("customer not found", func(t *testing.T) {
		svc, m := setupOrderService(t)

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(false, serviceerrors.NewNotFoundError("entity not found"))

		_, err := svc.CreateOrder(context.Background(), "", validRequest)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("product not found", func(t *testing.T) {
		svc, m := setupOrderService(t)

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(true, nil)

		m.productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(nil, serviceerrors.NewNotFoundError("product not found"))

		_, err := svc.CreateOrder(context.Background(), "", validRequest)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("deduct stock fails inside transaction", func(t *testing.T) {
		svc, m := setupOrderService(t)

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(true, nil)

		m.productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(product, nil)

		m.txManager.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		m.productRepo.EXPECT().
			DeductStock(gomock.Any(), productID, 2).
			Return(errors.New("insufficient stock"))

		_, err := svc.CreateOrder(context.Background(), "", validRequest)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("order repo create fails inside transaction", func(t *testing.T) {
		svc, m := setupOrderService(t)

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(true, nil)

		m.productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(product, nil)

		m.txManager.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		m.productRepo.EXPECT().
			DeductStock(gomock.Any(), productID, 2).
			Return(nil)

		m.orderRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(errors.New("insert failed"))

		_, err := svc.CreateOrder(context.Background(), "", validRequest)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("multiple items - calculates total correctly", func(t *testing.T) {
		svc, m := setupOrderService(t)
		productID2 := domain.ID("aabbccddee112233aabbccd2")
		product2 := &domain.Product{
			ID:    productID2,
			Name:  "Product 2",
			Price: domain.Amount(1500),
			Stock: 100,
		}

		multiItemReq := &dto.CreateOrderRequest{
			CustomerID: customerID,
			Items: []dto.OrderItem{
				{ProductID: productID, Quantity: 2},
				{ProductID: productID2, Quantity: 3},
			},
		}

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(true, nil)

		m.productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(product, nil)
		m.productRepo.EXPECT().
			GetByID(gomock.Any(), productID2).
			Return(product2, nil)

		m.txManager.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		m.productRepo.EXPECT().
			DeductStock(gomock.Any(), productID, 2).
			Return(nil)
		m.productRepo.EXPECT().
			DeductStock(gomock.Any(), productID2, 3).
			Return(nil)

		m.orderRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, order *domain.Order) error {
				order.ID = domain.ID("aabbccddee112233aabbccdd")
				return nil
			})

		order, err := svc.CreateOrder(context.Background(), "", multiItemReq)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// 2999*2 + 1500*3 = 5998 + 4500 = 10498
		expectedTotal := domain.Amount(10498)
		if order.TotalAmount != expectedTotal {
			t.Fatalf("expected total %d, got %d", expectedTotal, order.TotalAmount)
		}
		if len(order.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(order.Items))
		}
	})
}

func TestOrderService_CreateOrder_Idempotency(t *testing.T) {
	customerID := domain.ID("ccddaabbee112233aabbccdd")
	productID := domain.ID("aabbccddee112233aabbccd1")

	validRequest := &dto.CreateOrderRequest{
		CustomerID: customerID,
		Items: []dto.OrderItem{
			{ProductID: productID, Quantity: 1},
		},
	}

	product := &domain.Product{
		ID:    productID,
		Name:  "Test Product",
		Price: domain.Amount(2999),
		Stock: 50,
	}

	t.Run("first request with idempotency key", func(t *testing.T) {
		svc, m := setupOrderService(t)
		idemKey := "idem-123"

		m.idemCache.EXPECT().
			SetNX(gomock.Any(), idemKey, gomock.Any(), 15*time.Minute).
			Return(true, nil)

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(true, nil)
		m.productRepo.EXPECT().
			GetByID(gomock.Any(), productID).
			Return(product, nil)
		m.txManager.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})
		m.productRepo.EXPECT().
			DeductStock(gomock.Any(), productID, 1).
			Return(nil)
		m.orderRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, order *domain.Order) error {
				order.ID = domain.ID("aabbccddee112233aabbccdd")
				return nil
			})

		m.idemCache.EXPECT().
			Set(gomock.Any(), idemKey, gomock.Any(), 15*time.Minute).
			Return(nil)

		order, err := svc.CreateOrder(context.Background(), idemKey, validRequest)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order == nil {
			t.Fatal("expected order, got nil")
		}
	})

	t.Run("duplicate idempotency key - returns cached order", func(t *testing.T) {
		svc, m := setupOrderService(t)
		idemKey := "idem-123"
		cachedOrder := &domain.Order{
			ID:         domain.ID("aabbccddee112233aabbccdd"),
			CustomerID: customerID,
			Status:     domain.OrderStatusCreated,
		}

		m.idemCache.EXPECT().
			SetNX(gomock.Any(), idemKey, gomock.Any(), 15*time.Minute).
			Return(false, nil)

		m.idemCache.EXPECT().
			Get(gomock.Any(), idemKey).
			Return(&IdempotencyEntry[domain.Order]{
				Status:      IdempotencyCompleted,
				PayloadHash: utils.HashJSON(validRequest),
				Result:      cachedOrder,
			}, nil)

		order, err := svc.CreateOrder(context.Background(), idemKey, validRequest)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order == nil {
			t.Fatal("expected order, got nil")
		}
		if order.ID != cachedOrder.ID {
			t.Fatalf("expected order id %s, got %s", cachedOrder.ID, order.ID)
		}
	})

	t.Run("idempotency claim error", func(t *testing.T) {
		svc, m := setupOrderService(t)
		idemKey := "idem-123"

		m.idemCache.EXPECT().
			SetNX(gomock.Any(), idemKey, gomock.Any(), 15*time.Minute).
			Return(false, errors.New("redis down"))

		_, err := svc.CreateOrder(context.Background(), idemKey, validRequest)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("processOrder fails - releases idempotency key", func(t *testing.T) {
		svc, m := setupOrderService(t)
		idemKey := "idem-123"

		m.idemCache.EXPECT().
			SetNX(gomock.Any(), idemKey, gomock.Any(), 15*time.Minute).
			Return(true, nil)

		m.customerRepo.EXPECT().
			Exists(gomock.Any(), customerID).
			Return(false, serviceerrors.NewNotFoundError("entity not found"))

		m.idemCache.EXPECT().
			Del(gomock.Any(), idemKey).
			Return(nil)

		_, err := svc.CreateOrder(context.Background(), idemKey, validRequest)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !serviceerrors.IsOfKind(err, serviceerrors.KindNotFound) {
			t.Fatalf("expected KindNotFound, got %v", err)
		}
	})
}
