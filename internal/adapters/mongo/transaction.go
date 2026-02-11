package mongo

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/core/port"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionManager struct {
	client *mongo.Client
}

func NewTransactionManager(client *mongo.Client) port.TransactionManager {
	return &TransactionManager{client: client}
}

func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := tm.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	return err
}
