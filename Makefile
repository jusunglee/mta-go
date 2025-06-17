.PHONY: all build test clean proto run-server run-local

# Build variables
BINARY_SERVER=mta-server
BINARY_LOCAL=mta-local
GO=go
GOFLAGS=-v

all: build

build: build-server build-local

build-server:
	$(GO) build $(GOFLAGS) -o $(BINARY_SERVER) ./cmd/server

build-local:
	$(GO) build $(GOFLAGS) -o $(BINARY_LOCAL) ./cmd/local

test:
	$(GO) test -v ./...

clean:
	rm -f $(BINARY_SERVER) $(BINARY_LOCAL)
	$(GO) clean

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto

run-server: build-server
	./$(BINARY_SERVER) -api-key=$(MTA_API_KEY)

run-local: build-local
	./$(BINARY_LOCAL) -api-key=$(MTA_API_KEY)

deps:
	$(GO) mod download
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

.DEFAULT_GOAL := build