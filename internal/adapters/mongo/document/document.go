package document

import "go.mongodb.org/mongo-driver/bson/primitive"

type Document interface {
	GetID() primitive.ObjectID
}
