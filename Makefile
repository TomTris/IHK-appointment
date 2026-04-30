.PHONY: build run clean test

# Build the application
build:
	go build -o bin/ihk-watcher ./cmd/ihk-watcher

# Run the application with default settings
run:
	go run ./cmd/ihk-watcher

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f termine.md appointments.log

# Run tests (if any)
test:
	go test ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/ihk-watcher-linux-amd64 ./cmd/ihk-watcher
	GOOS=darwin GOARCH=amd64 go build -o bin/ihk-watcher-darwin-amd64 ./cmd/ihk-watcher
	GOOS=darwin GOARCH=arm64 go build -o bin/ihk-watcher-darwin-arm64 ./cmd/ihk-watcher
	GOOS=windows GOARCH=amd64 go build -o bin/ihk-watcher-windows-amd64.exe ./cmd/ihk-watcher