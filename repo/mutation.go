package repo

import (
	"github.com/otardev9/transfer-midtask/contracts"
	"github.com/otardev9/transfer-midtask/domain"
)

// buildMutation returns a Mutation for any dirty fields; nil if nothing changed.
// Both repository implementations share this so the logic lives in one place.
func buildMutation(account *domain.Account) *contracts.Mutation {
	updates := make(map[string]interface{})

	if account.Changes.IsDirty("balance") {
		updates["balance"] = account.Balance()
	}
	if account.Changes.IsDirty("status") {
		updates["status"] = string(account.Status())
	}

	if len(updates) == 0 {
		return nil
	}

	return &contracts.Mutation{
		Table:   "accounts",
		ID:      string(account.ID()),
		Updates: updates,
	}
}
