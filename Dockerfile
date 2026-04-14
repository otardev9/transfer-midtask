# ── base: download deps and copy source ──────────────────────────────────────
FROM golang:1.22-alpine AS base
WORKDIR /app

# git  — required by go mod for some VCS operations
# gcc + musl-dev — required by the Go race detector (CGO)
RUN apk add --no-cache git ca-certificates gcc musl-dev

COPY go.mod ./
# go.sum may not exist yet in a fresh clone; tidy will create it.
COPY go.sum* ./
RUN go mod download || true

COPY . .
# Ensure go.sum is up to date (idempotent, safe to run every build).
RUN go mod tidy

# ── unit-test: run all non-integration tests ──────────────────────────────────
FROM base AS unit-test
CMD ["go", "test", "-v", "-race", "./..."]

# ── integration-test: run tests that require a live database ─────────────────
FROM base AS integration-test
CMD ["go", "test", "-v", "-race", "-tags=integration", "./..."]

# ── build: compile the demo binary ───────────────────────────────────────────
FROM base AS build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /app/bin/demo ./cmd/main.go

# ── prod: minimal runtime image ──────────────────────────────────────────────
FROM alpine:3.19 AS prod
RUN apk add --no-cache ca-certificates
COPY --from=build /app/bin/demo /demo
ENTRYPOINT ["/demo"]
