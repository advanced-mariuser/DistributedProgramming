package service

import (
	"errors"
	model2 "product/pkg/product/domain/model"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidStockQuantity = errors.New("stock quantity must be a positive number")
	ErrProductNotAvailable  = errors.New("operation cannot be performed on an unavailable or archived product")
)

type Event interface{ Type() string }
type EventDispatcher interface{ Dispatch(event Event) error }

type ProductService interface {
	CreateProduct(name, description string, priceCents int64, initialStock int) (*model2.Product, error)
	ChangeProductPrice(productID uuid.UUID, newPriceCents int64) error
	ReceiveStock(productID uuid.UUID, quantity int) error
	ReserveStock(productID uuid.UUID, quantity int) error
	ArchiveProduct(productID uuid.UUID) error
}

func NewProductService(repo model2.ProductRepository, dispatcher EventDispatcher) ProductService {
	return &productService{repo: repo, dispatcher: dispatcher}
}

type productService struct {
	repo       model2.ProductRepository
	dispatcher EventDispatcher
}

func (s *productService) CreateProduct(name, description string, priceCents int64, initialStock int) (*model2.Product, error) {
	if priceCents < 0 || initialStock < 0 {
		return nil, errors.New("price and stock cannot be negative")
	}

	productID, err := s.repo.NextID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	product := &model2.Product{
		ID:            productID,
		Name:          name,
		Description:   description,
		PriceCents:    priceCents,
		StockQuantity: initialStock,
		Status:        model2.Available,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(product); err != nil {
		return nil, err
	}

	_ = s.dispatcher.Dispatch(model2.ProductCreated{ProductID: productID, Name: name})
	return product, nil
}

func (s *productService) ChangeProductPrice(productID uuid.UUID, newPriceCents int64) error {
	product, err := s.repo.Find(productID)
	if err != nil {
		return err
	}
	if product.Status == model2.Archived {
		return ErrProductNotAvailable
	}
	if newPriceCents < 0 {
		return errors.New("price cannot be negative")
	}

	oldPrice := product.PriceCents
	product.PriceCents = newPriceCents

	if err := s.updateProduct(product); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model2.ProductPriceChanged{
		ProductID:     productID,
		OldPriceCents: oldPrice,
		NewPriceCents: newPriceCents,
	})
	return nil
}

func (s *productService) ReceiveStock(productID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidStockQuantity
	}
	return s.changeStock(productID, quantity)
}

func (s *productService) ReserveStock(productID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidStockQuantity
	}
	return s.changeStock(productID, -quantity)
}

func (s *productService) ArchiveProduct(productID uuid.UUID) error {
	product, err := s.repo.Find(productID)
	if err != nil {
		return err
	}
	if product.Status == model2.Archived {
		return nil
	}

	product.Status = model2.Archived

	if err := s.updateProduct(product); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model2.ProductArchived{ProductID: productID})
	return nil
}

func (s *productService) changeStock(productID uuid.UUID, amount int) error {
	product, err := s.repo.Find(productID)
	if err != nil {
		return err
	}
	if product.Status != model2.Available {
		return ErrProductNotAvailable
	}
	if product.StockQuantity+amount < 0 {
		return model2.ErrInsufficientStock
	}

	product.StockQuantity += amount

	if err := s.updateProduct(product); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model2.ProductStockChanged{
		ProductID:    productID,
		ChangeAmount: amount,
		NewQuantity:  product.StockQuantity,
	})
	return nil
}

func (s *productService) updateProduct(product *model2.Product) error {
	product.Version++
	product.UpdatedAt = time.Now().UTC()
	return s.repo.Update(product)
}
