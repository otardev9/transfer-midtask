package contracts

import (
	"context"

	"github.com/otardev9/transfer-midtask/domain"
)

// Mutation is a pending write to a single row. Created by repositories,
// applied by the Committer — never inside a usecase or repository method.
type Mutation struct {
	Table   string
	ID      string
	Updates map[string]interface{}
}

// Plan accumulates mutations to be committed as a unit.
type Plan struct {
	mutations []*Mutation
}

func NewPlan() *Plan { return &Plan{} }

func (p *Plan) Add(m *Mutation) {
	if m != nil {
		p.mutations = append(p.mutations, m)
	}
}

func (p *Plan) Mutations() []*Mutation { return p.mutations }

// AccountRepository is the port the usecase depends on.
type AccountRepository interface {
	Retrieve(ctx context.Context, id domain.AccountID) (*domain.Account, error)
	// UpdateMut returns a mutation for dirty fields only; nil if nothing changed.
	UpdateMut(account *domain.Account) *Mutation
}
