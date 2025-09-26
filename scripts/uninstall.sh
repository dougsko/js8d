#!/bin/bash

# js8d Uninstall Script
# Removes js8d daemon and all associated files

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

confirm_uninstall() {
    echo
    log_warn "This will completely remove js8d and all its data!"
    log_warn "Configuration files and message database will be deleted."
    echo
    read -p "Are you sure you want to continue? (yes/no): " -r
    echo
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "Uninstall cancelled"
        exit 0
    fi
}

stop_service() {
    log_info "Stopping js8d service..."

    if systemctl is-active --quiet js8d.service; then
        systemctl stop js8d.service
        log_info "Stopped js8d service"
    else
        log_info "js8d service is not running"
    fi

    if systemctl is-enabled --quiet js8d.service; then
        systemctl disable js8d.service
        log_info "Disabled js8d service"
    else
        log_info "js8d service is not enabled"
    fi
}

remove_systemd_service() {
    log_info "Removing systemd service..."

    if [[ -f "$SYSTEMD_DIR/js8d.service" ]]; then
        rm -f "$SYSTEMD_DIR/js8d.service"
        systemctl daemon-reload
        log_info "Removed systemd service file"
    else
        log_info "Systemd service file not found"
    fi
}

remove_binary() {
    log_info "Removing js8d binary..."

    if [[ -f "/usr/local/bin/js8d" ]]; then
        rm -f "/usr/local/bin/js8d"
        log_info "Removed js8d binary from /usr/local/bin/js8d"
    else
        log_info "js8d binary not found"
    fi
}

remove_directories() {
    log_info "Removing directories..."

    # Remove data directory (contains database)
    if [[ -d "$JS8D_DATA_DIR" ]]; then
        rm -rf "$JS8D_DATA_DIR"
        log_info "Removed data directory: $JS8D_DATA_DIR"
    fi

    # Remove log directory
    if [[ -d "$JS8D_LOG_DIR" ]]; then
        rm -rf "$JS8D_LOG_DIR"
        log_info "Removed log directory: $JS8D_LOG_DIR"
    fi

    # Remove runtime directory
    if [[ -d "$JS8D_RUN_DIR" ]]; then
        rm -rf "$JS8D_RUN_DIR"
        log_info "Removed runtime directory: $JS8D_RUN_DIR"
    fi

    # Ask about config directory
    if [[ -d "$JS8D_CONFIG_DIR" ]]; then
        echo
        read -p "Remove configuration directory $JS8D_CONFIG_DIR? (yes/no): " -r
        if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            rm -rf "$JS8D_CONFIG_DIR"
            log_info "Removed configuration directory: $JS8D_CONFIG_DIR"
        else
            log_info "Kept configuration directory: $JS8D_CONFIG_DIR"
        fi
    fi
}

remove_user() {
    log_info "Removing js8d user and group..."

    # Remove user
    if getent passwd "$JS8D_USER" &> /dev/null; then
        userdel "$JS8D_USER"
        log_info "Removed user: $JS8D_USER"
    else
        log_info "User $JS8D_USER not found"
    fi

    # Remove group
    if getent group "$JS8D_GROUP" &> /dev/null; then
        groupdel "$JS8D_GROUP"
        log_info "Removed group: $JS8D_GROUP"
    else
        log_info "Group $JS8D_GROUP not found"
    fi
}

remove_logrotate() {
    log_info "Removing log rotation configuration..."

    if [[ -f "/etc/logrotate.d/js8d" ]]; then
        rm -f "/etc/logrotate.d/js8d"
        log_info "Removed logrotate configuration"
    else
        log_info "Logrotate configuration not found"
    fi
}

show_completion() {
    log_info "js8d has been completely uninstalled!"
    echo
    log_info "All files and directories have been removed"

    if [[ -d "$JS8D_CONFIG_DIR" ]]; then
        log_info "Configuration directory preserved at: $JS8D_CONFIG_DIR"
    fi

    echo
    log_info "Thank you for using js8d!"
}

# Main uninstall process
main() {
    log_info "Starting js8d uninstall..."

    check_root
    confirm_uninstall
    stop_service
    remove_systemd_service
    remove_binary
    remove_directories
    remove_user
    remove_logrotate
    show_completion
}

# Run main function
main "$@"