.PHONY: all test race bench lint format fuzz cover check arch ratchet

all: format test

check: lint race arch ratchet

arch:
	go test -run TestArchitecture ./...

ratchet:
	go test -coverprofile=coverage.out ./...
	go run ./scripts/coverage_ratchet -profile=coverage.out -baseline=scripts/coverage_baseline.txt -check

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
