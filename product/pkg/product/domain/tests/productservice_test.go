package tests

import (
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	model2 "product/pkg/product/domain/model"
	"product/pkg/product/domain/service"
	"testing"
)

func setup(t *testing.T) (service.ProductService, *mockProductRepository, *mockEventDispatcher) {
	repo := &mockProductRepository{store: make(map[uuid.UUID]*model2.Product)}
	dispatcher := &mockEventDispatcher{}
	productService := service.NewProductService(repo, dispatcher)
	return productService, repo, dispatcher
}

func TestCreateProduct(t *testing.T) {
	productService, repo, _ := setup(t)

	product, err := productService.CreateProduct("Test Book", "A book about testing", 1999, 100)

	require.NoError(t, err)
	require.NotNil(t, product)
	assert.Equal(t, "Test Book", product.Name)
	assert.Equal(t, 100, product.StockQuantity)
	assert.Equal(t, 1, product.Version)
	assert.Equal(t, model2.Available, product.Status)

	savedProduct, _ := repo.Find(product.ID)
	assert.NotNil(t, savedProduct)
}

func TestReserveStock(t *testing.T) {
	productService, repo, dispatcher := setup(t)
	product, _ := productService.CreateProduct("Laptop", "Powerful laptop", 150000, 10)

	t.Run("Success", func(t *testing.T) {
		dispatcher.Reset()
		err := productService.ReserveStock(product.ID, 3)

		require.NoError(t, err)
		updatedProduct, _ := repo.Find(product.ID)
		assert.Equal(t, 7, updatedProduct.StockQuantity)
		assert.Equal(t, 2, updatedProduct.Version)

		require.Len(t, dispatcher.events, 1)
		event := dispatcher.events[0].(model2.ProductStockChanged)
		assert.Equal(t, -3, event.ChangeAmount)
		assert.Equal(t, 7, event.NewQuantity)
	})

	t.Run("Fail on insufficient stock", func(t *testing.T) {
		err := productService.ReserveStock(product.ID, 15) // Пытаемся зарезервировать больше, чем есть (7)
		assert.ErrorIs(t, err, model2.ErrInsufficientStock)
	})

	t.Run("Fail on negative quantity", func(t *testing.T) {
		err := productService.ReserveStock(product.ID, -5)
		assert.ErrorIs(t, err, service.ErrInvalidStockQuantity)
	})
}

type mockProductRepository struct {
	store map[uuid.UUID]*model2.Product
}

func (m *mockProductRepository) NextID() (uuid.UUID, error) { return uuid.New(), nil }
func (m *mockProductRepository) Create(p *model2.Product) error {
	m.store[p.ID] = p
	return nil
}
func (m *mockProductRepository) Find(id uuid.UUID) (*model2.Product, error) {
	if p, ok := m.store[id]; ok {
		clone := *p
		return &clone, nil
	}
	return nil, model2.ErrProductNotFound
}
func (m *mockProductRepository) Update(p *model2.Product) error {
	existing, ok := m.store[p.ID]
	if !ok {
		return model2.ErrProductNotFound
	}
	if existing.Version != p.Version-1 {
		return errors.New("optimistic lock failed")
	}
	m.store[p.ID] = p
	return nil
}

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
