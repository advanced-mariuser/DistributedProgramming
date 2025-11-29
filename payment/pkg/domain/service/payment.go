package service

import (
	"payment/pkg/domain/model"
	"time"

	"github.com/google/uuid"
)

type Event interface{ Type() string }
type EventDispatcher interface{ Dispatch(event Event) error }

type PaymentService interface {
	CreateWallet(userID uuid.UUID) (*model.Wallet, error)
	Deposit(userID uuid.UUID, amountCents int64, referenceID string) (*model.Wallet, error)
	PayForOrder(userID uuid.UUID, orderID uuid.UUID, amountCents int64) error
	GetBalance(userID uuid.UUID) (int64, error)
}

func NewPaymentService(repo model.PaymentRepository, dispatcher EventDispatcher) PaymentService {
	return &paymentService{repo: repo, dispatcher: dispatcher}
}

type paymentService struct {
	repo       model.PaymentRepository
	dispatcher EventDispatcher
}

func (s *paymentService) CreateWallet(userID uuid.UUID) (*model.Wallet, error) {
	id, err := s.repo.NextID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	wallet := &model.Wallet{
		ID:           id,
		UserID:       userID,
		BalanceCents: 0,
		Currency:     "INTERNAL_COIN",
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateWallet(wallet); err != nil {
		return nil, err
	}
	return wallet, nil
}

func (s *paymentService) Deposit(userID uuid.UUID, amountCents int64, referenceID string) (*model.Wallet, error) {
	if amountCents <= 0 {
		return nil, model.ErrInvalidAmount
	}

	return s.processTransaction(userID, amountCents, referenceID, model.Deposit)
}

func (s *paymentService) PayForOrder(userID uuid.UUID, orderID uuid.UUID, amountCents int64) error {
	if amountCents <= 0 {
		return model.ErrInvalidAmount
	}

	_, err := s.processTransaction(userID, amountCents, orderID.String(), model.Withdrawal)
	return err
}

func (s *paymentService) GetBalance(userID uuid.UUID) (int64, error) {
	wallet, err := s.repo.GetWalletByUserID(userID)
	if err != nil {
		return 0, err
	}
	return wallet.BalanceCents, nil
}

func (s *paymentService) processTransaction(userID uuid.UUID, amount int64, refID string, txType model.TransactionType) (*model.Wallet, error) {
	wallet, err := s.repo.GetWalletByUserID(userID)
	if err != nil {
		return nil, err
	}

	existingTx, err := s.repo.FindTransactionByRef(wallet.ID, refID)
	if err == nil && existingTx != nil {
		return wallet, nil
	}

	txID, err := s.repo.NextID()
	if err != nil {
		return nil, err
	}

	tx := &model.Transaction{
		ID:          txID,
		WalletID:    wallet.ID,
		Type:        txType,
		AmountCents: amount,
		ReferenceID: refID,
		Status:      model.TxPending,
		CreatedAt:   time.Now().UTC(),
	}

	if txType == model.Withdrawal {
		if wallet.BalanceCents < amount {
			// Фиксируем неудачную транзакцию (audit log)
			tx.Status = model.TxFailed
			tx.ErrorMessage = model.ErrInsufficientFunds.Error()
			_ = s.repo.SaveTransaction(tx)

			_ = s.dispatcher.Dispatch(model.PaymentFailed{
				WalletID: wallet.ID, ReferenceID: refID, Reason: model.ErrInsufficientFunds.Error(),
			})
			return nil, model.ErrInsufficientFunds
		}
		wallet.BalanceCents -= amount
	} else {
		wallet.BalanceCents += amount
	}

	wallet.Version++
	wallet.UpdatedAt = time.Now().UTC()
	tx.Status = model.TxCommitted

	if err := s.repo.SaveTransaction(tx); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateWallet(wallet); err != nil {
		return nil, err
	}

	if txType == model.Withdrawal {
		_ = s.dispatcher.Dispatch(model.FundsWithdrawn{
			WalletID: wallet.ID, UserID: userID, AmountCents: amount, ReferenceID: refID,
		})
	} else {
		_ = s.dispatcher.Dispatch(model.FundsDeposited{
			WalletID: wallet.ID, UserID: userID, AmountCents: amount, ReferenceID: refID, NewBalance: wallet.BalanceCents,
		})
	}

	return wallet, nil
}
