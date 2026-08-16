# Output binary names
BINARY_NAME=arit
ARITD_BINARY_NAME=aritd

# Output directory
BIN_DIR=bin

# Go Flags for Daemon
LDFLAGS=-ldflags="-s -w"

.PHONY: all build build-aritd test clean help

# Default target executed when typing just `make`
all: clean test build build-aritd

## build: Compiles the standard ARIT CLI binary
build:
	@echo "==> Compiling standard ARIT CLI..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "==> CLI binary generated at $(BIN_DIR)/$(BINARY_NAME)"

## build-aritd: Compiles the daemon version (aritd)
build-aritd:
	@echo "==> Compiling daemon version (aritd)..."
	@mkdir -p $(BIN_DIR)
	go build -tags aritd $(LDFLAGS) -o $(BIN_DIR)/$(ARITD_BINARY_NAME) .
	@echo "==> Daemon binary generated at $(BIN_DIR)/$(ARITD_BINARY_NAME)"

## test: Runs all unit tests in the project
test:
	@echo "==> Running unit tests..."
	go test -v ./...

## clean: Removes generated binaries
clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "==> Cleaned."

## help: Displays this help message
help:
	@echo "Available commands in Makefile:"
	@sed -n 's/^##//p' $< | column -t -s ':'
