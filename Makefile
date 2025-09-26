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

# DSP functionality is now implemented in pure Go - no separate library needed

# Clean
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)*
	rm -f coverage.out coverage.html

# Simple install (development)
.PHONY: install
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo mkdir -p /etc/js8d
	sudo cp configs/config.example.yaml /etc/js8d/
	@echo "js8d installed to /usr/local/bin/"
	@echo "Example config copied to /etc/js8d/config.example.yaml"

# Production install with systemd service
.PHONY: install-service
install-service: build
	@echo "Installing js8d as systemd service..."
	./scripts/install.sh

# Uninstall simple installation
.PHONY: uninstall
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "js8d uninstalled"

# Uninstall systemd service installation
.PHONY: uninstall-service
uninstall-service:
	@echo "Uninstalling js8d systemd service..."
	sudo ./scripts/uninstall.sh

# Docker targets
.PHONY: docker-build
docker-build:
	docker build -t js8d:$(VERSION) .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/configs:/etc/js8d js8d:$(VERSION)

# Distribution packages
DIST_DIR := dist
PACKAGE_FILES := README.md LICENSE configs/ docs/ web/ scripts/

.PHONY: dist-clean
dist-clean:
	rm -rf $(DIST_DIR)

.PHONY: dist-prepare
dist-prepare: dist-clean
	mkdir -p $(DIST_DIR)

.PHONY: dist-linux-amd64
dist-linux-amd64: dist-prepare
	@echo "Building Linux AMD64 distribution package..."
	mkdir -p $(DIST_DIR)/js8d-linux-amd64/{bin,configs,docs,web,scripts}
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-amd64/bin/js8d $(MAIN_PACKAGE)
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-amd64/bin/js8ctl ./cmd/js8ctl
	cp README.md LICENSE $(DIST_DIR)/js8d-linux-amd64/
	cp -r configs/* $(DIST_DIR)/js8d-linux-amd64/configs/
	cp -r docs/* $(DIST_DIR)/js8d-linux-amd64/docs/
	cp -r web/* $(DIST_DIR)/js8d-linux-amd64/web/
	cp -r scripts/* $(DIST_DIR)/js8d-linux-amd64/scripts/
	cd $(DIST_DIR) && tar -czf js8d-linux-amd64.tar.gz js8d-linux-amd64/

.PHONY: dist-linux-arm64
dist-linux-arm64: dist-prepare
	@echo "Building Linux ARM64 distribution package..."
	mkdir -p $(DIST_DIR)/js8d-linux-arm64/{bin,configs,docs,web,scripts}
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-arm64/bin/js8d $(MAIN_PACKAGE)
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-arm64/bin/js8ctl ./cmd/js8ctl
	cp README.md LICENSE $(DIST_DIR)/js8d-linux-arm64/
	cp -r configs/* $(DIST_DIR)/js8d-linux-arm64/configs/
	cp -r docs/* $(DIST_DIR)/js8d-linux-arm64/docs/
	cp -r web/* $(DIST_DIR)/js8d-linux-arm64/web/
	cp -r scripts/* $(DIST_DIR)/js8d-linux-arm64/scripts/
	cd $(DIST_DIR) && tar -czf js8d-linux-arm64.tar.gz js8d-linux-arm64/

.PHONY: dist-linux-arm
dist-linux-arm: dist-prepare
	@echo "Building Linux ARM distribution package..."
	mkdir -p $(DIST_DIR)/js8d-linux-arm/{bin,configs,docs,web,scripts}
	GOOS=linux GOARCH=arm GOARM=7 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-arm/bin/js8d $(MAIN_PACKAGE)
	GOOS=linux GOARCH=arm GOARM=7 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-arm/bin/js8ctl ./cmd/js8ctl
	cp README.md LICENSE $(DIST_DIR)/js8d-linux-arm/
	cp -r configs/* $(DIST_DIR)/js8d-linux-arm/configs/
	cp -r docs/* $(DIST_DIR)/js8d-linux-arm/docs/
	cp -r web/* $(DIST_DIR)/js8d-linux-arm/web/
	cp -r scripts/* $(DIST_DIR)/js8d-linux-arm/scripts/
	cd $(DIST_DIR) && tar -czf js8d-linux-arm.tar.gz js8d-linux-arm/

.PHONY: dist-linux-arm6
dist-linux-arm6: dist-prepare
	@echo "Building Linux ARM6 distribution package..."
	mkdir -p $(DIST_DIR)/js8d-linux-arm6/{bin,configs,docs,web,scripts}
	GOOS=linux GOARCH=arm GOARM=6 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-arm6/bin/js8d $(MAIN_PACKAGE)
	GOOS=linux GOARCH=arm GOARM=6 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-linux-arm6/bin/js8ctl ./cmd/js8ctl
	cp README.md LICENSE $(DIST_DIR)/js8d-linux-arm6/
	cp -r configs/* $(DIST_DIR)/js8d-linux-arm6/configs/
	cp -r docs/* $(DIST_DIR)/js8d-linux-arm6/docs/
	cp -r web/* $(DIST_DIR)/js8d-linux-arm6/web/
	cp -r scripts/* $(DIST_DIR)/js8d-linux-arm6/scripts/
	cd $(DIST_DIR) && tar -czf js8d-linux-arm6.tar.gz js8d-linux-arm6/

.PHONY: dist-darwin-amd64
dist-darwin-amd64: dist-prepare
	@echo "Building macOS AMD64 distribution package..."
	mkdir -p $(DIST_DIR)/js8d-darwin-amd64/{bin,configs,docs,web,scripts}
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-darwin-amd64/bin/js8d $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-darwin-amd64/bin/js8ctl ./cmd/js8ctl
	cp README.md LICENSE $(DIST_DIR)/js8d-darwin-amd64/
	cp -r configs/* $(DIST_DIR)/js8d-darwin-amd64/configs/
	cp -r docs/* $(DIST_DIR)/js8d-darwin-amd64/docs/
	cp -r web/* $(DIST_DIR)/js8d-darwin-amd64/web/
	cp -r scripts/* $(DIST_DIR)/js8d-darwin-amd64/scripts/
	cd $(DIST_DIR) && tar -czf js8d-darwin-amd64.tar.gz js8d-darwin-amd64/

.PHONY: dist-darwin-arm64
dist-darwin-arm64: dist-prepare
	@echo "Building macOS ARM64 distribution package..."
	mkdir -p $(DIST_DIR)/js8d-darwin-arm64/{bin,configs,docs,web,scripts}
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-darwin-arm64/bin/js8d $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/js8d-darwin-arm64/bin/js8ctl ./cmd/js8ctl
	cp README.md LICENSE $(DIST_DIR)/js8d-darwin-arm64/
	cp -r configs/* $(DIST_DIR)/js8d-darwin-arm64/configs/
	cp -r docs/* $(DIST_DIR)/js8d-darwin-arm64/docs/
	cp -r web/* $(DIST_DIR)/js8d-darwin-arm64/web/
	cp -r scripts/* $(DIST_DIR)/js8d-darwin-arm64/scripts/
	cd $(DIST_DIR) && tar -czf js8d-darwin-arm64.tar.gz js8d-darwin-arm64/

.PHONY: dist-all
dist-all: dist-linux-amd64 dist-linux-arm64 dist-linux-arm dist-linux-arm6 dist-darwin-amd64 dist-darwin-arm64
	@echo "All distribution packages built in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/*.tar.gz

.PHONY: dist-checksums
dist-checksums:
	@echo "Generating checksums..."
	cd $(DIST_DIR) && sha256sum *.tar.gz > SHA256SUMS
	@echo "Checksums generated in $(DIST_DIR)/SHA256SUMS"

# Release preparation
.PHONY: release
release: test fmt vet dist-all dist-checksums
	@echo "Release packages ready in $(DIST_DIR)/"

# Help
.PHONY: help
help:
	@echo "js8d Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build          - Build js8d binary for current platform"
	@echo "  build-all      - Build for all supported platforms"
	@echo ""
	@echo "Distribution targets:"
	@echo "  dist-linux-amd64   - Build Linux AMD64 distribution package"
	@echo "  dist-linux-arm64   - Build Linux ARM64 distribution package"
	@echo "  dist-linux-arm     - Build Linux ARM distribution package"
	@echo "  dist-linux-arm6    - Build Linux ARM6 distribution package"
	@echo "  dist-darwin-amd64  - Build macOS AMD64 distribution package"
	@echo "  dist-darwin-arm64  - Build macOS ARM64 distribution package"
	@echo "  dist-all           - Build all distribution packages"
	@echo "  dist-checksums     - Generate SHA256 checksums for packages"
	@echo "  release            - Complete release build (test + dist + checksums)"
	@echo ""
	@echo "Development targets:"
	@echo "  run            - Build and run with example config"
	@echo "  dev            - Run in development mode"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo ""
	@echo "Code quality:"
	@echo "  fmt            - Format Go code"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo ""
	@echo "Installation:"
	@echo "  install        - Simple install to system"
	@echo "  install-service - Production install with systemd service"
	@echo "  uninstall      - Remove simple installation"
	@echo "  uninstall-service - Remove systemd service installation"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean          - Clean build artifacts"
	@echo "  dist-clean     - Clean distribution packages"
	@echo ""
	@echo "Other:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  help           - Show this help"