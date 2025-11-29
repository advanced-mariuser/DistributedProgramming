package model

import "github.com/google/uuid"

type FundsDeposited struct {
	WalletID    uuid.UUID
	UserID      uuid.UUID
	AmountCents int64
	ReferenceID string
	NewBalance  int64
}

func (e FundsDeposited) Type() string { return "FundsDeposited" }

type FundsWithdrawn struct {
	WalletID    uuid.UUID
	UserID      uuid.UUID
	AmountCents int64
	ReferenceID string
}

func (e FundsWithdrawn) Type() string { return "FundsWithdrawn" }

type PaymentFailed struct {
	WalletID    uuid.UUID
	ReferenceID string
	Reason      string
}

func (e PaymentFailed) Type() string { return "PaymentFailed" }
