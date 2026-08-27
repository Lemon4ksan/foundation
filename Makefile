.PHONY: all test race bench lint format fuzz cover

all: format test

test:
	go test ./...

race:
	go test -race -timeout 90s ./...

bench:
	go test -bench=. -benchmem ./...

lint:
	golangci-lint run --timeout=5m ./...

format:
	golangci-lint run --fix ./...

fuzz:
	go run ./scripts/fuzz_all.go -fuzztime=3s

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
