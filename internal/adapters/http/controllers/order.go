package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rafaelleal24/challenge/internal/adapters/http/handlers"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/dto"
	"github.com/rafaelleal24/challenge/internal/core/service"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

type OrderController struct {
	orderService *service.OrderService
}

type OrderItemResponse struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int    `json:"unit_price"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type OrderResponse struct {
	ID          string              `json:"id"`
	CustomerID  string              `json:"customer_id"`
	Items       []OrderItemResponse `json:"items"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	TotalAmount int                 `json:"total_amount"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

func NewOrderItemResponse(item domain.OrderItem) OrderItemResponse {
	return OrderItemResponse{
		ID:          string(item.ID),
		ProductID:   string(item.ProductID),
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		UnitPrice:   int(item.UnitPrice),
	}
}

func NewOrderResponse(order *domain.Order) OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = NewOrderItemResponse(item)
	}
	return OrderResponse{
		ID:          string(order.ID),
		CustomerID:  string(order.CustomerID),
		Items:       items,
		Status:      string(order.Status),
		CreatedAt:   order.CreatedAt,
		TotalAmount: int(order.TotalAmount),
		UpdatedAt:   order.UpdatedAt,
	}
}

func NewOrderController(orderService *service.OrderService) *OrderController {
	return &OrderController{orderService: orderService}
}

// UpdateOrderStatus godoc
// @Summary     Update order status
// @Description Updates the status of an existing order
// @Tags        orders
// @Accept      json
// @Produce     json
// @Param       id      path     string              true "Order ID"
// @Param       request body     UpdateStatusRequest  true "New status"
// @Success     200     {object} MessageResponse
// @Failure     400     {object} handlers.ErrorResponse
// @Failure     404     {object} handlers.ErrorResponse
// @Failure     422     {object} handlers.ErrorResponse
// @Failure     429     {object} handlers.ErrorResponse
// @Failure     500     {object} handlers.ErrorResponse
// @Router      /api/v1/orders/{id}/status [patch]
func (orderController *OrderController) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	if !domain.ValidateID(orderID) {
		handlers.HandleError(c, serviceerrors.NewInvalidRequestError("Invalid order ID"))
		return
	}
	var request UpdateStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.HandleError(c, serviceerrors.NewInvalidRequestError(err.Error()))
		return
	}
	if err := orderController.orderService.UpdateOrderStatus(c.Request.Context(), domain.ID(orderID), domain.OrderStatus(request.Status)); err != nil {
		handlers.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "Order status updated successfully"})
}

// CreateOrder godoc
// @Summary     Create an order
// @Description Creates a new order with stock deduction and idempotency support
// @Tags        orders
// @Accept      json
// @Produce     json
// @Param       Idempotency-Key header   string                 false "Idempotency key"
// @Param       request         body     dto.CreateOrderRequest  true  "Order data"
// @Success     201             {object} OrderResponse
// @Failure     400             {object} handlers.ErrorResponse
// @Failure     404             {object} handlers.ErrorResponse
// @Failure     409             {object} handlers.ErrorResponse
// @Failure     422             {object} handlers.ErrorResponse
// @Failure     429             {object} handlers.ErrorResponse
// @Failure     500             {object} handlers.ErrorResponse
// @Router      /api/v1/orders [post]
func (OrderController *OrderController) CreateOrder(c *gin.Context) {
	var request dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.HandleError(c, serviceerrors.NewInvalidRequestError(err.Error()))
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")
	order, err := OrderController.orderService.CreateOrder(c.Request.Context(), idempotencyKey, &request)
	if err != nil {
		handlers.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, NewOrderResponse(order))
}

// GetOrderByID godoc
// @Summary     Get order by ID
// @Description Returns a single order by its ID
// @Tags        orders
// @Produce     json
// @Param       id  path     string true "Order ID"
// @Success     200 {object} OrderResponse
// @Failure     400 {object} handlers.ErrorResponse
// @Failure     404 {object} handlers.ErrorResponse
// @Failure     500 {object} handlers.ErrorResponse
// @Router      /api/v1/orders/{id} [get]
func (orderController *OrderController) GetOrderByID(c *gin.Context) {
	orderID := c.Param("id")
	if !domain.ValidateID(orderID) {
		handlers.HandleError(c, serviceerrors.NewInvalidRequestError("Invalid order ID"))
		return
	}
	order, err := orderController.orderService.GetOrderByID(c.Request.Context(), domain.ID(orderID))
	if err != nil {
		handlers.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, NewOrderResponse(order))
}
