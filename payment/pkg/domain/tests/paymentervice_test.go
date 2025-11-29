package tests

import (
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"payment/pkg/domain/model"
	"payment/pkg/domain/service"
	"testing"
)

// --- Setup ---

func setupPaymentTest(t *testing.T) (service.PaymentService, *mockPaymentRepository, *mockEventDispatcher) {
	repo := newMockPaymentRepository()
	dispatcher := &mockEventDispatcher{}
	svc := service.NewPaymentService(repo, dispatcher)
	return svc, repo, dispatcher
}

// --- Tests ---

func TestCreateWallet(t *testing.T) {
	svc, repo, _ := setupPaymentTest(t)
	userID := uuid.New()

	wallet, err := svc.CreateWallet(userID)

	require.NoError(t, err)
	require.NotNil(t, wallet)
	assert.Equal(t, userID, wallet.UserID)
	assert.Equal(t, int64(0), wallet.BalanceCents)
	assert.Equal(t, 1, wallet.Version)

	saved, err := repo.GetWalletByUserID(userID)
	require.NoError(t, err)
	assert.Equal(t, wallet.ID, saved.ID)
}

func TestDeposit_Success(t *testing.T) {
	svc, repo, dispatcher := setupPaymentTest(t)
	userID := uuid.New()
	wallet, _ := svc.CreateWallet(userID)
	dispatcher.Reset()

	refID := "ref_deposit_1"
	amount := int64(1000)

	updatedWallet, err := svc.Deposit(userID, amount, refID)

	require.NoError(t, err)
	assert.Equal(t, int64(1000), updatedWallet.BalanceCents)
	assert.Equal(t, 2, updatedWallet.Version)

	tx, err := repo.FindTransactionByRef(wallet.ID, refID)
	require.NoError(t, err)
	assert.Equal(t, model.Deposit, tx.Type)
	assert.Equal(t, model.TxCommitted, tx.Status)

	require.Len(t, dispatcher.events, 1)
	event, ok := dispatcher.events[0].(model.FundsDeposited)
	assert.True(t, ok)
	assert.Equal(t, amount, event.AmountCents)
	assert.Equal(t, int64(1000), event.NewBalance)
}

func TestDeposit_Idempotency(t *testing.T) {
	svc, _, dispatcher := setupPaymentTest(t)
	userID := uuid.New()
	svc.CreateWallet(userID)
	dispatcher.Reset()

	refID := "ref_idempotent_1"

	w1, err := svc.Deposit(userID, 500, refID)
	require.NoError(t, err)
	assert.Equal(t, int64(500), w1.BalanceCents)
	require.Len(t, dispatcher.events, 1)

	w2, err := svc.Deposit(userID, 500, refID)
	require.NoError(t, err)

	assert.Equal(t, int64(500), w2.BalanceCents)
	assert.Equal(t, w1.Version, w2.Version)

	require.Len(t, dispatcher.events, 1)
}

func TestPayForOrder_Success(t *testing.T) {
	svc, _, dispatcher := setupPaymentTest(t)
	userID := uuid.New()
	svc.CreateWallet(userID)
	svc.Deposit(userID, 2000, "initial_topup")
	dispatcher.Reset()

	orderID := uuid.New()
	err := svc.PayForOrder(userID, orderID, 500)

	require.NoError(t, err)

	balance, _ := svc.GetBalance(userID)
	assert.Equal(t, int64(1500), balance)

	require.Len(t, dispatcher.events, 1)
	event, ok := dispatcher.events[0].(model.FundsWithdrawn)
	assert.True(t, ok)
	assert.Equal(t, int64(500), event.AmountCents)
	assert.Equal(t, orderID.String(), event.ReferenceID)
}

func TestPayForOrder_InsufficientFunds(t *testing.T) {
	svc, repo, dispatcher := setupPaymentTest(t)
	userID := uuid.New()
	wallet, _ := svc.CreateWallet(userID)
	svc.Deposit(userID, 100, "tiny_deposit")
	dispatcher.Reset()

	orderID := uuid.New()
	err := svc.PayForOrder(userID, orderID, 500)

	assert.ErrorIs(t, err, model.ErrInsufficientFunds)

	balance, _ := svc.GetBalance(userID)
	assert.Equal(t, int64(100), balance)

	tx, findErr := repo.FindTransactionByRef(wallet.ID, orderID.String())
	require.NoError(t, findErr)
	assert.Equal(t, model.TxFailed, tx.Status)
	assert.Equal(t, "insufficient funds", tx.ErrorMessage)

	require.Len(t, dispatcher.events, 1)
	event, ok := dispatcher.events[0].(model.PaymentFailed)
	assert.True(t, ok)
	assert.Equal(t, orderID.String(), event.ReferenceID)
}

func TestOptimisticLocking_Fail(t *testing.T) {
	svc, repo, _ := setupPaymentTest(t)
	userID := uuid.New()
	wallet, _ := svc.CreateWallet(userID)

	storedWallet, _ := repo.GetWalletByUserID(userID)
	storedWallet.Version = 2
	repo.storeWallets[wallet.ID] = storedWallet

	oldWallet := *wallet
	oldWallet.BalanceCents += 100
	oldWallet.Version = 2

	toUpdate := *wallet
	toUpdate.BalanceCents += 50
	toUpdate.Version = 2

	err := repo.UpdateWallet(&toUpdate)
	assert.ErrorIs(t, err, model.ErrOptimisticLock)
}

// --- Mocks ---

type mockPaymentRepository struct {
	storeWallets map[uuid.UUID]*model.Wallet
	storeTxs     []*model.Transaction
}

func newMockPaymentRepository() *mockPaymentRepository {
	return &mockPaymentRepository{
		storeWallets: make(map[uuid.UUID]*model.Wallet),
		storeTxs:     make([]*model.Transaction, 0),
	}
}

func (m *mockPaymentRepository) NextID() (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockPaymentRepository) CreateWallet(w *model.Wallet) error {
	if _, exists := m.storeWallets[w.ID]; exists {
		return errors.New("wallet already exists")
	}
	val := *w
	m.storeWallets[w.ID] = &val
	m.storeWallets[w.UserID] = &val
	return nil
}

func (m *mockPaymentRepository) GetWalletByUserID(userID uuid.UUID) (*model.Wallet, error) {
	for _, w := range m.storeWallets {
		if w.UserID == userID {
			val := *w
			return &val, nil
		}
	}
	return nil, model.ErrWalletNotFound
}

func (m *mockPaymentRepository) UpdateWallet(w *model.Wallet) error {
	existing, ok := m.storeWallets[w.ID]
	if !ok {
		return model.ErrWalletNotFound
	}

	if existing.Version != w.Version-1 {
		return model.ErrOptimisticLock
	}

	val := *w
	m.storeWallets[w.ID] = &val
	return nil
}

func (m *mockPaymentRepository) SaveTransaction(tx *model.Transaction) error {
	val := *tx
	m.storeTxs = append(m.storeTxs, &val)
	return nil
}

func (m *mockPaymentRepository) FindTransactionByRef(walletID uuid.UUID, refID string) (*model.Transaction, error) {
	for _, tx := range m.storeTxs {
		if tx.WalletID == walletID && tx.ReferenceID == refID {
			val := *tx
			return &val, nil
		}
	}
	return nil, nil
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
