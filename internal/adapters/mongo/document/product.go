package document

import (
	"time"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	Price       int64              `bson:"price"`
	Stock       int                `bson:"stock"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

func (doc ProductDocument) GetID() primitive.ObjectID {
	return doc.ID
}

func (doc *ProductDocument) ToDomain() *domain.Product {
	return &domain.Product{
		ID:          domain.ID(doc.ID.Hex()),
		Name:        doc.Name,
		Description: doc.Description,
		Price:       domain.Amount(doc.Price),
		Stock:       doc.Stock,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

func ToProductDocument(p *domain.Product) *ProductDocument {
	return &ProductDocument{
		Name:        p.Name,
		Description: p.Description,
		Price:       int64(p.Price),
		Stock:       p.Stock,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
