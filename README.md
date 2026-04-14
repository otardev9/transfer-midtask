# transfer-midtask

Backend assessment: `TransferMoney` usecase.

```
Service → Usecase → Domain → Repository (returns mutations) → Committer (applies)
```

Repositories return mutations. They never apply them. The committer writes the entire Plan atomically in one transaction.

---

## Structure

```
├── domain/             Account entity, Withdraw/Deposit, ChangeTracker
├── contracts/          Repository + Committer interfaces, Mutation, Plan
├── usecases/transfer/  Execute — validates, calls domain, builds Plan
├── repo/               In-memory and Postgres implementations
├── migrations/         001_create_accounts.sql
├── cmd/main.go         Runnable demo (in-memory or Postgres)
├── REVIEW.md           Bug analysis of the provided snippet
└── ANSWERS.md          Q1–Q4
```

---

## Running

```bash
# Unit tests (no DB needed)
go test ./...

# With race detector
go test -race ./...

# Integration tests
make docker-up
go test -tags=integration ./...

# Demo — in-memory
go run ./cmd/main.go

# Demo — Postgres
make docker-up
DATABASE_URL="postgres://app:secret@localhost:5432/transfers?sslmode=disable" go run ./cmd/main.go
```

---

## CI

GitHub Actions runs lint + unit + integration tests on every push and PR. See [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

---

## Make targets

| Target | Description |
|---|---|
| `make test` | Unit tests |
| `make test-race` | Unit tests + race detector (needs gcc) |
| `make integration-test` | Integration tests (needs `DATABASE_URL`) |
| `make lint` | golangci-lint |
| `make docker-unit` | Unit + race inside Docker |
| `make docker-integration` | Full integration stack in Docker |
| `make docker-demo` | Demo binary against Postgres in Docker |
| `make docker-up / docker-down` | Start / stop Postgres |
