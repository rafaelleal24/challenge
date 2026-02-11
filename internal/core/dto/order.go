package dto

import "github.com/rafaelleal24/challenge/internal/core/domain"

type OrderItem struct {
	ProductID domain.ID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type CreateOrderRequest struct {
	CustomerID domain.ID   `json:"customer_id"`
	Items      []OrderItem `json:"items"`
}
