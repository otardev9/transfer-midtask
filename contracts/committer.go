package contracts

import "context"

// Committer applies a Plan atomically. On any failure the whole plan rolls back.
type Committer interface {
	Commit(ctx context.Context, plan *Plan) error
}
