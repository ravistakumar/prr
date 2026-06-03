.PHONY: build test lint fmt

build:
	go build -o prr ./cmd/prr

test:
	go test ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run
