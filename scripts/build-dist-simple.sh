#!/bin/bash

# js8d Simple Distribution Build Script
# Compatible with older bash versions (macOS default)

set -e

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

# Platform list (simple arrays instead of associative)
PLATFORMS="linux-amd64 linux-arm64 linux-arm linux-arm6 darwin-amd64 darwin-arm64"

check_dependencies() {
    log_info "Checking build dependencies..."

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    GO_VERSION=$(go version | cut -d' ' -f3)
    log_info "Using Go version: $GO_VERSION"

    # Check required files
    local required_files="README.md LICENSE Makefile go.mod"
    for file in $required_files; do
        if [[ ! -f "$PROJECT_DIR/$file" ]]; then
            log_error "Required file not found: $file"
            exit 1
        fi
    done

    log_info "All dependencies satisfied"
}

build_platform() {
    local platform=$1
    local package_dir="$DIST_DIR/js8d-$platform"

    log_section "Building $platform"

    # Set platform-specific variables
    case $platform in
        linux-amd64)
            GOOS="linux"
            GOARCH="amd64"
            GOARM=""
            ;;
        linux-arm64)
            GOOS="linux"
            GOARCH="arm64"
            GOARM=""
            ;;
        linux-arm)
            GOOS="linux"
            GOARCH="arm"
            GOARM="7"
            ;;
        linux-arm6)
            GOOS="linux"
            GOARCH="arm"
            GOARM="6"
            ;;
        darwin-amd64)
            GOOS="darwin"
            GOARCH="amd64"
            GOARM=""
            ;;
        darwin-arm64)
            GOOS="darwin"
            GOARCH="arm64"
            GOARM=""
            ;;
        *)
            log_error "Unknown platform: $platform"
            return 1
            ;;
    esac

    log_info "Building for $GOOS/$GOARCH${GOARM:+ (ARM v$GOARM)}"

    # Create package structure
    mkdir -p "$package_dir"/{bin,configs,docs,web,scripts}

    # Build flags
    local ldflags="-X main.Version=$VERSION -X main.Build=$(date -u '+%Y-%m-%d_%H:%M:%S')"

    # Build main daemon
    log_info "Building js8d daemon..."
    if [[ -n "$GOARM" ]]; then
        GOOS="$GOOS" GOARCH="$GOARCH" GOARM="$GOARM" go build -ldflags "$ldflags" -o "$package_dir/bin/js8d" "$PROJECT_DIR/cmd/js8d"
    else
        GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$ldflags" -o "$package_dir/bin/js8d" "$PROJECT_DIR/cmd/js8d"
    fi

    # Build control client
    log_info "Building js8ctl client..."
    if [[ -n "$GOARM" ]]; then
        GOOS="$GOOS" GOARCH="$GOARCH" GOARM="$GOARM" go build -ldflags "$ldflags" -o "$package_dir/bin/js8ctl" "$PROJECT_DIR/cmd/js8ctl"
    else
        GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$ldflags" -o "$package_dir/bin/js8ctl" "$PROJECT_DIR/cmd/js8ctl"
    fi

    # Copy files
    cp "$PROJECT_DIR/README.md" "$PROJECT_DIR/LICENSE" "$package_dir/"
    cp -r "$PROJECT_DIR/configs"/* "$package_dir/configs/"
    cp -r "$PROJECT_DIR/docs"/* "$package_dir/docs/"
    cp -r "$PROJECT_DIR/web"/* "$package_dir/web/"
    cp -r "$PROJECT_DIR/scripts"/* "$package_dir/scripts/"

    # Create installation instructions
    create_install_instructions "$platform" "$package_dir"

    # Set executable permissions
    chmod +x "$package_dir/bin"/*

    # Create tarball
    log_info "Creating tarball..."
    cd "$DIST_DIR"
    tar -czf "js8d-$platform.tar.gz" "js8d-$platform/"

    # Cleanup directory (keep tarball)
    rm -rf "js8d-$platform/"

    local size=$(du -h "js8d-$platform.tar.gz" | cut -f1)
    log_info "Package created: js8d-$platform.tar.gz ($size)"
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
EOF
}

generate_checksums() {
    log_section "Generating checksums"

    cd "$DIST_DIR"

    # Generate SHA256 checksums
    if command -v sha256sum &> /dev/null; then
        sha256sum *.tar.gz > SHA256SUMS
    elif command -v shasum &> /dev/null; then
        shasum -a 256 *.tar.gz > SHA256SUMS
    else
        log_warn "No SHA256 utility found - skipping checksums"
        return
    fi

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
        if [[ -f "$file" ]]; then
            local size=$(du -h "$file" | cut -f1)
            echo "- $file ($size)" >> RELEASE_NOTES.txt
        fi
    done

    echo "" >> RELEASE_NOTES.txt
    echo "SHA256 Checksums:" >> RELEASE_NOTES.txt
    echo "----------------" >> RELEASE_NOTES.txt
    if [[ -f "SHA256SUMS" ]]; then
        cat SHA256SUMS >> RELEASE_NOTES.txt
    fi

    log_info "Generated checksums and release notes"
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
    local total_size=$(du -sh . | cut -f1)
    echo "Total size: $total_size"
}

# Main build process
main() {
    log_section "js8d Distribution Builder"
    echo "Version: $VERSION"
    echo ""

    # Change to project directory
    cd "$PROJECT_DIR"

    # Check dependencies
    check_dependencies

    # Clean and create dist directory
    rm -rf "$DIST_DIR"
    mkdir -p "$DIST_DIR"

    # Build platforms
    if [[ $# -eq 0 ]]; then
        # Build all platforms
        for platform in $PLATFORMS; do
            build_platform "$platform"
        done
    else
        # Build specific platform
        local platform="$1"
        local found=false
        for p in $PLATFORMS; do
            if [[ "$p" == "$platform" ]]; then
                found=true
                break
            fi
        done

        if [[ "$found" == "false" ]]; then
            log_error "Unknown platform: $platform"
            echo "Available platforms: $PLATFORMS"
            exit 1
        fi

        build_platform "$platform"
    fi

    # Generate checksums and release notes
    generate_checksums

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
        for platform in $PLATFORMS; do
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
    *)
        # Run main function with all arguments
        main "$@"
        ;;
esac