package transfer

import (
	"context"
	"errors"

	"github.com/otardev9/transfer-midtask/contracts"
	"github.com/otardev9/transfer-midtask/domain"
)

// Interactor handles the TransferMoney use case.
type Interactor struct {
	repo contracts.AccountRepository
}

func NewInteractor(repo contracts.AccountRepository) *Interactor {
	return &Interactor{repo: repo}
}

type TransferRequest struct {
	FromAccountID domain.AccountID
	ToAccountID   domain.AccountID
	Amount        int64
}

func (uc *Interactor) Execute(ctx context.Context, req *TransferRequest) (*contracts.Plan, error) {
	if req == nil {
		return nil, errors.New("request must not be nil")
	}
	if req.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if req.FromAccountID == req.ToAccountID {
		return nil, errors.New("source and destination accounts must differ")
	}

	source, err := uc.repo.Retrieve(ctx, req.FromAccountID)
	if err != nil {
		return nil, err
	}

	dest, err := uc.repo.Retrieve(ctx, req.ToAccountID)
	if err != nil {
		return nil, err
	}

	if err := source.Withdraw(req.Amount); err != nil {
		return nil, err
	}

	if err := dest.Deposit(req.Amount); err != nil {
		return nil, err
	}

	plan := contracts.NewPlan()
	plan.Add(uc.repo.UpdateMut(source))
	plan.Add(uc.repo.UpdateMut(dest))

	return plan, nil
}
