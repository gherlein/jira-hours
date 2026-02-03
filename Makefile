.DEFAULT_GOAL := help

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_DIR := $(dir $(MKFILE_PATH))

.PHONY: help build install test clean deps run check fmt vet
.PHONY: validate-test run-test dry-run all build-all demo test-features

help: ## Display this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

all: deps build ## Install dependencies and build

deps: ## Download and tidy Go dependencies
	@echo "Downloading dependencies..."
	@cd $(PROJECT_DIR) && go mod download
	@echo "Dependencies ready"

build: ## Build the jira-hours CLI binary
	@echo "Building jira-hours..."
	@cd $(PROJECT_DIR) && go build -o bin/jira-hours ./cmd/hours
	@echo "✓ Built: bin/jira-hours"

install: ## Install jira-hours to GOPATH/bin
	@echo "Installing jira-hours..."
	@cd $(PROJECT_DIR) && go install ./cmd/hours
	@echo "✓ Installed to GOPATH/bin"

test: ## Run Go tests
	@go test -v ./...

fmt: ## Format Go code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

check: fmt vet test ## Run formatting, vetting, and tests

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ dist/
	@echo "✓ Cleaned"

validate-test: build ## Validate test data file (Feb 2026)
	@cd $(PROJECT_DIR) && ./bin/jira-hours validate --month 2026-02

validate-jan: build ## Validate January 2026 data
	@cd $(PROJECT_DIR) && ./bin/jira-hours validate --month 2026-01

run-test: build ## Run test with mock client
	@cd $(PROJECT_DIR) && ./bin/jira-hours log --month 2026-02 --mock

dry-run: build ## Dry run for January 2026
	@cd $(PROJECT_DIR) && ./bin/jira-hours log --month 2026-01 --dry-run

run: build ## Run all tests
	@cd $(PROJECT_DIR) && ./test.sh

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@cd $(PROJECT_DIR) && mkdir -p dist
	@cd $(PROJECT_DIR) && GOOS=darwin GOARCH=amd64 go build -o dist/jira-hours-darwin-amd64 ./cmd/hours
	@cd $(PROJECT_DIR) && GOOS=darwin GOARCH=arm64 go build -o dist/jira-hours-darwin-arm64 ./cmd/hours
	@cd $(PROJECT_DIR) && GOOS=linux GOARCH=amd64 go build -o dist/jira-hours-linux-amd64 ./cmd/hours
	@cd $(PROJECT_DIR) && GOOS=windows GOARCH=amd64 go build -o dist/jira-hours-windows-amd64.exe ./cmd/hours
	@echo "✓ Built all platforms in dist/"

demo: build ## Run workflow demonstration
	@cd $(PROJECT_DIR) && go run demo-workflow.go

test-features: build ## Test delete and idempotent features
	@cd $(PROJECT_DIR) && ./test-features.sh

final-demo: build ## Run complete feature demonstration
	@cd $(PROJECT_DIR) && ./final-demo.sh
