.PHONY: build clean test version

# Build variables
BINARY_NAME=awsinv
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_SHA?=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Ensure COMMIT_SHA doesn't contain spaces
COMMIT_SHA_CLEAN=$(shell echo "$(COMMIT_SHA)" | tr -d ' ')

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.CommitSHA=$(COMMIT_SHA_CLEAN) -X main.BuildDate=$(BUILD_DATE)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_SHA)"
	@echo "Build Date: $(BUILD_DATE)"
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/awsinv

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/awsinv

build-darwin:
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/awsinv
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/awsinv

build-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/awsinv

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BINARY_NAME) $(shell go env GOPATH)/bin/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_SHA)"
	@echo "Build Date: $(BUILD_DATE)"

# Run the binary
run: build
	./$(BINARY_NAME)

# Run with example flags
run-example: build
	./$(BINARY_NAME) --services ec2,rds --regions us-east-1 --output json --verbose

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Generate documentation
docs:
	@echo "Generating documentation..."
	@echo "# AWS Inventory Tool (awsinv)" > README.md
	@echo "" >> README.md
	@echo "A production-quality Go CLI tool for inventorying active AWS resources across regions." >> README.md
	@echo "" >> README.md
	@echo "## Features" >> README.md
	@echo "" >> README.md
	@echo "- Enumerates AWS services across all enabled regions" >> README.md
	@echo "- Normalizes resources into a unified Resource model" >> README.md
	@echo "- Outputs results in table, JSON, or CSV formats" >> README.md
	@echo "- Fast, parallel, and robust with graceful error handling" >> README.md
	@echo "- Minimal external dependencies" >> README.md
	@echo "- Compiles as a static binary" >> README.md
	@echo "" >> README.md
	@echo "## Supported Services" >> README.md
	@echo "" >> README.md
	@echo "- EC2 instances" >> README.md
	@echo "- RDS database instances" >> README.md
	@echo "- Lambda functions" >> README.md
	@echo "- S3 buckets" >> README.md
	@echo "" >> README.md
	@echo "## Usage" >> README.md
	@echo "" >> README.md
	@echo "\`\`\`bash" >> README.md
	@echo "# Basic usage" >> README.md
	@echo "./awsinv" >> README.md
	@echo "" >> README.md
	@echo "# Specific services and regions" >> README.md
	@echo "./awsinv --services ec2,rds --regions us-east-1,us-west-2" >> README.md
	@echo "" >> README.md
	@echo "# JSON output with filtering" >> README.md
	@echo "./awsinv --output json --filter state=running --filter name=prod*" >> README.md
	@echo "" >> README.md
	@echo "# Verbose output with role assumption" >> README.md
	@echo "./awsinv --verbose --role-arn arn:aws:iam::123456789012:role/InventoryRole" >> README.md
	@echo "\`\`\`" >> README.md

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms (Linux, macOS, Windows)"
	@echo "  install        - Install the binary to GOPATH/bin"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  version        - Show version information"
	@echo "  run            - Build and run the binary"
	@echo "  run-example    - Run with example flags"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code (requires golangci-lint)"
	@echo "  docs           - Generate documentation"
	@echo "  help           - Show this help" 