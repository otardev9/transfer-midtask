package repo

import (
	"context"
	"fmt"
	"sync"

	"github.com/otardev9/transfer-midtask/contracts"
	"github.com/otardev9/transfer-midtask/domain"
)

// AccountRepo is an in-memory implementation of contracts.AccountRepository
// and contracts.Committer.
type AccountRepo struct {
	mu       sync.RWMutex
	accounts map[domain.AccountID]*domain.Account
}

func NewAccountRepo() *AccountRepo {
	return &AccountRepo{accounts: make(map[domain.AccountID]*domain.Account)}
}

// Seed adds an account so tests can pre-populate the store.
func (r *AccountRepo) Seed(account *domain.Account) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts[account.ID()] = account
}

// Retrieve returns a fresh copy of the stored account so callers cannot
// mutate the live record directly. The returned account has a clean
// ChangeTracker, matching the behaviour of the Postgres implementation.
func (r *AccountRepo) Retrieve(_ context.Context, id domain.AccountID) (*domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, ok := r.accounts[id]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	return domain.NewAccount(account.ID(), account.Balance(), account.Status()), nil
}

func (r *AccountRepo) UpdateMut(account *domain.Account) *contracts.Mutation {
	return buildMutation(account)
}

// Commit applies all mutations in the plan to the in-memory store atomically
// under a write lock. If any target account is missing the operation is
// aborted and no changes are persisted.
func (r *AccountRepo) Commit(_ context.Context, plan *contracts.Plan) error {
	mutations := plan.Mutations()
	if len(mutations) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, mut := range mutations {
		id := domain.AccountID(mut.ID)
		current, ok := r.accounts[id]
		if !ok {
			return fmt.Errorf("account %s not found", id)
		}

		bal := current.Balance()
		status := current.Status()

		if v, ok := mut.Updates["balance"]; ok {
			bal = v.(int64)
		}
		if v, ok := mut.Updates["status"]; ok {
			status = domain.AccountStatus(v.(string))
		}

		r.accounts[id] = domain.NewAccount(id, bal, status)
	}
	return nil
}
