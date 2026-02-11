package repository

import (
	"context"
	"time"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/document"
	"github.com/rafaelleal24/challenge/internal/adapters/outbox"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OutboxRepository struct {
	collection *mongo.Collection
}

func NewOutboxRepository(db *mongo.Database) outbox.Repository {
	return &OutboxRepository{
		collection: db.Collection("outbox"),
	}
}

func (r *OutboxRepository) Insert(ctx context.Context, entry outbox.Entry) error {
	doc := document.OutboxDocument{
		EventName:  entry.EventName,
		EntityName: entry.EntityName,
		EventData:  string(entry.EventData),
		CreatedAt:  time.Now(),
	}
	_, err := r.collection.InsertOne(ctx, doc)
	return err
}

func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]outbox.Entry, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []document.OutboxDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	entries := make([]outbox.Entry, len(docs))
	for i, doc := range docs {
		entries[i] = outbox.Entry{
			ID:         doc.ID.Hex(),
			EventName:  doc.EventName,
			EntityName: doc.EntityName,
			EventData:  []byte(doc.EventData),
		}
	}

	return entries, nil
}

func (r *OutboxRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return parseError(err)
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}
