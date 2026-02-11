package port

import (
	"context"

	"github.com/rafaelleal24/challenge/internal/core/domain"
)

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock

type CustomerPort interface {
	Create(ctx context.Context) (domain.ID, error)
	Exists(ctx context.Context, id domain.ID) (bool, error)
}
