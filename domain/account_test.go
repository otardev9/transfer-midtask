package domain_test

import (
	"testing"

	"github.com/otardev9/transfer-midtask/domain"
)

// ── ChangeTracker ────────────────────────────────────────────────────────────

func TestChangeTracker_IsDirty_FalseByDefault(t *testing.T) {
	var ct domain.ChangeTracker
	if ct.IsDirty("balance") {
		t.Error("a fresh ChangeTracker must report nothing as dirty")
	}
}

func TestChangeTracker_Mark_ThenIsDirty(t *testing.T) {
	var ct domain.ChangeTracker
	ct.Mark("balance")
	if !ct.IsDirty("balance") {
		t.Error("balance must be dirty after Mark")
	}
	if ct.IsDirty("status") {
		t.Error("status must still be clean")
	}
}

// ── Withdraw ─────────────────────────────────────────────────────────────────

func TestWithdraw_Success(t *testing.T) {
	a := domain.NewAccount("x", 1000, domain.AccountStatusActive)
	if err := a.Withdraw(300); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Balance() != 700 {
		t.Errorf("balance: want 700, got %d", a.Balance())
	}
	if !a.Changes.IsDirty("balance") {
		t.Error("balance must be dirty after Withdraw")
	}
}

func TestWithdraw_ExactBalance(t *testing.T) {
	a := domain.NewAccount("x", 500, domain.AccountStatusActive)
	if err := a.Withdraw(500); err != nil {
		t.Fatalf("draining exact balance should succeed: %v", err)
	}
	if a.Balance() != 0 {
		t.Errorf("balance: want 0, got %d", a.Balance())
	}
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	a := domain.NewAccount("x", 100, domain.AccountStatusActive)
	if err := a.Withdraw(300); err == nil {
		t.Fatal("expected error for insufficient funds")
	}
	if a.Balance() != 100 {
		t.Error("balance must not change on a failed Withdraw")
	}
	if a.Changes.IsDirty("balance") {
		t.Error("balance must not be marked dirty after a failed Withdraw")
	}
}

func TestWithdraw_ZeroAmount(t *testing.T) {
	a := domain.NewAccount("x", 1000, domain.AccountStatusActive)
	if err := a.Withdraw(0); err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestWithdraw_NegativeAmount(t *testing.T) {
	a := domain.NewAccount("x", 1000, domain.AccountStatusActive)
	if err := a.Withdraw(-50); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestWithdraw_ClosedAccount(t *testing.T) {
	a := domain.NewAccount("x", 1000, domain.AccountStatusClosed)
	if err := a.Withdraw(100); err == nil {
		t.Fatal("expected error for closed account")
	}
	if a.Changes.IsDirty("balance") {
		t.Error("balance must not be dirty after a rejected Withdraw")
	}
}

// ── Deposit ──────────────────────────────────────────────────────────────────

func TestDeposit_Success(t *testing.T) {
	a := domain.NewAccount("x", 500, domain.AccountStatusActive)
	if err := a.Deposit(200); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Balance() != 700 {
		t.Errorf("balance: want 700, got %d", a.Balance())
	}
	if !a.Changes.IsDirty("balance") {
		t.Error("balance must be dirty after Deposit")
	}
}

func TestDeposit_ZeroAmount(t *testing.T) {
	a := domain.NewAccount("x", 500, domain.AccountStatusActive)
	if err := a.Deposit(0); err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestDeposit_NegativeAmount(t *testing.T) {
	a := domain.NewAccount("x", 500, domain.AccountStatusActive)
	if err := a.Deposit(-100); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestDeposit_ClosedAccount(t *testing.T) {
	a := domain.NewAccount("x", 500, domain.AccountStatusClosed)
	if err := a.Deposit(100); err == nil {
		t.Fatal("expected error for closed account")
	}
	if a.Changes.IsDirty("balance") {
		t.Error("balance must not be dirty after a rejected Deposit")
	}
}
