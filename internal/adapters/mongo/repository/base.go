package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/document"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BaseRepository[T document.Document] struct {
	collection *mongo.Collection
}

func NewBaseRepository[T document.Document](db *mongo.Database, collectionName string) *BaseRepository[T] {
	return &BaseRepository[T]{
		collection: db.Collection(collectionName),
	}
}

func (r *BaseRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, parseError(err)
	}

	var entity T
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&entity)
	if err != nil {
		return nil, parseError(err)
	}

	return &entity, nil
}

func (r *BaseRepository[T]) Find(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]T, error) {

	cursor, err := r.collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, parseError(err)
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, parseError(err)
	}

	return entities, nil
}

func (r *BaseRepository[T]) FindOne(ctx context.Context, filter bson.M) (*T, error) {

	var entity T
	err := r.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		return nil, parseError(err)
	}

	return &entity, nil
}

func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {

	_, err := r.collection.InsertOne(ctx, entity)
	if err != nil {
		return parseError(err)
	}

	return nil
}

func (r *BaseRepository[T]) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return parseError(err)
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)
	if err != nil {
		return parseError(err)
	}

	if result.MatchedCount == 0 {
		return serviceerrors.NewNotFoundError("entity not found")
	}

	return nil
}

func (r *BaseRepository[T]) DeleteByID(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return parseError(err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return parseError(err)
	}

	if result.DeletedCount == 0 {
		return serviceerrors.NewNotFoundError("entity not found")
	}

	return nil
}

func parseError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return serviceerrors.NewNotFoundError("entity not found")
	}
	if mongo.IsDuplicateKeyError(err) {
		return serviceerrors.NewConflictError("duplicate key error")
	}
	if isInvalidObjectIDError(err) {
		return serviceerrors.NewInvalidRequestError("invalid ID format")
	}
	return err
}

func isInvalidObjectIDError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not a valid ObjectID")
}
