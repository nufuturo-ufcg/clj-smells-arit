# Output binary names
BINARY_NAME=arit
ARITD_BINARY_NAME=aritd

# Output directory
BIN_DIR=bin

# Go Flags for Daemon
LDFLAGS=-ldflags="-s -w"

.PHONY: all build build-aritd test baseline-check clean help

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

## baseline-check: Compares normal and cross-namespace JSON reports
baseline-check:
	@test -n "$(BASELINE_NORMAL)" && test -n "$(CURRENT_NORMAL)" && test -n "$(BASELINE_CROSS)" && test -n "$(CURRENT_CROSS)" || (echo "Use BASELINE_NORMAL=... CURRENT_NORMAL=... BASELINE_CROSS=... CURRENT_CROSS=..."; exit 2)
	tools/check_baseline.sh "$(BASELINE_NORMAL)" "$(CURRENT_NORMAL)" "$(BASELINE_CROSS)" "$(CURRENT_CROSS)"

## clean: Removes generated binaries
clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "==> Cleaned."

## help: Displays this help message
help:
	@echo "Available commands in Makefile:"
	@sed -n 's/^##//p' $< | column -t -s ':'
