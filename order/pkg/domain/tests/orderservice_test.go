package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order/pkg/domain/model"
	"order/pkg/domain/service"
)

func setup(t *testing.T) (service.OrderService, *mockOrderRepository, *mockEventDispatcher) {
	repo := &mockOrderRepository{
		store: make(map[uuid.UUID]*model.Order),
	}
	dispatcher := &mockEventDispatcher{}
	orderService := service.NewOrderService(repo, dispatcher)
	return orderService, repo, dispatcher
}

func TestCreateNewOrder(t *testing.T) {
	orderService, repo, dispatcher := setup(t)
	customerID := uuid.New()

	order, err := orderService.CreateNewOrder(customerID)

	require.NoError(t, err)
	require.NotNil(t, order)
	assert.Equal(t, 1, order.Version)
	assert.Equal(t, model.Open, order.Status)
	assert.Equal(t, customerID, order.CustomerID)

	savedOrder, ok := repo.store[order.ID]
	require.True(t, ok)
	assert.Equal(t, order.ID, savedOrder.ID)

	require.Len(t, dispatcher.events, 1)
	_, ok = dispatcher.events[0].(model.OrderCreated)
	require.True(t, ok)
}

func TestAddItemToOrder(t *testing.T) {
	orderService, repo, dispatcher := setup(t)
	order, _ := orderService.CreateNewOrder(uuid.New())

	t.Run("Success", func(t *testing.T) {
		dispatcher.Reset()
		itemID, err := orderService.AddItemToOrder(order.ID, uuid.New(), 5000)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, itemID)

		updatedOrder := repo.store[order.ID]
		assert.Equal(t, 2, updatedOrder.Version)
		assert.Len(t, updatedOrder.Items, 1)
		assert.Equal(t, int64(5000), updatedOrder.TotalCents)

		require.Len(t, dispatcher.events, 1)
		_, ok := dispatcher.events[0].(model.ItemAddedToOrder)
		assert.True(t, ok)
	})

	t.Run("Fail on invalid state", func(t *testing.T) {
		repo.store[order.ID].Status = model.Paid
		_, err := orderService.AddItemToOrder(order.ID, uuid.New(), 100)
		assert.ErrorIs(t, err, service.ErrOrderCannotBeModified)
	})

	t.Run("Fail on negative price", func(t *testing.T) {
		repo.store[order.ID].Status = model.Open
		_, err := orderService.AddItemToOrder(order.ID, uuid.New(), -100)
		assert.ErrorIs(t, err, service.ErrNegativePrice)
	})
}

func TestSubmitOrderForPayment(t *testing.T) {
	orderService, repo, dispatcher := setup(t)
	order, _ := orderService.CreateNewOrder(uuid.New())

	t.Run("Fail on empty order", func(t *testing.T) {
		err := orderService.SubmitOrderForPayment(order.ID)
		assert.ErrorIs(t, err, service.ErrOrderIsEmpty)
	})

	t.Run("Success", func(t *testing.T) {
		_, _ = orderService.AddItemToOrder(order.ID, uuid.New(), 1000)
		dispatcher.Reset()

		err := orderService.SubmitOrderForPayment(order.ID)
		require.NoError(t, err)

		updatedOrder := repo.store[order.ID]
		assert.Equal(t, model.Pending, updatedOrder.Status)
		assert.Equal(t, 3, updatedOrder.Version)

		require.Len(t, dispatcher.events, 1)
		_, ok := dispatcher.events[0].(model.OrderSubmittedForPayment)
		assert.True(t, ok)
	})
}

func TestOptimisticLockInRepository(t *testing.T) {
	_, repo, _ := setup(t)
	order, _ := repo.CreateAndReturn(model.Order{ID: uuid.New(), Version: 1})

	order.Version++
	err := repo.Update(order)
	require.NoError(t, err)
	assert.Equal(t, 2, repo.store[order.ID].Version)

	err = repo.Update(order)
	require.Error(t, err, "Update with same version should fail")
	assert.ErrorIs(t, err, model.ErrOptimisticLock)
}

var _ model.OrderRepository = &mockOrderRepository{}

type mockOrderRepository struct {
	store map[uuid.UUID]*model.Order
}

func (m *mockOrderRepository) NextID() (uuid.UUID, error) {
	return uuid.NewRandom()
}

func (m *mockOrderRepository) CreateAndReturn(order model.Order) (*model.Order, error) {
	stored := order
	m.store[order.ID] = &stored
	clone := stored
	return &clone, nil
}

func (m *mockOrderRepository) Create(order *model.Order) error {
	if _, exists := m.store[order.ID]; exists {
		return errors.New("order with this ID already exists")
	}
	m.store[order.ID] = order
	return nil
}

func (m *mockOrderRepository) Find(id uuid.UUID) (*model.Order, error) {
	if order, ok := m.store[id]; ok && order.DeletedAt == nil {
		clone := *order
		return &clone, nil
	}
	return nil, model.ErrOrderNotFound
}

func (m *mockOrderRepository) Update(order *model.Order) error {
	existing, ok := m.store[order.ID]
	if !ok {
		return model.ErrOrderNotFound
	}

	if existing.Version != order.Version-1 {
		return model.ErrOptimisticLock
	}

	updated := *order
	m.store[order.ID] = &updated
	return nil
}

func (m *mockOrderRepository) Delete(id uuid.UUID) error {
	order, ok := m.store[id]
	if !ok {
		return model.ErrOrderNotFound
	}
	now := time.Now().UTC()
	order.DeletedAt = &now
	m.store[id] = order
	return nil
}

var _ service.EventDispatcher = &mockEventDispatcher{}

type mockEventDispatcher struct {
	events []service.Event
}

func (m *mockEventDispatcher) Dispatch(event service.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventDispatcher) Reset() {
	m.events = nil
}
