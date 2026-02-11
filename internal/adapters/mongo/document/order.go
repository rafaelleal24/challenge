package document

import (
	"time"

	"github.com/rafaelleal24/challenge/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderItemDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ProductID   primitive.ObjectID `bson:"product_id"`
	ProductName string             `bson:"product_name"`
	Quantity    int                `bson:"quantity"`
	UnitPrice   int64              `bson:"unit_price"`
}

type OrderDocument struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty"`
	CustomerID  primitive.ObjectID  `bson:"customer_id"`
	Items       []OrderItemDocument `bson:"items"`
	Status      string              `bson:"status"`
	TotalAmount int64               `bson:"total_amount"`
	CreatedAt   time.Time           `bson:"created_at"`
	UpdatedAt   time.Time           `bson:"updated_at"`
}

func (doc OrderDocument) GetID() primitive.ObjectID {
	return doc.ID
}

func (doc *OrderDocument) ToDomain() *domain.Order {
	items := make([]domain.OrderItem, len(doc.Items))
	for i, itemDoc := range doc.Items {
		items[i] = domain.OrderItem{
			ID:          domain.ID(itemDoc.ID.Hex()),
			ProductID:   domain.ID(itemDoc.ProductID.Hex()),
			ProductName: itemDoc.ProductName,
			Quantity:    itemDoc.Quantity,
			UnitPrice:   domain.Amount(itemDoc.UnitPrice),
		}
	}

	return &domain.Order{
		ID:          domain.ID(doc.ID.Hex()),
		CustomerID:  domain.ID(doc.CustomerID.Hex()),
		Items:       items,
		Status:      domain.OrderStatus(doc.Status),
		TotalAmount: domain.Amount(doc.TotalAmount),
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

func ToDocument(order *domain.Order) *OrderDocument {
	items := make([]OrderItemDocument, len(order.Items))
	for i, item := range order.Items {
		itemDoc := OrderItemDocument{
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   int64(item.UnitPrice),
		}

		if item.ID != "" {
			objectID, _ := primitive.ObjectIDFromHex(string(item.ID))
			itemDoc.ID = objectID
		} else {
			itemDoc.ID = primitive.NewObjectID()
		}

		if item.ProductID != "" {
			productID, _ := primitive.ObjectIDFromHex(string(item.ProductID))
			itemDoc.ProductID = productID
		}

		items[i] = itemDoc
	}

	doc := &OrderDocument{
		Items:       items,
		Status:      string(order.Status),
		TotalAmount: int64(order.TotalAmount),
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}

	if order.ID != "" {
		objectID, _ := primitive.ObjectIDFromHex(string(order.ID))
		doc.ID = objectID
	}

	if order.CustomerID != "" {
		customerID, _ := primitive.ObjectIDFromHex(string(order.CustomerID))
		doc.CustomerID = customerID
	}

	return doc
}
