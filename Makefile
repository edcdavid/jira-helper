APP_NAME := jira-helper
BUILD_DIR := build
BIN := $(BUILD_DIR)/$(APP_NAME)

.PHONY: all build run clean fmt lint deps

all: build

build:
	@echo "🔨 Building $(APP_NAME)..."
	@go build -o $(BIN) main.go

run: build
	@echo "🚀 Running $(APP_NAME)..."
	@$(BIN)

clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(BUILD_DIR)

fmt:
	@echo "🧼 Formatting code..."
	@go fmt ./...

lint:
	@echo "🔍 Linting code..."
	@golangci-lint run

deps:
	@echo "📥 Getting dependencies..."
	@go mod tidy
