package service

import (
	"log"

	"github.com/google/uuid"

	"product/pkg/common/domain"
	"product/pkg/domain/model"
)

type ProductService interface {
	CreateProduct(name, description string, priceCents int64, initialStock int) (uuid.UUID, error)
	ChangeProductPrice(productID uuid.UUID, newPriceCents int64) error
	ReceiveStock(productID uuid.UUID, quantity int) error
	ReserveStock(productID uuid.UUID, quantity int) error
	ArchiveProduct(productID uuid.UUID) error
}

type productService struct {
	repo       model.ProductRepository
	dispatcher domain.EventDispatcher
}

func NewProductService(repo model.ProductRepository, dispatcher domain.EventDispatcher) ProductService {
	return &productService{
		repo:       repo,
		dispatcher: dispatcher,
	}
}

func (s *productService) CreateProduct(name, description string, priceCents int64, initialStock int) (uuid.UUID, error) {
	productID, err := s.repo.NextID()
	if err != nil {
		return uuid.Nil, err
	}

	product, err := model.NewProduct(productID, name, description, priceCents, initialStock)
	if err != nil {
		return uuid.Nil, err
	}

	if err := s.repo.Store(product); err != nil {
		return uuid.Nil, err
	}

	s.dispatchEvents(product.Events())

	return product.ID(), nil
}

func (s *productService) ChangeProductPrice(productID uuid.UUID, newPriceCents int64) error {
	product, err := s.repo.Find(productID)
	if err != nil {
		return err
	}

	if err := product.ChangePrice(newPriceCents); err != nil {
		return err // Возвращаем бизнес-ошибку от агрегата.
	}

	if err := s.repo.Store(product); err != nil {
		return err // Здесь может вернуться ошибка оптимистической блокировки.
	}

	s.dispatchEvents(product.Events())

	return nil
}

func (s *productService) ReceiveStock(productID uuid.UUID, quantity int) error {
	return s.executeOnProduct(productID, func(p *model.Product) error {
		return p.ReceiveStock(quantity)
	})
}

func (s *productService) ReserveStock(productID uuid.UUID, quantity int) error {
	return s.executeOnProduct(productID, func(p *model.Product) error {
		return p.ReserveStock(quantity)
	})
}

func (s *productService) ArchiveProduct(productID uuid.UUID) error {
	return s.executeOnProduct(productID, func(p *model.Product) error {
		return p.Archive()
	})
}

func (s *productService) executeOnProduct(productID uuid.UUID, action func(p *model.Product) error) error {
	product, err := s.repo.Find(productID)
	if err != nil {
		return err
	}

	if err := action(product); err != nil {
		return err
	}

	if err := s.repo.Store(product); err != nil {
		return err
	}

	s.dispatchEvents(product.Events())
	return nil
}

func (s *productService) dispatchEvents(events []domain.Event) {
	for _, event := range events {
		if err := s.dispatcher.Dispatch(event); err != nil {
			// В реальном проекте используйте структурированный логгер.
			log.Printf("ERROR: failed to dispatch event %s: %v", event.Type(), err)
		}
	}
}
