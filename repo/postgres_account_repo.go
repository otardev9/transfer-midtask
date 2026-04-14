package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/otardev9/transfer-midtask/contracts"
	"github.com/otardev9/transfer-midtask/domain"
)

// PostgresAccountRepo implements contracts.AccountRepository against Postgres.
type PostgresAccountRepo struct {
	db *sql.DB
}

func NewPostgresAccountRepo(db *sql.DB) *PostgresAccountRepo {
	return &PostgresAccountRepo{db: db}
}

func (r *PostgresAccountRepo) Retrieve(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
	const q = `SELECT id, balance, status FROM accounts WHERE id = $1`

	var (
		accID   string
		balance int64
		status  string
	)

	err := r.db.QueryRowContext(ctx, q, string(id)).Scan(&accID, &balance, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("account %s: %w", id, domain.ErrAccountNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("retrieve account %s: %w", id, err)
	}

	return domain.NewAccount(domain.AccountID(accID), balance, domain.AccountStatus(status)), nil
}

func (r *PostgresAccountRepo) UpdateMut(account *domain.Account) *contracts.Mutation {
	return buildMutation(account)
}
