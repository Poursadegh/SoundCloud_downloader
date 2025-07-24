.PHONY: all clean proto server client help

# Default target
all: proto server client

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/downloader.proto

# Build server
server: proto
	@echo "Building server..."
	go build -o bin/server server/server.go

# Build client
client: proto
	@echo "Building client..."
	go build -o bin/client client/client.go

# Run server
run-server: server
	@echo "Starting server..."
	./bin/server

# Run client (example)
run-client: client
	@echo "Starting client..."
	./bin/client

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f proto/*.pb.go

# Create bin directory
bin:
	mkdir -p bin

# Help
help:
	@echo "Available targets:"
	@echo "  all        - Build everything (proto, server, client)"
	@echo "  proto      - Generate protobuf files"
	@echo "  server     - Build server"
	@echo "  client     - Build client"
	@echo "  run-server - Build and run server"
	@echo "  run-client - Build and run client"
	@echo "  deps       - Install dependencies"
	@echo "  clean      - Clean build artifacts"
	@echo "  help       - Show this help" 