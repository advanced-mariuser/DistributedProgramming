package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"order/pkg/domain/model"
)

var (
	ErrOrderCannotBeModified = errors.New("order cannot be modified in its current state")
	ErrOrderIsEmpty          = errors.New("cannot process an empty order")
	ErrNegativePrice         = errors.New("item price cannot be negative")
	ErrOrderItemNotFound     = errors.New("order item not found")
)

type Event interface {
	Type() string
}

type EventDispatcher interface {
	Dispatch(event Event) error
}

type OrderService interface {
	CreateNewOrder(customerID uuid.UUID) (*model.Order, error)
	AddItemToOrder(orderID, productID uuid.UUID, priceCents int64) (uuid.UUID, error)
	RemoveItemFromOrder(orderID, itemID uuid.UUID) error

	SubmitOrderForPayment(orderID uuid.UUID) error
	MarkOrderAsPaid(orderID uuid.UUID) error
	CancelOrder(orderID uuid.UUID, reason string) error
}

func NewOrderService(repo model.OrderRepository, dispatcher EventDispatcher) OrderService {
	return &orderService{repo: repo, dispatcher: dispatcher}
}

type orderService struct {
	repo       model.OrderRepository
	dispatcher EventDispatcher
}

func (s *orderService) CreateNewOrder(customerID uuid.UUID) (*model.Order, error) {
	orderID, err := s.repo.NextID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	order := &model.Order{
		ID:         orderID,
		CustomerID: customerID,
		Status:     model.Open,
		Version:    1, // Начальная версия
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(order); err != nil {
		return nil, err
	}

	_ = s.dispatcher.Dispatch(model.OrderCreated{OrderID: orderID, CustomerID: customerID})
	return order, nil
}

func (s *orderService) AddItemToOrder(orderID, productID uuid.UUID, priceCents int64) (uuid.UUID, error) {
	if priceCents < 0 {
		return uuid.Nil, ErrNegativePrice
	}

	order, err := s.repo.Find(orderID)
	if err != nil {
		return uuid.Nil, err
	}

	if order.Status != model.Open {
		return uuid.Nil, ErrOrderCannotBeModified
	}

	itemID, err := s.repo.NextID()
	if err != nil {
		return uuid.Nil, err
	}

	order.Items = append(order.Items, model.Item{ID: itemID, ProductID: productID, PriceCents: priceCents})
	s.recalculateTotal(order)

	if err := s.updateOrder(order); err != nil {
		return uuid.Nil, err
	}

	_ = s.dispatcher.Dispatch(model.ItemAddedToOrder{OrderID: orderID, ItemID: itemID, ProductID: productID})
	return itemID, nil
}

func (s *orderService) RemoveItemFromOrder(orderID, itemID uuid.UUID) error {
	order, err := s.repo.Find(orderID)
	if err != nil {
		return err
	}

	if order.Status != model.Open {
		return ErrOrderCannotBeModified
	}

	itemIndex := -1
	for i, item := range order.Items {
		if item.ID == itemID {
			itemIndex = i
			break
		}
	}
	if itemIndex == -1 {
		return ErrOrderItemNotFound
	}

	order.Items = append(order.Items[:itemIndex], order.Items[itemIndex+1:]...)
	s.recalculateTotal(order)

	if err := s.updateOrder(order); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model.ItemRemovedFromOrder{OrderID: orderID, ItemID: itemID})
	return nil
}

func (s *orderService) SubmitOrderForPayment(orderID uuid.UUID) error {
	order, err := s.repo.Find(orderID)
	if err != nil {
		return err
	}
	if len(order.Items) == 0 {
		return ErrOrderIsEmpty
	}
	if order.Status != model.Open {
		return ErrOrderCannotBeModified
	}

	order.Status = model.Pending
	if err := s.updateOrder(order); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model.OrderSubmittedForPayment{OrderID: orderID, TotalCents: order.TotalCents})
	return nil
}

func (s *orderService) MarkOrderAsPaid(orderID uuid.UUID) error {
	order, err := s.repo.Find(orderID)
	if err != nil {
		return err
	}
	if order.Status != model.Pending {
		return ErrOrderCannotBeModified
	}
	order.Status = model.Paid

	return s.updateOrder(order)
}

func (s *orderService) CancelOrder(orderID uuid.UUID, reason string) error {
	order, err := s.repo.Find(orderID)
	if err != nil {
		return err
	}

	if order.Status != model.Open && order.Status != model.Pending {
		return ErrOrderCannotBeModified
	}
	order.Status = model.Cancelled

	if err := s.updateOrder(order); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model.OrderCancelled{OrderID: orderID, Reason: reason})
	return nil
}

func (s *orderService) recalculateTotal(order *model.Order) {
	var total int64
	for _, item := range order.Items {
		total += item.PriceCents
	}
	order.TotalCents = total
}

func (s *orderService) updateOrder(order *model.Order) error {
	order.Version++
	order.UpdatedAt = time.Now().UTC()
	return s.repo.Update(order)
}
