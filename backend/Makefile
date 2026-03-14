.PHONY: help build generate run clean docker-build docker-up docker-down

help:
	@echo "Threat-Scan System - Available targets:"
	@echo "  make generate       - Generate protobuf code"
	@echo "  make build          - Build the Go service"
	@echo "  make run            - Run the service locally"
	@echo "  make docker-build   - Build Docker images"
	@echo "  make docker-up      - Start Docker containers"
	@echo "  make docker-down    - Stop Docker containers"
	@echo "  make docker-logs    - View Docker logs"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts"

generate:
	@echo "Installing protoc plugins..."
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/scan.proto

build: generate
	@echo "Building service..."
	go build -o bin/threat-scan-service ./cmd

run: build
	@echo "Running service..."
	./bin/threat-scan-service -config config/config.yaml

docker-build:
	@echo "Building Docker images..."
	docker-compose -f docker-compose.yaml build

docker-up:
	@echo "Starting Docker containers..."
	docker-compose -f docker-compose.yaml up -d

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose -f docker-compose.yaml down

docker-logs:
	docker-compose -f docker-compose.yaml logs -f

docker-ps:
	docker-compose -f docker-compose.yaml ps

test:
	@echo "Running tests..."
	go test ./...

clean:
	@echo "Cleaning artifacts..."
	rm -rf bin/
	go clean
	docker-compose -f docker-compose.yaml down -v

fmt:
	go fmt ./...
	goimports -w .

lint:
	golangci-lint run ./...
