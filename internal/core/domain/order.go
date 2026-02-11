package domain

import "time"

type OrderStatus string

const (
	OrderStatusCreated    OrderStatus = "created"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

func (s OrderStatus) IsValid() bool {
	return s == OrderStatusCreated || s == OrderStatusProcessing || s == OrderStatusShipped || s == OrderStatusDelivered || s == OrderStatusCancelled
}

type Order struct {
	ID          ID
	CustomerID  ID
	Items       []OrderItem
	Status      OrderStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TotalAmount Amount
}

type OrderItem struct {
	ID          ID
	ProductID   ID
	ProductName string
	Quantity    int
	UnitPrice   Amount
}

func (o *OrderItem) CalculateTotalAmount() Amount {
	return o.UnitPrice.Multiply(o.Quantity)
}

func NewOrderItem(productID ID, productName string, quantity int, unitPrice Amount) *OrderItem {
	return &OrderItem{
		ProductID:   productID,
		ProductName: productName,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
	}
}

func CalculateTotalAmount(items []OrderItem) Amount {
	totalAmount := Amount(0)
	for _, item := range items {
		totalAmount = totalAmount.Add(item.UnitPrice.Multiply(item.Quantity))
	}
	return totalAmount
}

func NewOrder(customerID ID, status OrderStatus, items []OrderItem) *Order {
	return &Order{
		CustomerID:  customerID,
		Items:       items,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		TotalAmount: CalculateTotalAmount(items),
	}
}

type OrderUpdateStatusEvent struct {
	OrderID    ID          `json:"order_id"`
	Status     OrderStatus `json:"status"`
	OldStatus  OrderStatus `json:"old_status"`
	UpdatedAt  time.Time   `json:"updated_at"`
	CustomerID ID          `json:"customer_id"`
}

func (e *OrderUpdateStatusEvent) GetName() string {
	return "order.update_status"
}

func (e *OrderUpdateStatusEvent) GetEntityName() string {
	return "order"
}

func NewOrderUpdateStatusEvent(orderID ID, status OrderStatus, oldStatus OrderStatus, updatedAt time.Time, customerID ID) *OrderUpdateStatusEvent {
	return &OrderUpdateStatusEvent{
		OrderID:    orderID,
		Status:     status,
		OldStatus:  oldStatus,
		UpdatedAt:  updatedAt,
		CustomerID: customerID,
	}
}
