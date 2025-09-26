#!/bin/bash

# js8d Installation Script
# Installs js8d daemon as a systemd service

set -e

# Configuration
JS8D_USER="js8d"
JS8D_GROUP="js8d"
JS8D_CONFIG_DIR="/etc/js8d"
JS8D_DATA_DIR="/var/lib/js8d"
JS8D_LOG_DIR="/var/log/js8d"
JS8D_RUN_DIR="/run/js8d"
SYSTEMD_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_dependencies() {
    log_info "Checking dependencies..."

    # Check for required commands
    local missing_deps=()

    for cmd in systemctl useradd groupadd; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        exit 1
    fi

    # Check for optional dependencies
    if ! command -v hamlib-utils &> /dev/null; then
        log_warn "hamlib-utils not found - radio control may not work"
        log_warn "Install with: apt-get install libhamlib-utils (Debian/Ubuntu)"
    fi

    if ! command -v alsa-utils &> /dev/null; then
        log_warn "alsa-utils not found - audio may not work properly"
        log_warn "Install with: apt-get install alsa-utils (Debian/Ubuntu)"
    fi
}

create_user() {
    log_info "Creating js8d user and group..."

    # Create group if it doesn't exist
    if ! getent group "$JS8D_GROUP" &> /dev/null; then
        groupadd --system "$JS8D_GROUP"
        log_info "Created group: $JS8D_GROUP"
    else
        log_info "Group $JS8D_GROUP already exists"
    fi

    # Create user if it doesn't exist
    if ! getent passwd "$JS8D_USER" &> /dev/null; then
        useradd --system \
                --gid "$JS8D_GROUP" \
                --home-dir "$JS8D_DATA_DIR" \
                --no-create-home \
                --shell /usr/sbin/nologin \
                --comment "JS8 Digital Mode Daemon" \
                "$JS8D_USER"
        log_info "Created user: $JS8D_USER"
    else
        log_info "User $JS8D_USER already exists"
    fi

    # Add user to audio and dialout groups for hardware access
    usermod -a -G audio,dialout "$JS8D_USER" || log_warn "Could not add $JS8D_USER to audio/dialout groups"
}

create_directories() {
    log_info "Creating directories..."

    # Create directories with proper permissions
    mkdir -p "$JS8D_CONFIG_DIR"
    mkdir -p "$JS8D_DATA_DIR"
    mkdir -p "$JS8D_LOG_DIR"
    mkdir -p "$JS8D_RUN_DIR"

    # Set ownership and permissions
    chown root:root "$JS8D_CONFIG_DIR"
    chmod 755 "$JS8D_CONFIG_DIR"

    chown "$JS8D_USER:$JS8D_GROUP" "$JS8D_DATA_DIR"
    chmod 755 "$JS8D_DATA_DIR"

    chown "$JS8D_USER:$JS8D_GROUP" "$JS8D_LOG_DIR"
    chmod 755 "$JS8D_LOG_DIR"

    chown "$JS8D_USER:$JS8D_GROUP" "$JS8D_RUN_DIR"
    chmod 755 "$JS8D_RUN_DIR"

    log_info "Created directories with proper permissions"
}

install_binary() {
    log_info "Installing js8d binary..."

    # Check if binary exists in current directory
    if [[ ! -f "./js8d" ]]; then
        log_error "js8d binary not found in current directory"
        log_error "Please run 'make build' first or download a release binary"
        exit 1
    fi

    # Install binary
    cp "./js8d" "/usr/local/bin/js8d"
    chown root:root "/usr/local/bin/js8d"
    chmod 755 "/usr/local/bin/js8d"

    log_info "Installed js8d binary to /usr/local/bin/js8d"
}

install_config() {
    log_info "Installing configuration..."

    # Check if production config exists
    if [[ -f "./configs/config.production.yaml" ]]; then
        # Install config if it doesn't exist
        if [[ ! -f "$JS8D_CONFIG_DIR/config.yaml" ]]; then
            cp "./configs/config.production.yaml" "$JS8D_CONFIG_DIR/config.yaml"
            chown root:root "$JS8D_CONFIG_DIR/config.yaml"
            chmod 644 "$JS8D_CONFIG_DIR/config.yaml"
            log_info "Installed default configuration to $JS8D_CONFIG_DIR/config.yaml"
            log_warn "Please edit $JS8D_CONFIG_DIR/config.yaml with your callsign and radio settings"
        else
            log_info "Configuration file already exists at $JS8D_CONFIG_DIR/config.yaml"
        fi
    else
        log_warn "Production config template not found - you'll need to create $JS8D_CONFIG_DIR/config.yaml manually"
    fi
}

install_systemd_service() {
    log_info "Installing systemd service..."

    # Check if service file exists
    if [[ ! -f "./configs/js8d.service" ]]; then
        log_error "Systemd service file not found at ./configs/js8d.service"
        exit 1
    fi

    # Install service file
    cp "./configs/js8d.service" "$SYSTEMD_DIR/js8d.service"
    chown root:root "$SYSTEMD_DIR/js8d.service"
    chmod 644 "$SYSTEMD_DIR/js8d.service"

    # Reload systemd
    systemctl daemon-reload

    log_info "Installed systemd service"
}

setup_logging() {
    log_info "Setting up log rotation..."

    # Create logrotate configuration
    cat > /etc/logrotate.d/js8d << 'EOF'
/var/log/js8d/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 644 js8d js8d
    postrotate
        /bin/systemctl reload js8d.service > /dev/null 2>&1 || true
    endscript
}
EOF

    log_info "Configured log rotation"
}

show_post_install() {
    log_info "Installation completed successfully!"
    echo
    echo "Next steps:"
    echo "1. Edit the configuration file:"
    echo "   sudo nano $JS8D_CONFIG_DIR/config.yaml"
    echo
    echo "2. Update your callsign and radio settings in the config file"
    echo
    echo "3. Enable and start the service:"
    echo "   sudo systemctl enable js8d.service"
    echo "   sudo systemctl start js8d.service"
    echo
    echo "4. Check service status:"
    echo "   sudo systemctl status js8d.service"
    echo
    echo "5. View logs:"
    echo "   sudo journalctl -u js8d.service -f"
    echo
    echo "6. Access web interface:"
    echo "   http://localhost:8080"
    echo
    log_warn "Remember to configure your callsign and radio settings before starting the service!"
}

# Main installation process
main() {
    log_info "Starting js8d installation..."

    check_root
    check_dependencies
    create_user
    create_directories
    install_binary
    install_config
    install_systemd_service
    setup_logging
    show_post_install
}

# Run main function
main "$@"