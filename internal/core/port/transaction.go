package port

import "context"

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock

type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
