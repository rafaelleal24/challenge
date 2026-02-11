package dto

type CreateProductRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Price       int    `json:"price" binding:"required,gt=0"`
	Stock       int    `json:"stock" binding:"required,gte=0"`
}
