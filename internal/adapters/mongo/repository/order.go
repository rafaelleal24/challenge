package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rafaelleal24/challenge/internal/adapters/mongo/document"
	"github.com/rafaelleal24/challenge/internal/adapters/outbox"
	"github.com/rafaelleal24/challenge/internal/core/domain"
	"github.com/rafaelleal24/challenge/internal/core/logger"
	"github.com/rafaelleal24/challenge/internal/core/port"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

type OrderRepository struct {
	*BaseRepository[document.OrderDocument]
	db         *mongo.Database
	collection *mongo.Collection
	outbox     outbox.Repository
}

func NewOrderRepository(db *mongo.Database, outbox outbox.Repository) port.OrderPort {
	baseRepo := NewBaseRepository[document.OrderDocument](db, "orders")

	repo := &OrderRepository{
		BaseRepository: baseRepo,
		db:             db,
		collection:     db.Collection("orders"),
		outbox:         outbox,
	}

	if err := repo.createIndexes(context.Background()); err != nil {
		logger.Error(context.Background(), "failed to create indexes", err, map[string]any{
			"collection": "orders",
		})
	}

	return repo
}

func (r *OrderRepository) createIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "customer_id", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetUnique(false),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	if order.ID != "" {
		return errors.New("cannot create order with existing ID")
	}

	doc := document.ToDocument(order)
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return parseError(err)
	}

	order.ID = domain.ID(result.InsertedID.(primitive.ObjectID).Hex())
	order.CreatedAt = doc.CreatedAt
	order.UpdatedAt = doc.UpdatedAt

	for i := range order.Items {
		order.Items[i].ID = domain.ID(doc.Items[i].ID.Hex())
	}

	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Order, error) {
	doc, err := r.FindByID(ctx, string(id))
	if err != nil {
		return nil, err
	}

	return doc.ToDomain(), nil
}

func (r *OrderRepository) GetByCustomerID(ctx context.Context, customerID domain.ID, limit, offset int64) ([]*domain.Order, error) {
	objectID, err := primitive.ObjectIDFromHex(string(customerID))
	if err != nil {
		return nil, parseError(err)
	}

	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	filter := bson.M{"customer_id": objectID}

	docs, err := r.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	orders := make([]*domain.Order, len(docs))
	for i, doc := range docs {
		orders[i] = doc.ToDomain()
	}

	return orders, nil
}

func (r *OrderRepository) GetByStatus(ctx context.Context, status domain.OrderStatus, limit, offset int64) ([]*domain.Order, error) {
	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	filter := bson.M{"status": string(status)}

	docs, err := r.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	orders := make([]*domain.Order, len(docs))
	for i, doc := range docs {
		orders[i] = doc.ToDomain()
	}

	return orders, nil
}

func (r *OrderRepository) UpdateStatusWithOutbox(ctx context.Context, id domain.ID, status domain.OrderStatus, event domain.Event) error {
	objectID, err := primitive.ObjectIDFromHex(string(id))
	if err != nil {
		return parseError(err)
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	session, err := r.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		result, err := r.collection.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{
			"$set": bson.M{
				"status":     string(status),
				"updated_at": time.Now(),
			},
		})
		if err != nil {
			return nil, parseError(err)
		}
		if result.MatchedCount == 0 {
			return nil, serviceerrors.NewNotFoundError("entity not found")
		}

		entry := outbox.Entry{
			EventName:  event.GetName(),
			EntityName: event.GetEntityName(),
			EventData:  eventData,
		}
		if err := r.outbox.Insert(sessCtx, entry); err != nil {
			return nil, err
		}

		return nil, nil
	})

	return err
}

func (r *OrderRepository) Delete(ctx context.Context, id domain.ID) error {
	return r.DeleteByID(ctx, string(id))
}
