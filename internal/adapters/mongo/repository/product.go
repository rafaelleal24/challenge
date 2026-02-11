package repository

import (
	"context"
	"fmt"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/document"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/port"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProductRepository struct {
	*BaseRepository[document.ProductDocument]
	collection *mongo.Collection
}

func NewProductRepository(db *mongo.Database) port.ProductPort {
	return &ProductRepository{
		BaseRepository: NewBaseRepository[document.ProductDocument](db, "products"),
		collection:     db.Collection("products"),
	}
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	doc := document.ToProductDocument(product)

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return parseError(err)
	}

	product.ID = domain.ID(result.InsertedID.(primitive.ObjectID).Hex())
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Product, error) {
	doc, err := r.FindByID(ctx, string(id))
	if err != nil {
		return nil, err
	}

	return doc.ToDomain(), nil
}

func (r *ProductRepository) DeductStock(ctx context.Context, id domain.ID, quantity int) error {
	objectID, err := primitive.ObjectIDFromHex(string(id))
	if err != nil {
		return parseError(err)
	}

	result := r.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": objectID, "stock": bson.M{"$gte": quantity}},
		bson.M{"$inc": bson.M{"stock": -quantity}},
	)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return serviceerrors.NewUnprocessableEntityError(fmt.Sprintf("insufficient stock for product %s", id))
		}
		return result.Err()
	}

	return nil
}

func (r *ProductRepository) GetAll(ctx context.Context) ([]*domain.Product, error) {
	docs, err := r.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	products := make([]*domain.Product, len(docs))
	for i, doc := range docs {
		products[i] = doc.ToDomain()
	}

	return products, nil
}
