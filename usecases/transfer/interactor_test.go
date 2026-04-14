package transfer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/otardev9/transfer-midtask/domain"
	"github.com/otardev9/transfer-midtask/repo"
	"github.com/otardev9/transfer-midtask/usecases/transfer"
)

func setup(accounts ...*domain.Account) *transfer.Interactor {
	r := repo.NewAccountRepo()
	for _, a := range accounts {
		r.Seed(a)
	}
	return transfer.NewInteractor(r)
}

func TestExecute_HappyPath(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 1000, domain.AccountStatusActive),
		domain.NewAccount("acc-b", 500, domain.AccountStatusActive),
	)

	plan, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-b",
		Amount:        300,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}

	muts := plan.Mutations()
	if len(muts) != 2 {
		t.Fatalf("expected 2 mutations, got %d", len(muts))
	}

	for _, m := range muts {
		if _, ok := m.Updates["balance"]; !ok {
			t.Errorf("mutation for %s is missing balance field", m.ID)
		}
		if _, ok := m.Updates["status"]; ok {
			t.Errorf("mutation for %s should not include unchanged status field", m.ID)
		}
	}
}

func TestExecute_InsufficientFunds(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 100, domain.AccountStatusActive),
		domain.NewAccount("acc-b", 500, domain.AccountStatusActive),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-b",
		Amount:        300,
	})
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("want ErrInsufficientFunds, got %v", err)
	}
}

func TestExecute_NegativeAmount(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 1000, domain.AccountStatusActive),
		domain.NewAccount("acc-b", 500, domain.AccountStatusActive),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-b",
		Amount:        -50,
	})
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("want ErrInvalidAmount, got %v", err)
	}
}

func TestExecute_ZeroAmount(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 1000, domain.AccountStatusActive),
		domain.NewAccount("acc-b", 500, domain.AccountStatusActive),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-b",
		Amount:        0,
	})
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("want ErrInvalidAmount, got %v", err)
	}
}

func TestExecute_SameAccount(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 1000, domain.AccountStatusActive),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-a",
		Amount:        100,
	})
	if err == nil {
		t.Fatal("expected error when source and destination are the same")
	}
}

func TestExecute_SourceNotFound(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-b", 500, domain.AccountStatusActive),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-missing",
		ToAccountID:   "acc-b",
		Amount:        100,
	})
	if !errors.Is(err, domain.ErrAccountNotFound) {
		t.Fatalf("want ErrAccountNotFound, got %v", err)
	}
}

func TestExecute_DestNotFound(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 1000, domain.AccountStatusActive),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-missing",
		Amount:        100,
	})
	if !errors.Is(err, domain.ErrAccountNotFound) {
		t.Fatalf("want ErrAccountNotFound, got %v", err)
	}
}

func TestExecute_ClosedSourceAccount(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 1000, domain.AccountStatusClosed),
		domain.NewAccount("acc-b", 500, domain.AccountStatusActive),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-b",
		Amount:        100,
	})
	if !errors.Is(err, domain.ErrAccountNotActive) {
		t.Fatalf("want ErrAccountNotActive, got %v", err)
	}
}

func TestExecute_ClosedDestAccount(t *testing.T) {
	uc := setup(
		domain.NewAccount("acc-a", 1000, domain.AccountStatusActive),
		domain.NewAccount("acc-b", 500, domain.AccountStatusClosed),
	)

	_, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-b",
		Amount:        100,
	})
	if !errors.Is(err, domain.ErrAccountNotActive) {
		t.Fatalf("want ErrAccountNotActive, got %v", err)
	}
}

func TestExecute_WithdrawFailure_DestUnchanged(t *testing.T) {
	r := repo.NewAccountRepo()
	r.Seed(domain.NewAccount("acc-a", 50, domain.AccountStatusActive))
	r.Seed(domain.NewAccount("acc-b", 500, domain.AccountStatusActive))

	uc := transfer.NewInteractor(r)

	plan, err := uc.Execute(context.Background(), &transfer.TransferRequest{
		FromAccountID: "acc-a",
		ToAccountID:   "acc-b",
		Amount:        300,
	})
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("want ErrInsufficientFunds, got %v", err)
	}
	if plan != nil {
		t.Fatal("plan must be nil when an error is returned")
	}
}
