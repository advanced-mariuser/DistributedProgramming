package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock quantity")
)

type ProductStatus int

const (
	Available ProductStatus = iota
	Unavailable
	Archived
)

type Product struct {
	ID            uuid.UUID
	Name          string
	Description   string
	PriceCents    int64
	StockQuantity int
	Status        ProductStatus
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProductRepository interface {
	NextID() (uuid.UUID, error)
	Create(product *Product) error
	Update(product *Product) error
	Find(id uuid.UUID) (*Product, error)
}
