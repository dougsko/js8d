#!/usr/bin/env bash

# js8d Distribution Build Script
# Builds complete distribution packages for all supported platforms

set -e

# Check for bash version 4.0+ (required for associative arrays)
if [ "${BASH_VERSION%%.*}" -lt 4 ]; then
    echo "Error: This script requires bash 4.0 or later"
    echo "Current version: $BASH_VERSION"
    exit 1
fi

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$PROJECT_DIR/dist"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Platform configurations
declare -A PLATFORMS=(
    ["linux-amd64"]="linux amd64"
    ["linux-arm64"]="linux arm64"
    ["linux-arm"]="linux arm"
    ["linux-arm6"]="linux arm"
    ["darwin-amd64"]="darwin amd64"
    ["darwin-arm64"]="darwin arm64"
)

declare -A ARM_VERSIONS=(
    ["linux-arm"]="7"
    ["linux-arm6"]="6"
)

check_dependencies() {
    log_info "Checking build dependencies..."

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    GO_VERSION=$(go version | cut -d' ' -f3)
    log_info "Using Go version: $GO_VERSION"

    # Check git for version info
    if ! command -v git &> /dev/null; then
        log_warn "Git not found - using default version"
    fi

    # Check required files
    local required_files=("README.md" "LICENSE" "Makefile" "go.mod")
    for file in "${required_files[@]}"; do
        if [[ ! -f "$PROJECT_DIR/$file" ]]; then
            log_error "Required file not found: $file"
            exit 1
        fi
    done

    log_info "All dependencies satisfied"
}

create_package_structure() {
    local platform=$1
    local package_dir="$DIST_DIR/js8d-$platform"

    log_info "Creating package structure for $platform..."

    # Create directory structure
    mkdir -p "$package_dir"/{bin,configs,docs,web,scripts}

    # Copy static files
    cp "$PROJECT_DIR/README.md" "$package_dir/"
    cp "$PROJECT_DIR/LICENSE" "$package_dir/"

    # Copy configuration files
    cp -r "$PROJECT_DIR/configs"/* "$package_dir/configs/"

    # Copy documentation
    cp -r "$PROJECT_DIR/docs"/* "$package_dir/docs/"

    # Copy web assets
    cp -r "$PROJECT_DIR/web"/* "$package_dir/web/"

    # Copy scripts
    cp -r "$PROJECT_DIR/scripts"/* "$package_dir/scripts/"

    # Create installation instructions
    create_install_instructions "$platform" "$package_dir"

    log_info "Package structure created for $platform"
}

create_install_instructions() {
    local platform=$1
    local package_dir=$2
    local install_file="$package_dir/INSTALL.txt"

    cat > "$install_file" << EOF
js8d Installation Instructions
=============================

Platform: $platform
Version: $VERSION
Build Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')

Quick Installation:
------------------

1. Extract this package:
   tar -xzf js8d-$platform.tar.gz
   cd js8d-$platform/

2. Install binaries (requires sudo):
   sudo cp bin/js8d /usr/local/bin/
   sudo cp bin/js8ctl /usr/local/bin/
   sudo chmod +x /usr/local/bin/js8d /usr/local/bin/js8ctl

3. Create configuration directory:
   sudo mkdir -p /etc/js8d
   sudo cp configs/config.production.yaml /etc/js8d/config.yaml

4. Edit configuration:
   sudo nano /etc/js8d/config.yaml
   # Update your callsign, grid square, and radio settings

5. Test installation:
   js8d -config /etc/js8d/config.yaml -version

For detailed installation instructions, see docs/INSTALLATION.md

Systemd Service Installation (Linux):
------------------------------------

1. Follow steps 1-4 above

2. Install as systemd service:
   sudo ./scripts/install.sh

3. Enable and start service:
   sudo systemctl enable js8d.service
   sudo systemctl start js8d.service

4. Check status:
   sudo systemctl status js8d.service

5. View logs:
   sudo journalctl -u js8d.service -f

Web Interface:
-------------
After starting js8d, access the web interface at:
http://localhost:8080

For remote access, configure your firewall to allow port 8080.

Documentation:
-------------
- Installation Guide: docs/INSTALLATION.md
- Configuration Reference: docs/CONFIGURATION.md
- API Documentation: docs/API.md
- Troubleshooting: docs/TROUBLESHOOTING.md

Support:
-------
- GitHub: https://github.com/dougsko/js8d
- Issues: https://github.com/dougsko/js8d/issues
- Documentation: https://github.com/dougsko/js8d/tree/main/docs

EOF
}

build_platform() {
    local platform=$1
    local goos=${PLATFORMS[$platform]%% *}
    local goarch=${PLATFORMS[$platform]##* }
    local package_dir="$DIST_DIR/js8d-$platform"

    log_section "Building $platform ($goos/$goarch)"

    # Set ARM version if needed
    local goarm=""
    if [[ -n "${ARM_VERSIONS[$platform]:-}" ]]; then
        goarm="${ARM_VERSIONS[$platform]}"
        export GOARM="$goarm"
        log_info "Using GOARM=$goarm for $platform"
    fi

    # Build flags
    local ldflags="-X main.Version=$VERSION -X main.Build=$(date -u '+%Y-%m-%d_%H:%M:%S')"

    # Create package structure
    create_package_structure "$platform"

    # Build main daemon
    log_info "Building js8d daemon..."
    GOOS="$goos" GOARCH="$goarch" go build -ldflags "$ldflags" -o "$package_dir/bin/js8d" "$PROJECT_DIR/cmd/js8d"

    # Build control client
    log_info "Building js8ctl client..."
    GOOS="$goos" GOARCH="$goarch" go build -ldflags "$ldflags" -o "$package_dir/bin/js8ctl" "$PROJECT_DIR/cmd/js8ctl"

    # Build encoding utility if it exists
    if [[ -d "$PROJECT_DIR/cmd/js8encode" ]]; then
        log_info "Building js8encode utility..."
        GOOS="$goos" GOARCH="$goarch" go build -ldflags "$ldflags" -o "$package_dir/bin/js8encode" "$PROJECT_DIR/cmd/js8encode"
    fi

    # Set executable permissions
    chmod +x "$package_dir/bin"/*

    # Create tarball
    log_info "Creating tarball..."
    cd "$DIST_DIR"
    tar -czf "js8d-$platform.tar.gz" "js8d-$platform/"

    # Cleanup directory (keep tarball)
    rm -rf "js8d-$platform/"

    log_info "Package created: js8d-$platform.tar.gz"
}

generate_checksums() {
    log_section "Generating checksums"

    cd "$DIST_DIR"

    # Generate SHA256 checksums
    sha256sum *.tar.gz > SHA256SUMS

    # Generate release notes
    cat > RELEASE_NOTES.txt << EOF
js8d Release $VERSION
====================

Build Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Go Version: $(go version | cut -d' ' -f3)

Packages:
--------
EOF

    for file in *.tar.gz; do
        local size=$(du -h "$file" | cut -f1)
        echo "- $file ($size)" >> RELEASE_NOTES.txt
    done

    echo "" >> RELEASE_NOTES.txt
    echo "SHA256 Checksums:" >> RELEASE_NOTES.txt
    echo "----------------" >> RELEASE_NOTES.txt
    cat SHA256SUMS >> RELEASE_NOTES.txt

    log_info "Generated SHA256SUMS and RELEASE_NOTES.txt"
}

create_source_package() {
    log_section "Creating source package"

    local source_dir="$DIST_DIR/js8d-$VERSION-src"

    # Create source directory
    mkdir -p "$source_dir"

    # Copy source files (exclude build artifacts and git)
    rsync -av --exclude='.git' --exclude='dist/' --exclude='*.log' \
          --exclude='js8d' --exclude='js8ctl' --exclude='coverage.*' \
          "$PROJECT_DIR/" "$source_dir/"

    # Create source tarball
    cd "$DIST_DIR"
    tar -czf "js8d-$VERSION-src.tar.gz" "js8d-$VERSION-src/"
    rm -rf "js8d-$VERSION-src/"

    log_info "Source package created: js8d-$VERSION-src.tar.gz"
}

verify_packages() {
    log_section "Verifying packages"

    cd "$DIST_DIR"

    # Verify checksums
    if sha256sum -c SHA256SUMS; then
        log_info "All checksums verified successfully"
    else
        log_error "Checksum verification failed"
        exit 1
    fi

    # Test extract one package
    local test_package=$(ls js8d-linux-amd64.tar.gz 2>/dev/null | head -1)
    if [[ -n "$test_package" ]]; then
        log_info "Testing package extraction: $test_package"
        local test_dir="test_extract"
        mkdir -p "$test_dir"
        cd "$test_dir"
        tar -xzf "../$test_package"

        # Check if binaries exist and are executable
        local extracted_dir=$(ls -d js8d-*/ | head -1)
        if [[ -x "$extracted_dir/bin/js8d" && -x "$extracted_dir/bin/js8ctl" ]]; then
            log_info "Package extraction test passed"
        else
            log_error "Package extraction test failed - binaries not found or executable"
            exit 1
        fi

        cd ..
        rm -rf "$test_dir"
    fi
}

show_summary() {
    log_section "Build Summary"

    echo "Version: $VERSION"
    echo "Build completed at: $(date)"
    echo ""
    echo "Packages created in $DIST_DIR:"

    cd "$DIST_DIR"
    for file in *.tar.gz; do
        if [[ -f "$file" ]]; then
            local size=$(du -h "$file" | cut -f1)
            echo "  $file ($size)"
        fi
    done

    echo ""
    echo "Total size: $(du -sh . | cut -f1)"
    echo ""
    echo "Next steps:"
    echo "1. Test packages on target platforms"
    echo "2. Upload to release repository"
    echo "3. Update documentation with new version"
    echo ""
    echo "Files ready for distribution:"
    echo "- Packages: *.tar.gz"
    echo "- Checksums: SHA256SUMS"
    echo "- Release notes: RELEASE_NOTES.txt"
}

# Main build process
main() {
    log_section "js8d Distribution Builder"
    echo "Version: $VERSION"
    echo "Project: $PROJECT_DIR"
    echo "Output: $DIST_DIR"
    echo ""

    # Change to project directory
    cd "$PROJECT_DIR"

    # Check dependencies
    check_dependencies

    # Clean and create dist directory
    rm -rf "$DIST_DIR"
    mkdir -p "$DIST_DIR"

    # Run tests first
    log_section "Running tests"
    if ! go test ./...; then
        log_error "Tests failed - aborting build"
        exit 1
    fi
    log_info "All tests passed"

    # Format and vet code
    log_info "Formatting and vetting code..."
    go fmt ./...
    go vet ./...

    # Build all platforms
    for platform in "${!PLATFORMS[@]}"; do
        build_platform "$platform"
    done

    # Create source package
    create_source_package

    # Generate checksums and release notes
    generate_checksums

    # Verify packages
    verify_packages

    # Show summary
    show_summary

    log_info "Distribution build completed successfully!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "js8d Distribution Builder"
        echo ""
        echo "Usage: $0 [platform]"
        echo ""
        echo "Platforms:"
        for platform in "${!PLATFORMS[@]}"; do
            echo "  $platform"
        done
        echo ""
        echo "Environment variables:"
        echo "  VERSION    - Override version (default: git describe)"
        echo ""
        echo "Examples:"
        echo "  $0                    # Build all platforms"
        echo "  $0 linux-arm64      # Build specific platform"
        echo "  VERSION=1.0.0 $0    # Build with specific version"
        exit 0
        ;;
    "")
        # Build all platforms
        main
        ;;
    *)
        # Build specific platform
        platform="$1"
        if [[ -z "${PLATFORMS[$platform]:-}" ]]; then
            log_error "Unknown platform: $platform"
            echo "Available platforms: ${!PLATFORMS[*]}"
            exit 1
        fi

        log_section "Building single platform: $platform"
        check_dependencies
        rm -rf "$DIST_DIR"
        mkdir -p "$DIST_DIR"
        build_platform "$platform"
        log_info "Single platform build completed!"
        ;;
esac