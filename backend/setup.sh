#!/bin/bash

# Script to set up the threat-scan development environment

set -e

echo "Setting up threat-scan development environment..."

# Check prerequisites
echo "Checking prerequisites..."
command -v go >/dev/null 2>&1 || { echo "Go is required but not installed."; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed."; exit 1; }
command -v protoc >/dev/null 2>&1 || { echo "protoc is required but not installed."; exit 1; }

echo "✓ Go, Docker, and protoc are installed"

# Download Go dependencies
echo "Downloading Go dependencies..."
go mod download
go mod tidy

# Generate protobuf code
echo "Generating protobuf code..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/scan.proto

echo "✓ Protobuf code generated"

# Build Go service
echo "Building Go service..."
mkdir -p bin
go build -o bin/threat-scan-service ./cmd

echo "✓ Go service built successfully"

# Build Docker images
echo "Building Docker images..."
docker-compose build

echo "✓ Docker images built successfully"

echo ""
echo "Setup complete! To start the system:"
echo ""
echo "  docker-compose up -d"
echo ""
echo "Check status with:"
echo "  docker-compose ps"
echo "  docker-compose logs -f"
echo ""
