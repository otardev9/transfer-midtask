.PHONY: test test-race integration-test lint tidy \
        docker-unit docker-integration docker-demo \
        docker-up docker-down

test:
	go test -v ./...

test-race:
	go test -v -race ./...

integration-test:
	go test -v -tags=integration ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-unit:
	docker compose run --rm unit-test

docker-integration:
	docker compose run --rm integration-test

docker-demo:
	docker compose run --rm demo

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down
