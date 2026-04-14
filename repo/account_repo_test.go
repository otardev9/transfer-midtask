package repo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/otardev9/transfer-midtask/domain"
	"github.com/otardev9/transfer-midtask/repo"
)

func TestUpdateMut_NilWhenClean(t *testing.T) {
	r := repo.NewAccountRepo()
	a := domain.NewAccount("x", 1000, domain.AccountStatusActive)
	r.Seed(a)

	// No domain method called — nothing is dirty.
	if mut := r.UpdateMut(a); mut != nil {
		t.Errorf("expected nil mutation for untouched account, got %+v", mut)
	}
}

func TestUpdateMut_OnlyBalanceWhenWithdrawn(t *testing.T) {
	r := repo.NewAccountRepo()
	a := domain.NewAccount("x", 1000, domain.AccountStatusActive)
	r.Seed(a)

	if err := a.Withdraw(300); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}

	mut := r.UpdateMut(a)
	if mut == nil {
		t.Fatal("expected non-nil mutation after Withdraw")
	}
	if got, ok := mut.Updates["balance"]; !ok || got != int64(700) {
		t.Errorf("balance: want 700, got %v (present=%v)", got, ok)
	}
	if _, ok := mut.Updates["status"]; ok {
		t.Error("status must not appear in mutation when unchanged")
	}
	if mut.Table != "accounts" {
		t.Errorf("table: want accounts, got %s", mut.Table)
	}
	if mut.ID != "x" {
		t.Errorf("id: want x, got %s", mut.ID)
	}
}

func TestUpdateMut_OnlyBalanceWhenDeposited(t *testing.T) {
	r := repo.NewAccountRepo()
	a := domain.NewAccount("x", 500, domain.AccountStatusActive)
	r.Seed(a)

	if err := a.Deposit(200); err != nil {
		t.Fatalf("Deposit: %v", err)
	}

	mut := r.UpdateMut(a)
	if mut == nil {
		t.Fatal("expected non-nil mutation after Deposit")
	}
	if got := mut.Updates["balance"]; got != int64(700) {
		t.Errorf("balance: want 700, got %v", got)
	}
	if _, ok := mut.Updates["status"]; ok {
		t.Error("status must not appear in mutation when unchanged")
	}
}

func TestUpdateMut_OnlyStatusWhenMarkedDirty(t *testing.T) {
	r := repo.NewAccountRepo()
	a := domain.NewAccount("x", 1000, domain.AccountStatusActive)
	r.Seed(a)

	// Force-mark status dirty (simulates a suspend operation).
	a.Changes.Mark("status")

	mut := r.UpdateMut(a)
	if mut == nil {
		t.Fatal("expected non-nil mutation when status is dirty")
	}
	if _, ok := mut.Updates["status"]; !ok {
		t.Error("status must appear in mutation when dirty")
	}
	if _, ok := mut.Updates["balance"]; ok {
		t.Error("balance must not appear in mutation when clean")
	}
}

func TestUpdateMut_BothFieldsWhenBothDirty(t *testing.T) {
	r := repo.NewAccountRepo()
	a := domain.NewAccount("x", 1000, domain.AccountStatusActive)
	r.Seed(a)

	_ = a.Withdraw(100)
	a.Changes.Mark("status")

	mut := r.UpdateMut(a)
	if mut == nil {
		t.Fatal("expected non-nil mutation")
	}
	if _, ok := mut.Updates["balance"]; !ok {
		t.Error("balance missing from mutation")
	}
	if _, ok := mut.Updates["status"]; !ok {
		t.Error("status missing from mutation")
	}
}

func TestAccountRepo_Retrieve_NotFound(t *testing.T) {
	r := repo.NewAccountRepo()
	_, err := r.Retrieve(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrAccountNotFound) {
		t.Fatalf("want ErrAccountNotFound, got %v", err)
	}
}

func TestAccountRepo_Retrieve_Found(t *testing.T) {
	r := repo.NewAccountRepo()
	r.Seed(domain.NewAccount("alice", 500, domain.AccountStatusActive))

	a, err := r.Retrieve(context.Background(), "alice")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if a.Balance() != 500 {
		t.Errorf("balance: want 500, got %d", a.Balance())
	}
}
