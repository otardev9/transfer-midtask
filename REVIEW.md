# Bug analysis — provided snippet

```go
func (uc *Interactor) Execute(ctx context.Context, req *TransferRequest) error {
    source, _ := uc.repo.Retrieve(ctx, req.FromAccountID)
    dest, _ := uc.repo.Retrieve(ctx, req.ToAccountID)

    source.balance -= req.Amount
    dest.balance += req.Amount

    mutation1 := &Mutation{
        Table:   "accounts",
        ID:      string(source.id),
        Updates: map[string]interface{}{"balance": source.balance},
    }

    mutation2 := &Mutation{
        Table:   "accounts",
        ID:      string(dest.id),
        Updates: map[string]interface{}{"balance": dest.balance},
    }

    if err := uc.db.Apply(mutation1); err != nil {
        return err
    }
    if err := uc.db.Apply(mutation2); err != nil {
        return err
    }

    return nil
}
```

---

## Bug #1 — Errors silently discarded on Retrieve

```go
source, _ := uc.repo.Retrieve(ctx, req.FromAccountID)
dest, _ := uc.repo.Retrieve(ctx, req.ToAccountID)
```

Both errors are thrown away with `_`. If either account does not exist (or the
store is unavailable), the variable is `nil` and the very next line dereferences
it, causing a **panic**. Every error returned by a collaborator must be
propagated to the caller.

---

## Bug #2 — Nil pointer dereference (consequence of Bug #1)

```go
source.balance -= req.Amount   // panics when source == nil
dest.balance   += req.Amount   // panics when dest   == nil
```

Because the errors above are silently discarded, there is no guard before
accessing fields on potentially-nil pointers. The program will crash at runtime
rather than returning a meaningful error.

---

## Bug #3 — Direct field mutation bypasses domain methods

```go
source.balance -= req.Amount
dest.balance   += req.Amount
```

`balance` is unexported in Go. This code only compiles because the buggy
`Execute` lives in the same package as `Account` — which is itself an
architecture violation (the usecase belongs in a separate package and
must use the domain's public API). By reaching past `Withdraw` and
`Deposit`, the code skips **all business rules**:

- No check that `balance >= amount` (negative balances possible)
- No check that either account is `active`
- `ChangeTracker` is never marked, so `UpdateMut` would return `nil` if it were
  later called correctly

---

## Bug #4 — No input validation

The function never checks:

- `req.Amount > 0` — a negative or zero amount silently "transfers" nothing or
  reverses the flow (source receives funds, dest loses them)
- `req.FromAccountID != req.ToAccountID` — a self-transfer would double the
  effect on the same account if it were ever retrieved twice as separate copies

---

## Bug #5 — Usecase constructs Mutation objects directly (architecture violation)

```go
mutation1 := &Mutation{
    Table:   "accounts",
    ID:      string(source.id),
    Updates: map[string]interface{}{"balance": source.balance},
}
```

The usecase layer must not know the persistence schema. Creating `Mutation`
objects directly couples business logic to storage details. The correct pattern
is to call `uc.repo.UpdateMut(account)` and let the repository decide which
fields to include.

---

## Bug #6 — Mutations applied inside the usecase (architecture violation)

```go
if err := uc.db.Apply(mutation1); err != nil { return err }
if err := uc.db.Apply(mutation2); err != nil { return err }
```

The architecture rule is:

> Repositories **return** mutations; a **Committer** applies them.

The usecase calling `uc.db.Apply` directly violates this rule. The function
should return a `*Plan`; the caller (service layer) commits the entire plan
atomically.

---

## Bug #7 — Non-atomic application produces partial writes (money loss)

Because mutations are applied one at a time and outside a transaction:

1. `Apply(mutation1)` succeeds → `source` loses `req.Amount` in the database.
2. `Apply(mutation2)` fails for any reason (network blip, constraint, timeout).
3. The function returns the error — but `source` is already debited.
4. **Funds vanish**: neither account ends up with the money.

The correct approach returns both mutations in a single `Plan` and applies them
in one atomic transaction.

---

## Bug #8 — Wrong return type

The signature returns `error`, but the architecture requires `(*Plan, error)`.
The caller needs the plan to commit atomically. There is no way to retrieve the
mutations when the return type is only `error`.

---

## Bug #9 — Mutations crafted from stale in-memory data, bypassing ChangeTracker

```go
Updates: map[string]interface{}{"balance": source.balance},
```

The mutation is constructed by reading the in-memory value at the time of
execution. Because `Withdraw`/`Deposit` were never called (Bug #3),
`ChangeTracker` was never marked, so the intent of the write is invisible to
any tooling that depends on it. More critically, `source.balance` holds the
value that was read at Retrieve time. If a concurrent process modifies
`balance` between the Retrieve and this write, the stale value silently
overwrites the concurrent change — a **lost update on balance itself**.

A correct implementation calls `uc.repo.UpdateMut(account)`, which consults
`ChangeTracker` to emit only fields with confirmed mutations and nothing else.

---

## Summary

| # | Category            | Severity |
|---|---------------------|----------|
| 1 | Error handling      | Critical |
| 2 | Nil dereference     | Critical |
| 3 | Domain bypass       | Critical |
| 4 | Missing validation  | High     |
| 5 | Architecture (repo) | High     |
| 6 | Architecture (apply)| High     |
| 7 | Non-atomic writes   | Critical |
| 8 | Wrong return type   | High     |
| 9 | Stale-value / ChangeTracker bypass | Medium   |
