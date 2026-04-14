package repo

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/otardev9/transfer-midtask/contracts"
)

// PostgresCommitter applies a Plan in a single database transaction.
type PostgresCommitter struct {
	db *sql.DB
}

func NewPostgresCommitter(db *sql.DB) *PostgresCommitter {
	return &PostgresCommitter{db: db}
}

func (c *PostgresCommitter) Commit(ctx context.Context, plan *contracts.Plan) error {
	mutations := plan.Mutations()
	if len(mutations) == 0 {
		return nil
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, mut := range mutations {
		query, args := buildUpdateQuery(mut)

		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("apply mutation %s/%s: %w", mut.Table, mut.ID, err)
		}

		n, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected %s/%s: %w", mut.Table, mut.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("no row found for %s/%s", mut.Table, mut.ID)
		}
	}

	return tx.Commit()
}

// buildUpdateQuery turns a Mutation into a parameterised UPDATE statement.
// Column names are from our own code (never user input) so embedding them
// in the query string is safe. Values are always parameterised.
// Keys are sorted to keep generated SQL deterministic.
func buildUpdateQuery(mut *contracts.Mutation) (string, []interface{}) {
	cols := make([]string, 0, len(mut.Updates))
	for col := range mut.Updates {
		cols = append(cols, col)
	}
	sort.Strings(cols)

	setClauses := make([]string, len(cols))
	args := make([]interface{}, 0, len(cols)+1)

	for i, col := range cols {
		setClauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
		args = append(args, mut.Updates[col])
	}
	args = append(args, mut.ID)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		mut.Table,
		strings.Join(setClauses, ", "),
		len(cols)+1,
	)

	return query, args
}
