BINARY_NAME=qaitor
VERSION=1.0.0
BUILD_DIR=bin
MODULE=github.com/agromai/qaitor

LDFLAGS=-ldflags "-X main.Version=$(VERSION) -s -w"

.PHONY: all build build-linux build-windows run install clean tidy

all: build

## build: Build for the current platform
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

## build-linux: Cross-compile for Linux (amd64)
build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

## build-windows: Cross-compile for Windows (amd64)
build-windows:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

## build-all: Build for all supported platforms
build-all: build build-linux build-windows

## run: Build and run QAITOR
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

## install: Install QAITOR to GOPATH/bin
install:
	go install $(LDFLAGS) .

## tidy: Run go mod tidy
tidy:
	go mod tidy

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## help: Show this help
help:
	@echo "QAITOR Build System"
	@echo "==================="
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
