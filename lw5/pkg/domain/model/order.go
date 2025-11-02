package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrOrderNotFound  = errors.New("order not found")
	ErrOptimisticLock = errors.New("order has been modified by another transaction")
)

type OrderStatus int

const (
	Open OrderStatus = iota
	Pending
	Paid
	Cancelled
)

type Order struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Status     OrderStatus
	Items      []Item
	TotalCents int64
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type Item struct {
	ID         uuid.UUID
	ProductID  uuid.UUID
	PriceCents int64
}

type OrderRepository interface {
	NextID() (uuid.UUID, error)
	Create(order *Order) error
	Find(id uuid.UUID) (*Order, error)
	Update(order *Order) error // Заменяем Store на Update
	Delete(id uuid.UUID) error
}
