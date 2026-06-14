.PHONY: all server agent test lint clean docker

# Build variables
BINARY_SERVER=cloudbridge-server
BINARY_AGENT=cloudbridge-agent
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
BUILD_DIR=./bin

all: server agent

# Build server binary
server:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER) ./cmd/server

# Build agent binary
agent:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_AGENT) ./cmd/agent

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

# Build Docker images
docker:
	docker build -f deploy/Dockerfile.server -t cloudbridge-server:$(VERSION) .
	docker build -f deploy/Dockerfile.agent -t cloudbridge-agent:$(VERSION) .

# Run server locally (dev mode)
run-server:
	go run ./cmd/server --debug --signal-addr :10980 --stun-addr :10978 --relay-addr :10988

# Run agent locally (dev mode)
run-agent:
	go run ./cmd/agent --server ws://localhost:10980/signal --name dev-agent

# Generate protobuf (if needed later)
proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto