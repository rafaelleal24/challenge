package outbox

import "context"

type Entry struct {
	ID         string
	EventName  string
	EntityName string
	EventData  []byte
}

//go:generate mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock
type Repository interface {
	Insert(ctx context.Context, entry Entry) error
	FetchPending(ctx context.Context, limit int) ([]Entry, error)
	Delete(ctx context.Context, id string) error
}
