.PHONY: test clean setup tidy deps help

# Default target
all: test

# Run the tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/

# Setup .env file if it doesn't exist
setup:
	@if [ ! -f .env ]; then \
		echo "Creating .env file from .env.example..."; \
		cp .env.example .env; \
		echo "Please edit .env file with your API keys"; \
	else \
		echo ".env file already exists"; \
	fi

# Run go mod tidy
tidy:
	@echo "Running go mod tidy..."
	@go mod tidy

# Install go dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

# Help target
help:
	@echo "Available targets:"
	@echo "  all     - Default target, runs tests"
	@echo "  test    - Run the tests"
	@echo "  clean   - Clean build artifacts"
	@echo "  setup   - Setup .env file from .env.example if it doesn't exist"
	@echo "  tidy    - Run go mod tidy"
	@echo "  deps    - Install go dependencies" 