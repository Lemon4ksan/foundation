.PHONY: all test race bench lint format

all: format test

test:
	go test -v ./...

race:
	go test -race -timeout 60s ./...

bench:
	go test -bench=. -benchmem ./...

lint:
	golangci-lint run ./...

format:
	go fmt ./...
