package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrDuplicateTransaction = errors.New("transaction with this reference already exists")
	ErrInvalidAmount        = errors.New("amount must be positive")
	ErrOptimisticLock       = errors.New("wallet was modified by another transaction")
)

type TransactionType int

const (
	Deposit TransactionType = iota
	Withdrawal
)

type TransactionStatus int

const (
	TxPending TransactionStatus = iota
	TxCommitted
	TxFailed
)

type Wallet struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	BalanceCents int64
	Currency     string // e.g., "USD", "COINS"
	Version      int    // Для оптимистической блокировки
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Transaction struct {
	ID           uuid.UUID
	WalletID     uuid.UUID
	Type         TransactionType
	AmountCents  int64
	ReferenceID  string // ID заказа или пополнения для идемпотентности
	Status       TransactionStatus
	ErrorMessage string
	CreatedAt    time.Time
}

type PaymentRepository interface {
	NextID() (uuid.UUID, error)

	CreateWallet(wallet *Wallet) error
	GetWalletByUserID(userID uuid.UUID) (*Wallet, error)
	UpdateWallet(wallet *Wallet) error

	SaveTransaction(tx *Transaction) error
	FindTransactionByRef(walletID uuid.UUID, referenceID string) (*Transaction, error)
}
