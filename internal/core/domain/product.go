package domain

import "time"

type Product struct {
	ID          ID
	Name        string
	Description string
	Price       Amount
	Stock       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewProduct(name string, description string, price Amount, stock int) *Product {
	return &Product{
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
