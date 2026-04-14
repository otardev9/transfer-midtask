package domain

import "errors"

// Sentinel errors for domain rules. Callers should use errors.Is to check them.
var (
	ErrInvalidAmount    = errors.New("amount must be positive")
	ErrAccountNotActive = errors.New("account is not active")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountNotFound  = errors.New("account not found")
)

type AccountID string

type AccountStatus string

const (
	AccountStatusActive AccountStatus = "active"
	AccountStatusClosed AccountStatus = "closed"
)

// ChangeTracker records which fields have been mutated since the entity was loaded.
type ChangeTracker struct {
	dirty map[string]bool
}

func (ct *ChangeTracker) Mark(field string) {
	if ct.dirty == nil {
		ct.dirty = make(map[string]bool)
	}
	ct.dirty[field] = true
}

func (ct *ChangeTracker) IsDirty(field string) bool {
	return ct.dirty[field]
}

type Account struct {
	id      AccountID
	balance int64 // cents
	status  AccountStatus
	Changes ChangeTracker
}

func NewAccount(id AccountID, balance int64, status AccountStatus) *Account {
	return &Account{
		id:      id,
		balance: balance,
		status:  status,
	}
}

func (a *Account) ID() AccountID         { return a.id }
func (a *Account) Balance() int64        { return a.balance }
func (a *Account) Status() AccountStatus { return a.status }

func (a *Account) Withdraw(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.status != AccountStatusActive {
		return ErrAccountNotActive
	}
	if a.balance < amount {
		return ErrInsufficientFunds
	}
	a.balance -= amount
	a.Changes.Mark("balance")
	return nil
}

func (a *Account) Deposit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.status != AccountStatusActive {
		return ErrAccountNotActive
	}
	a.balance += amount
	a.Changes.Mark("balance")
	return nil
}
