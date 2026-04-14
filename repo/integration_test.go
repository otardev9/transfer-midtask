//go:build integration

package repo_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/otardev9/transfer-midtask/contracts"
	"github.com/otardev9/transfer-midtask/domain"
	"github.com/otardev9/transfer-midtask/repo"
	"github.com/otardev9/transfer-midtask/usecases/transfer"
)

// openDB connects to the database specified by DATABASE_URL.
// If the variable is unset the test is skipped rather than failed so that
// `go test ./...` (without -tags=integration) continues to work without a DB.
func openDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		t.Fatalf("db.Ping: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// seedAccount inserts or replaces a row so tests start from a known state.
func seedAccount(t *testing.T, db *sql.DB, id string, balance int64, status string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO accounts (id, balance, status) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET balance = $2, status = $3`,
		id, balance, status,
	)
	if err != nil {
		t.Fatalf("seedAccount %s: %v", id, err)
	}
	t.Cleanup(func() {
		db.ExecContext(context.Background(), "DELETE FROM accounts WHERE id = $1", id) //nolint:errcheck
	})
}

func queryBalance(t *testing.T, db *sql.DB, id string) int64 {
	t.Helper()
	var balance int64
	if err := db.QueryRowContext(context.Background(),
		"SELECT balance FROM accounts WHERE id = $1", id,
	).Scan(&balance); err != nil {
		t.Fatalf("queryBalance %s: %v", id, err)
	}
	return balance
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestIntegration_Transfer_HappyPath(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	// Use the test name as a suffix to keep rows isolated across parallel runs.
	src := fmt.Sprintf("src-%s", t.Name())
	dst := fmt.Sprintf("dst-%s", t.Name())

	seedAccount(t, db, src, 1000, "active")
	seedAccount(t, db, dst, 500, "active")

	r := repo.NewPostgresAccountRepo(db)
	committer := repo.NewPostgresCommitter(db)
	uc := transfer.NewInteractor(r)

	plan, err := uc.Execute(ctx, &transfer.TransferRequest{
		FromAccountID: domain.AccountID(src),
		ToAccountID:   domain.AccountID(dst),
		Amount:        300,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if err := committer.Commit(ctx, plan); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if got := queryBalance(t, db, src); got != 700 {
		t.Errorf("source balance: want 700, got %d", got)
	}
	if got := queryBalance(t, db, dst); got != 800 {
		t.Errorf("dest balance: want 800, got %d", got)
	}
}

func TestIntegration_Transfer_InsufficientFunds_NoDB_Change(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	src := fmt.Sprintf("src-%s", t.Name())
	dst := fmt.Sprintf("dst-%s", t.Name())

	seedAccount(t, db, src, 100, "active")
	seedAccount(t, db, dst, 500, "active")

	r := repo.NewPostgresAccountRepo(db)
	uc := transfer.NewInteractor(r)

	_, err := uc.Execute(ctx, &transfer.TransferRequest{
		FromAccountID: domain.AccountID(src),
		ToAccountID:   domain.AccountID(dst),
		Amount:        300,
	})
	if err == nil {
		t.Fatal("expected insufficient-funds error")
	}

	// No plan was returned, so nothing was committed — DB must be unchanged.
	if got := queryBalance(t, db, src); got != 100 {
		t.Errorf("source must be unchanged: want 100, got %d", got)
	}
	if got := queryBalance(t, db, dst); got != 500 {
		t.Errorf("dest must be unchanged: want 500, got %d", got)
	}
}

func TestIntegration_Transfer_ClosedSource(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	src := fmt.Sprintf("src-%s", t.Name())
	dst := fmt.Sprintf("dst-%s", t.Name())

	seedAccount(t, db, src, 1000, "closed")
	seedAccount(t, db, dst, 500, "active")

	r := repo.NewPostgresAccountRepo(db)
	uc := transfer.NewInteractor(r)

	_, err := uc.Execute(ctx, &transfer.TransferRequest{
		FromAccountID: domain.AccountID(src),
		ToAccountID:   domain.AccountID(dst),
		Amount:        100,
	})
	if err == nil {
		t.Fatal("expected error for closed source account")
	}
}

// TestIntegration_Committer_Atomicity seeds two accounts then verifies that
// if a mutation targets a non-existent row, the whole commit rolls back and
// the first account is also left unchanged.
func TestIntegration_Committer_Atomicity(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	src := fmt.Sprintf("src-%s", t.Name())
	seedAccount(t, db, src, 1000, "active")

	// Build a plan manually: first mutation is valid, second targets a ghost row.
	plan := contracts.NewPlan()
	plan.Add(&contracts.Mutation{
		Table:   "accounts",
		ID:      src,
		Updates: map[string]interface{}{"balance": int64(700)},
	})
	plan.Add(&contracts.Mutation{
		Table:   "accounts",
		ID:      "ghost-account", // does not exist → committer returns error
		Updates: map[string]interface{}{"balance": int64(999)},
	})

	committer := repo.NewPostgresCommitter(db)
	if err := committer.Commit(ctx, plan); err == nil {
		t.Fatal("expected commit to fail due to missing ghost row")
	}

	// The first mutation must have been rolled back.
	if got := queryBalance(t, db, src); got != 1000 {
		t.Errorf("source must be rolled back to 1000, got %d", got)
	}
}
