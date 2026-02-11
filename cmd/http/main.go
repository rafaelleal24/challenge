package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rafaelleal24/challenge/internal/adapters/config"
	"github.com/rafaelleal24/challenge/internal/adapters/http"
	"github.com/rafaelleal24/challenge/internal/adapters/http/controllers"
	"github.com/rafaelleal24/challenge/internal/adapters/mongo"
	"github.com/rafaelleal24/challenge/internal/adapters/mongo/repository"
	"github.com/rafaelleal24/challenge/internal/adapters/outbox"
	"github.com/rafaelleal24/challenge/internal/adapters/rabbitmq"
	"github.com/rafaelleal24/challenge/internal/adapters/redis"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/logger"
	"github.com/rafaelleal24/challenge/internal/core/service"
)

// @title       Challenge API
// @version     1.0
// @description Order management API

// @host     localhost:8080
// @BasePath /

//go:generate swag init -d ../.. -g cmd/http/main.go -o ../../docs --parseInternal

func main() {
	// initialize config and logger
	cfg := config.NewConfig()
	if err := logger.Initialize(cfg.Logger.Endpoint, cfg.Logger.ServiceName, cfg.Logger.IsProduction); err != nil {
		// logger not available yet, fall back to stderr
		fmt.Println("failed to initialize logger: " + err.Error())
		os.Exit(1)
	}

	// cancellable context for background goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// initialize database connection
	mongoClient, err := mongo.NewConnection(cfg.Mongo)
	if err != nil {
		logger.Fatal(ctx, "Failed to connect to MongoDB", err, nil)
	}
	defer mongo.Disconnect(mongoClient)
	logger.Info(ctx, "Connected to MongoDB", map[string]any{"database": cfg.Mongo.Database})

	// initialize redis connection
	redisClient, err := redis.NewConnection(cfg.Redis)
	if err != nil {
		logger.Fatal(ctx, "Failed to connect to Redis", err, nil)
	}
	defer redisClient.Close()
	logger.Info(ctx, "Connected to Redis", nil)

	// initialize rabbitmq connection
	broker, err := rabbitmq.NewRabbitMQAdapter(cfg.RabbitMQ)
	if err != nil {
		logger.Fatal(ctx, "Failed to connect to RabbitMQ", err, nil)
	}
	defer broker.Close()
	logger.Info(ctx, "Connected to RabbitMQ", nil)

	// initialize database and repos
	database := mongoClient.Database(cfg.Mongo.Database)
	customerRepository := repository.NewCustomerRepository(database)
	productRepository := repository.NewProductRepository(database)
	outboxRepository := repository.NewOutboxRepository(database)
	orderRepository := repository.NewOrderRepository(database, outboxRepository)
	txManager := mongo.NewTransactionManager(mongoClient)

	// caches and rate limiter
	orderCache := redis.NewCache[domain.Order](redisClient, "order-cache")
	idempotencyCache := redis.NewCache[service.IdempotencyEntry[domain.Order]](redisClient, "idempotency-cache")
	rateLimiter := redis.NewRateLimiter(redisClient)

	// outbox handler (uses cancellable context)
	outboxHandler := outbox.NewHandler(outboxRepository, broker, cfg.Outbox)
	go outboxHandler.Start(ctx)
	logger.Info(ctx, "Outbox handler started", map[string]any{"interval": cfg.Outbox.Interval.String(), "batch_size": cfg.Outbox.BatchSize})

	// services
	customerService := service.NewCustomerService(customerRepository)
	productService := service.NewProductService(productRepository)
	idempotencyService := service.NewIdempotencyService(idempotencyCache, 15*time.Minute, 1*time.Second, 10*time.Second)
	orderService := service.NewOrderService(orderRepository, productService, customerService, orderCache, idempotencyService, txManager)

	// controllers
	orderController := controllers.NewOrderController(orderService)
	productController := controllers.NewProductController(productService)
	customerController := controllers.NewCustomerController(customerService)
	healthController := controllers.NewHealthController([]controllers.HealthChecker{
		{Name: "mongodb", Check: func(ctx context.Context) error { return mongoClient.Ping(ctx, nil) }},
		{Name: "redis", Check: func(ctx context.Context) error { return redisClient.Ping(ctx) }},
		{Name: "rabbitmq", Check: func(ctx context.Context) error { return broker.HealthCheck() }},
	})

	// router
	router := http.NewRouter(healthController, orderController, productController, customerController, rateLimiter)

	// graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info(ctx, "Received shutdown signal", map[string]interface{}{"signal": sig.String()})
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := logger.Shutdown(shutdownCtx); err != nil {
			fmt.Println("logger shutdown error: " + err.Error())
		}
	}()

	logger.Info(ctx, "Starting HTTP server", map[string]any{"addr": cfg.HTTP.BindInterface + ":" + cfg.HTTP.Port})
	err = router.ListenAndServe(ctx, cfg.HTTP)
	if err != nil {
		logger.Fatal(ctx, "Failed to start HTTP server", err, nil)
	}
}
