package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rafaelleal24/challenge/internal/adapters/config"
	"github.com/rafaelleal24/challenge/internal/adapters/http/controllers"
	"github.com/rafaelleal24/challenge/internal/adapters/http/middleware"
)

type Router struct {
	healthController   *controllers.HealthController
	orderController    *controllers.OrderController
	productController  *controllers.ProductController
	customerController *controllers.CustomerController
	rateLimiter        middleware.RateLimiter
}

func NewRouter(
	healthController *controllers.HealthController,
	orderController *controllers.OrderController,
	productController *controllers.ProductController,
	customerController *controllers.CustomerController,
	rateLimiter middleware.RateLimiter,
) *Router {
	return &Router{
		healthController:   healthController,
		orderController:    orderController,
		productController:  productController,
		customerController: customerController,
		rateLimiter:        rateLimiter,
	}
}

func (r *Router) SetupRoutes(router *gin.Engine) {
	rl := r.rateLimiter

	apiGroup := router.Group("/api")
	v1Group := apiGroup.Group("/v1")
	{
		v1Group.Use(middleware.LogRequest())
		v1Group.GET("/health", r.healthController.Health)

		v1Group.POST("/orders", middleware.RateLimit(rl, 15, 1*time.Minute), r.orderController.CreateOrder)
		v1Group.GET("/orders/:id", r.orderController.GetOrderByID)
		v1Group.PATCH("/orders/:id/status", middleware.RateLimit(rl, 20, 1*time.Minute), r.orderController.UpdateOrderStatus)

		v1Group.POST("/products", r.productController.CreateProduct)
		v1Group.GET("/products", r.productController.GetAll)

		v1Group.POST("/customers", r.customerController.CreateCustomer)
	}
}

func (r *Router) ListenAndServe(ctx context.Context, config config.HTTPConfig) error {
	engine := gin.Default()
	r.SetupRoutes(engine)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.BindInterface, config.Port),
		Handler: engine,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
