package model

import "github.com/google/uuid"

type ProductCreated struct {
	ProductID uuid.UUID
	Name      string
}

func (e ProductCreated) Type() string { return "ProductCreated" }

type ProductPriceChanged struct {
	ProductID     uuid.UUID
	OldPriceCents int64
	NewPriceCents int64
}

func (e ProductPriceChanged) Type() string { return "ProductPriceChanged" }

type ProductStockChanged struct {
	ProductID    uuid.UUID
	ChangeAmount int // Положительное число - приход, отрицательное - расход
	NewQuantity  int
}

func (e ProductStockChanged) Type() string { return "ProductStockChanged" }

type ProductArchived struct {
	ProductID uuid.UUID
}

func (e ProductArchived) Type() string { return "ProductArchived" }
