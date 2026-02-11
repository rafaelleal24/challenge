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

type ProductController struct {
	productService *service.ProductService
}

type ProductResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewProductResponse(product *domain.Product) ProductResponse {
	return ProductResponse{
		ID:          string(product.ID),
		Name:        product.Name,
		Description: product.Description,
		Price:       int(product.Price),
		Stock:       product.Stock,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}

func NewProductController(productService *service.ProductService) *ProductController {
	return &ProductController{productService: productService}
}

// CreateProduct godoc
// @Summary     Create a product
// @Description Creates a new product
// @Tags        products
// @Accept      json
// @Produce     json
// @Param       request body     dto.CreateProductRequest true "Product data"
// @Success     201     {object} ProductResponse
// @Failure     400     {object} handlers.ErrorResponse
// @Failure     500     {object} handlers.ErrorResponse
// @Router      /api/v1/products [post]
func (pc *ProductController) CreateProduct(c *gin.Context) {
	var request dto.CreateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.HandleError(c, serviceerrors.NewInvalidRequestError(err.Error()))
		return
	}
	product, err := pc.productService.CreateProduct(c.Request.Context(), &request)
	if err != nil {
		handlers.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, NewProductResponse(product))
}

// GetAll godoc
// @Summary     List all products
// @Description Returns all products
// @Tags        products
// @Produce     json
// @Success     200 {array} ProductResponse
// @Failure     500 {object} handlers.ErrorResponse
// @Router      /api/v1/products [get]
func (pc *ProductController) GetAll(c *gin.Context) {
	products, err := pc.productService.GetAll(c.Request.Context())
	if err != nil {
		handlers.HandleError(c, err)
		return
	}

	response := make([]ProductResponse, len(products))
	for i, product := range products {
		response[i] = NewProductResponse(product)
	}

	c.JSON(http.StatusOK, response)
}
