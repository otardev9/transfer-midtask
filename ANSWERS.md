# Answers

## Q1

`source.Withdraw` succeeds: balance drops in memory, `Changes` marks `balance` dirty. Then `dest.Deposit` fails and `Execute` returns the error immediately. No `Plan` is built — the committer never sees anything. Both DB rows are untouched.

The source `Account` object in memory has a half-applied balance, but it's scoped to this request. Nothing committed it, so nothing in the DB changed.

## Q2

Without a transaction, the two writes are independent. `Apply(mutation1)` debits the source and succeeds. Then the DB node bounces, or a constraint fires on the dest row, before `Apply(mutation2)` runs. Source is now down 300 cents. Dest never received them. The money is gone with no way to recover it programmatically.

Wrapping both in a single `BEGIN`/`COMMIT` means either both land or neither does. That's the only safe option.

## Q3

Two concurrent requests touch the same account:

- Transfer (Op A): reads `{balance: 1000, status: "active"}`, decrements balance, marks `balance` dirty
- Compliance suspend (Op B): reads the same row, flips status, marks `status` dirty

If each writes only its dirty field:

```sql
Op A: UPDATE accounts SET balance = 700 WHERE id = 'X'
Op B: UPDATE accounts SET status = 'suspended' WHERE id = 'X'
```

Order doesn't matter — both land correctly.

If Op A instead writes all fields and lands after Op B:

```sql
Op A: UPDATE accounts SET balance = 700, status = 'active' WHERE id = 'X'
```

Op B's suspension is silently overwritten. The account is active again and can keep transacting. Dirty-field tracking prevents this by making each write surgical.

## Q4

`account.Status()` returns whatever was loaded into memory at `Retrieve` time. If another request has since changed that column, this write stomps it — a classic last-writer-wins race. With only two fields it's already a problem; it gets worse as the entity grows.

The dirty-field approach sidesteps this because untouched fields simply aren't in the `UPDATE`. The only safe alternative to dirty tracking is optimistic locking — add a `version` column, do `WHERE id = $1 AND version = $2`, and fail the write if the version has moved. Without that guard, writing all fields unconditionally is a data loss bug waiting to happen.
