// Demo wires up and runs a single transfer.
// Set DATABASE_URL to use Postgres; otherwise falls back to in-memory.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/lib/pq"

	"github.com/otardev9/transfer-midtask/contracts"
	"github.com/otardev9/transfer-midtask/domain"
	"github.com/otardev9/transfer-midtask/repo"
	"github.com/otardev9/transfer-midtask/usecases/transfer"
)

func main() {
	ctx := context.Background()

	var (
		r         contracts.AccountRepository
		committer contracts.Committer
	)

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			slog.Error("open database", "err", err)
			os.Exit(1)
		}
		defer db.Close()

		r = repo.NewPostgresAccountRepo(db)
		committer = repo.NewPostgresCommitter(db)
		slog.Info("store", "backend", "postgres")
	} else {
		mem := repo.NewAccountRepo()
		mem.Seed(domain.NewAccount("alice", 100_000, domain.AccountStatusActive))
		mem.Seed(domain.NewAccount("bob", 50_000, domain.AccountStatusActive))
		r = mem
		committer = mem
		slog.Info("store", "backend", "memory", "hint", "set DATABASE_URL for postgres")
	}

	uc := transfer.NewInteractor(r)

	plan, err := uc.Execute(ctx, &transfer.TransferRequest{
		FromAccountID: "alice",
		ToAccountID:   "bob",
		Amount:        10_000, // $100.00 in cents
	})
	if err != nil {
		slog.Error("execute transfer", "err", err)
		os.Exit(1)
	}

	for _, m := range plan.Mutations() {
		slog.Info("mutation", "table", m.Table, "id", m.ID, "updates", m.Updates)
	}

	if err := committer.Commit(ctx, plan); err != nil {
		slog.Error("commit", "err", err)
		os.Exit(1)
	}
	slog.Info("transfer committed", "from", "alice", "to", "bob", "amount_cents", 10_000)
}
