.PHONY: build clean fmt lint test install setup check-readme-aliases

BINARY_NAME=cw
BUILD_DIR=./bin

setup:
	@command -v lefthook >/dev/null || (echo "Install lefthook: brew install lefthook" && exit 1)
	lefthook install

build:
	go build -ldflags="-s -w" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/chatwoot

clean:
	rm -rf $(BUILD_DIR)

fmt:
	go fmt ./...

lint:
	golangci-lint run

test:
	go test ./...

check-readme-aliases:
	./scripts/check-readme-aliases.sh

install:
	go build -ldflags="-s -w" -trimpath -o $(shell go env GOPATH)/bin/$(BINARY_NAME) ./cmd/chatwoot

# Development helpers
run:
	go run ./cmd/chatwoot $(ARGS)

deps:
	go mod tidy
	go mod download
