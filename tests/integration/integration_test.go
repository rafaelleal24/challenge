package integration_test

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	adaptconfig "github.com/rafaelleal24/challenge/internal/adapters/config"
	adaptmongo "github.com/rafaelleal24/challenge/internal/adapters/mongo"
	"github.com/rafaelleal24/challenge/internal/adapters/mongo/repository"
	"github.com/rafaelleal24/challenge/internal/adapters/outbox"
	adaptrabbitmq "github.com/rafaelleal24/challenge/internal/adapters/rabbitmq"
	adaptredis "github.com/rafaelleal24/challenge/internal/adapters/redis"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/dto"
	"github.com/rafaelleal24/challenge/internal/core/service"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcrabbit "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient  *mongo.Client
	redisClient  *adaptredis.Client
	broker       *adaptrabbitmq.RabbitMQAdapter
	amqpEndpoint string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	mongoContainer, err := mongodb.Run(ctx, "mongo:7", mongodb.WithReplicaSet("rs0"))
	if err != nil {
		log.Fatalf("mongodb container: %v", err)
	}
	mongoEndpoint, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("mongodb connection string: %v", err)
	}
	mongoClient, err = mongo.Connect(ctx, options.Client().
		ApplyURI(mongoEndpoint).
		SetDirect(true).
		SetConnectTimeout(30*time.Second).
		SetServerSelectionTimeout(30*time.Second))
	if err != nil {
		log.Fatalf("mongodb connect: %v", err)
	}
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("mongodb ping: %v", err)
	}

	// --- Redis ---
	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		log.Fatalf("redis container: %v", err)
	}
	redisEndpoint, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("redis connection string: %v", err)
	}
	redisClient, err = adaptredis.NewConnection(adaptconfig.RedisConfig{URL: redisEndpoint})
	if err != nil {
		log.Fatalf("redis connect: %v", err)
	}

	// --- RabbitMQ ---
	rabbitContainer, err := tcrabbit.Run(ctx, "rabbitmq:3-management-alpine")
	if err != nil {
		log.Fatalf("rabbitmq container: %v", err)
	}
	amqpEndpoint, err = rabbitContainer.AmqpURL(ctx)
	if err != nil {
		log.Fatalf("rabbitmq amqp url: %v", err)
	}
	broker, err = adaptrabbitmq.NewRabbitMQAdapter(adaptconfig.RabbitMQConfig{
		URL:        amqpEndpoint,
		MaxRetries: 2,
		RetryDelay: 100 * time.Millisecond,
		ExchangeConfigs: []adaptconfig.ExchangeConfig{
			{Name: "exchange.order", Type: "direct", Durable: true, AutoDelete: false},
		},
	})
	if err != nil {
		log.Fatalf("rabbitmq adapter: %v", err)
	}

	code := m.Run()

	_ = broker.Close()
	_ = redisClient.Close()
	_ = mongoClient.Disconnect(ctx)
	_ = mongoContainer.Terminate(ctx)
	_ = redisContainer.Terminate(ctx)
	_ = rabbitContainer.Terminate(ctx)

	os.Exit(code)
}

func setupConsumer(t *testing.T, routingKey string) <-chan amqp.Delivery {
	t.Helper()

	conn, err := amqp.Dial(amqpEndpoint)
	if err != nil {
		t.Fatalf("consumer dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("consumer channel: %v", err)
	}
	t.Cleanup(func() { ch.Close() })

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatalf("queue declare: %v", err)
	}
	if err := ch.QueueBind(q.Name, routingKey, "exchange.order", false, nil); err != nil {
		t.Fatalf("queue bind: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	return msgs
}

func buildServices(t *testing.T, dbName string) (
	*service.OrderService,
	*service.ProductService,
	*service.CustomerService,
	*outbox.Handler,
) {
	t.Helper()
	db := mongoClient.Database(dbName)

	outboxRepo := repository.NewOutboxRepository(db)
	orderRepo := repository.NewOrderRepository(db, outboxRepo)
	productRepo := repository.NewProductRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	txManager := adaptmongo.NewTransactionManager(mongoClient)

	customerService := service.NewCustomerService(customerRepo)
	productService := service.NewProductService(productRepo)

	orderCache := adaptredis.NewCache[domain.Order](redisClient, dbName+"-order")
	idempotencyCache := adaptredis.NewCache[service.IdempotencyEntry[domain.Order]](redisClient, dbName+"-idemp")
	idempotencyService := service.NewIdempotencyService(idempotencyCache, 5*time.Minute, 500*time.Millisecond, 10*time.Second)

	orderService := service.NewOrderService(orderRepo, productService, customerService, orderCache, idempotencyService, txManager)

	outboxHandler := outbox.NewHandler(outboxRepo, broker, adaptconfig.OutboxConfig{
		Interval:  100 * time.Millisecond,
		BatchSize: 50,
	})

	return orderService, productService, customerService, outboxHandler
}

func TestIntegration_CreateOrder_FullCycle(t *testing.T) {
	msgs := setupConsumer(t, "order.update_status")

	orderSvc, productSvc, customerSvc, outboxHandler := buildServices(t, "int_full_cycle")
	ctx := context.Background()

	handlerCtx, cancelHandler := context.WithCancel(ctx)
	defer cancelHandler()
	go outboxHandler.Start(handlerCtx)

	customerID, err := customerSvc.Create(ctx)
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	product, err := productSvc.CreateProduct(ctx, &dto.CreateProductRequest{
		Name: "Integration Widget", Description: "e2e", Price: 2999, Stock: 50,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	order, err := orderSvc.CreateOrder(ctx, "", &dto.CreateOrderRequest{
		CustomerID: customerID,
		Items:      []dto.OrderItem{{ProductID: product.ID, Quantity: 3}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.ID == "" {
		t.Fatal("order ID should not be empty")
	}
	if order.Status != domain.OrderStatusCreated {
		t.Fatalf("expected status 'created', got %q", order.Status)
	}
	if expected := domain.Amount(2999 * 3); order.TotalAmount != expected {
		t.Fatalf("expected total %d, got %d", expected, order.TotalAmount)
	}

	productAfter, _ := productSvc.GetByID(ctx, product.ID)
	if productAfter.Stock != 47 {
		t.Fatalf("expected stock 47, got %d", productAfter.Stock)
	}

	if err := orderSvc.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusProcessing); err != nil {
		t.Fatalf("update status: %v", err)
	}

	select {
	case msg := <-msgs:
		var event domain.OrderUpdateStatusEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if event.OrderID != order.ID {
			t.Fatalf("event order_id: expected %s, got %s", order.ID, event.OrderID)
		}
		if event.Status != domain.OrderStatusProcessing {
			t.Fatalf("event status: expected 'processing', got %q", event.Status)
		}
		if event.OldStatus != domain.OrderStatusCreated {
			t.Fatalf("event old_status: expected 'created', got %q", event.OldStatus)
		}
		if event.CustomerID != customerID {
			t.Fatalf("event customer_id: expected %s, got %s", customerID, event.CustomerID)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for order.update_status event")
	}

	fetched, _ := orderSvc.GetOrderByID(ctx, order.ID)
	if fetched.Status != domain.OrderStatusProcessing {
		t.Fatalf("expected fetched status 'processing', got %q", fetched.Status)
	}
}

func TestIntegration_CreateOrder_Idempotency(t *testing.T) {
	orderSvc, productSvc, customerSvc, _ := buildServices(t, "int_idempotency")
	ctx := context.Background()

	customerID, _ := customerSvc.Create(ctx)
	product, _ := productSvc.CreateProduct(ctx, &dto.CreateProductRequest{
		Name: "Idemp Widget", Description: "test", Price: 1000, Stock: 100,
	})

	request := &dto.CreateOrderRequest{
		CustomerID: customerID,
		Items:      []dto.OrderItem{{ProductID: product.ID, Quantity: 2}},
	}

	order1, err := orderSvc.CreateOrder(ctx, "idemp-key-1", request)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	order2, err := orderSvc.CreateOrder(ctx, "idemp-key-1", request)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if order2.ID != order1.ID {
		t.Fatalf("expected same order: %s vs %s", order1.ID, order2.ID)
	}

	// Stock deducted only once
	p, _ := productSvc.GetByID(ctx, product.ID)
	if p.Stock != 98 {
		t.Fatalf("expected stock 98 (single deduction), got %d", p.Stock)
	}
}

func TestIntegration_CreateOrder_InsufficientStock(t *testing.T) {
	orderSvc, productSvc, customerSvc, _ := buildServices(t, "int_low_stock")
	ctx := context.Background()

	customerID, _ := customerSvc.Create(ctx)
	product, _ := productSvc.CreateProduct(ctx, &dto.CreateProductRequest{
		Name: "Low Stock", Description: "test", Price: 500, Stock: 2,
	})

	_, err := orderSvc.CreateOrder(ctx, "", &dto.CreateOrderRequest{
		CustomerID: customerID,
		Items:      []dto.OrderItem{{ProductID: product.ID, Quantity: 5}},
	})
	if err == nil {
		t.Fatal("expected insufficient stock error")
	}

	unchanged, _ := productSvc.GetByID(ctx, product.ID)
	if unchanged.Stock != 2 {
		t.Fatalf("stock should be unchanged after rollback: expected 2, got %d", unchanged.Stock)
	}
}

func TestIntegration_CreateOrder_InvalidCustomer(t *testing.T) {
	orderSvc, productSvc, _, _ := buildServices(t, "int_bad_customer")
	ctx := context.Background()

	product, _ := productSvc.CreateProduct(ctx, &dto.CreateProductRequest{
		Name: "Widget", Description: "test", Price: 500, Stock: 10,
	})

	_, err := orderSvc.CreateOrder(ctx, "", &dto.CreateOrderRequest{
		CustomerID: "aabbccddee112233aabbccdd",
		Items:      []dto.OrderItem{{ProductID: product.ID, Quantity: 1}},
	})
	if err == nil {
		t.Fatal("expected error for non-existing customer")
	}
}

func TestIntegration_GetOrderByID_Cache(t *testing.T) {
	orderSvc, productSvc, customerSvc, _ := buildServices(t, "int_cache")
	ctx := context.Background()

	customerID, _ := customerSvc.Create(ctx)
	product, _ := productSvc.CreateProduct(ctx, &dto.CreateProductRequest{
		Name: "Cache Widget", Description: "test", Price: 1500, Stock: 20,
	})

	order, _ := orderSvc.CreateOrder(ctx, "", &dto.CreateOrderRequest{
		CustomerID: customerID,
		Items:      []dto.OrderItem{{ProductID: product.ID, Quantity: 1}},
	})

	f1, err := orderSvc.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("first get: %v", err)
	}

	// Second fetch → cache hit
	f2, err := orderSvc.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("second get: %v", err)
	}

	if f1.ID != f2.ID || f1.TotalAmount != f2.TotalAmount {
		t.Fatal("cached order should match original")
	}
}

func TestIntegration_MultipleStatusUpdates(t *testing.T) {
	msgs := setupConsumer(t, "order.update_status")

	orderSvc, productSvc, customerSvc, outboxHandler := buildServices(t, "int_multi_status")
	ctx := context.Background()

	handlerCtx, cancelHandler := context.WithCancel(ctx)
	defer cancelHandler()
	go outboxHandler.Start(handlerCtx)

	customerID, _ := customerSvc.Create(ctx)
	product, _ := productSvc.CreateProduct(ctx, &dto.CreateProductRequest{
		Name: "Multi Widget", Description: "test", Price: 1000, Stock: 10,
	})
	order, _ := orderSvc.CreateOrder(ctx, "", &dto.CreateOrderRequest{
		CustomerID: customerID,
		Items:      []dto.OrderItem{{ProductID: product.ID, Quantity: 1}},
	})

	transitions := []domain.OrderStatus{domain.OrderStatusProcessing, domain.OrderStatusShipped}
	for _, status := range transitions {
		if err := orderSvc.UpdateOrderStatus(ctx, order.ID, status); err != nil {
			t.Fatalf("update to %q: %v", status, err)
		}

		select {
		case msg := <-msgs:
			var event domain.OrderUpdateStatusEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if event.Status != status {
				t.Fatalf("expected status %q in event, got %q", status, event.Status)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for event with status %q", status)
		}
	}

	final, _ := orderSvc.GetOrderByID(ctx, order.ID)
	if final.Status != domain.OrderStatusShipped {
		t.Fatalf("expected final status 'shipped', got %q", final.Status)
	}
}
