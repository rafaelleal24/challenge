package document

import "go.mongodb.org/mongo-driver/bson/primitive"

type CustomerDocument struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
}

func (doc CustomerDocument) GetID() primitive.ObjectID {
	return doc.ID
}
