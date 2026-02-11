package document

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OutboxDocument struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	EventName  string             `bson:"event_name"`
	EntityName string             `bson:"entity_name"`
	EventData  string             `bson:"event_data"`
	CreatedAt  time.Time          `bson:"created_at"`
}
