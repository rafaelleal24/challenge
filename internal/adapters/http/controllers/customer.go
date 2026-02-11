package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rafaelleal24/challenge/internal/adapters/http/handlers"
	"github.com/rafaelleal24/challenge/internal/core/service"
)

type CustomerResponse struct {
	ID string `json:"id"`
}

type CustomerController struct {
	customerService *service.CustomerService
}

func NewCustomerController(customerService *service.CustomerService) *CustomerController {
	return &CustomerController{customerService: customerService}
}

// CreateCustomer godoc
// @Summary     Create a customer
// @Description Creates a new customer and returns its ID
// @Tags        customers
// @Produce     json
// @Success     201 {object} CustomerResponse
// @Failure     500 {object} handlers.ErrorResponse
// @Router      /api/v1/customers [post]
func (cc *CustomerController) CreateCustomer(c *gin.Context) {
	id, err := cc.customerService.Create(c.Request.Context())
	if err != nil {
		handlers.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, CustomerResponse{ID: string(id)})
}
