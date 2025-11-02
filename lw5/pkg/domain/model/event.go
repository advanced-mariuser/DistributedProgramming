package model

import "github.com/google/uuid"

type OrderCreated struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
}

func (e OrderCreated) Type() string { return "OrderCreated" }

type ItemAddedToOrder struct {
	OrderID   uuid.UUID
	ItemID    uuid.UUID
	ProductID uuid.UUID
}

func (e ItemAddedToOrder) Type() string { return "ItemAddedToOrder" }

type ItemRemovedFromOrder struct {
	OrderID uuid.UUID
	ItemID  uuid.UUID
}

func (e ItemRemovedFromOrder) Type() string { return "ItemRemovedFromOrder" }

type OrderSubmittedForPayment struct {
	OrderID    uuid.UUID
	TotalCents int64
}

func (e OrderSubmittedForPayment) Type() string { return "OrderSubmittedForPayment" }

type OrderPaid struct {
	OrderID uuid.UUID
}

func (e OrderPaid) Type() string { return "OrderPaid" }

type OrderCancelled struct {
	OrderID uuid.UUID
	Reason  string
}

func (e OrderCancelled) Type() string { return "OrderCancelled" }
