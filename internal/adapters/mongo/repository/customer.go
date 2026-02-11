package repository

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/document"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/port"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CustomerRepository struct {
	*BaseRepository[document.CustomerDocument]
	collection *mongo.Collection
}

func NewCustomerRepository(db *mongo.Database) port.CustomerPort {
	return &CustomerRepository{
		BaseRepository: NewBaseRepository[document.CustomerDocument](db, "customers"),
		collection:     db.Collection("customers"),
	}
}

func (r *CustomerRepository) Create(ctx context.Context) (domain.ID, error) {
	doc := document.CustomerDocument{}
	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return "", parseError(err)
	}
	return domain.ID(result.InsertedID.(primitive.ObjectID).Hex()), nil
}

func (r *CustomerRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	_, err := r.FindByID(ctx, string(id))
	if err != nil {
		return false, parseError(err)
	}

	return true, nil
}
