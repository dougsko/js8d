# js8d Makefile

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Go build flags
LDFLAGS := -X main.Version=$(VERSION) -X main.Build=$(BUILD_TIME)
GOFLAGS := -ldflags "$(LDFLAGS)"

# Targets
BINARY_NAME := js8d
MAIN_PACKAGE := ./cmd/js8d

# Default target
.PHONY: all
all: build

# Build the daemon
.PHONY: build
build:
	@echo "Building js8d..."
	go build $(GOFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Build for different platforms
.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Building for Linux ARM64 (Pi 4)..."
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)

.PHONY: build-linux-arm
build-linux-arm:
	@echo "Building for Linux ARM (Pi 3)..."
	GOOS=linux GOARCH=arm GOARM=7 go build $(GOFLAGS) -o $(BINARY_NAME)-linux-arm $(MAIN_PACKAGE)

.PHONY: build-linux-arm6
build-linux-arm6:
	@echo "Building for Linux ARM6 (Pi Zero)..."
	GOOS=linux GOARCH=arm GOARM=6 go build $(GOFLAGS) -o $(BINARY_NAME)-linux-arm6 $(MAIN_PACKAGE)

.PHONY: build-all
build-all: build build-linux-arm64 build-linux-arm build-linux-arm6

# Development targets
.PHONY: run
run: build
	./$(BINARY_NAME) -config configs/config.example.yaml

.PHONY: dev
dev:
	go run $(MAIN_PACKAGE) -config configs/config.example.yaml

# Testing
.PHONY: test
test:
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Code quality
.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

# Dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: deps-update
deps-update:
	go get -u ./...
	go mod tidy

# DSP library (TODO: implement)
.PHONY: build-dsp
build-dsp:
	@echo "Building DSP library..."
	cd libjs8dsp && mkdir -p build && cd build && cmake .. && make

# Clean
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)*
	rm -f coverage.out coverage.html
	rm -rf libjs8dsp/build

# Install
.PHONY: install
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo mkdir -p /etc/js8d
	sudo cp configs/config.example.yaml /etc/js8d/
	@echo "js8d installed to /usr/local/bin/"
	@echo "Example config copied to /etc/js8d/config.example.yaml"

# Uninstall
.PHONY: uninstall
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "js8d uninstalled"

# Docker targets
.PHONY: docker-build
docker-build:
	docker build -t js8d:$(VERSION) .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/configs:/etc/js8d js8d:$(VERSION)

# Help
.PHONY: help
help:
	@echo "js8d Makefile targets:"
	@echo ""
	@echo "  build          - Build js8d binary"
	@echo "  build-all      - Build for all supported platforms"
	@echo "  run            - Build and run with example config"
	@echo "  dev            - Run in development mode"
	@echo "  test           - Run tests"
	@echo "  fmt            - Format Go code"
	@echo "  vet            - Run go vet"
	@echo "  deps           - Download dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install to system"
	@echo "  help           - Show this help"