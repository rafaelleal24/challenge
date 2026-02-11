package port

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/core/domain"
)

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock

type BrokerPort interface {
	Publish(ctx context.Context, event domain.Event) error
	PublishRaw(ctx context.Context, eventName, entityName string, data []byte) error
	Close() error
}
