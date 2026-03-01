#!/usr/bin/env sh
set -e

BINARY_NAME="bs3"
USER_INSTALL_DIR="$HOME/.local/bin"
SYSTEM_INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/bs3"

if [ -w "$SYSTEM_INSTALL_DIR" ]; then
    SUDO=""
elif command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
else
    SUDO=""
fi

info() {
    printf "\033[32m%s\033[0m\n" "$1"
}

warn() {
    printf "\033[33m%s\033[0m\n" "$1"
}

removed=0

# Remove from ~/.local/bin
if [ -f "$USER_INSTALL_DIR/$BINARY_NAME" ]; then
    rm -f "$USER_INSTALL_DIR/$BINARY_NAME"
    info "Removed $USER_INSTALL_DIR/$BINARY_NAME"
    removed=1
fi

# Remove from /usr/local/bin
if [ -f "$SYSTEM_INSTALL_DIR/$BINARY_NAME" ]; then
    $SUDO rm -f "$SYSTEM_INSTALL_DIR/$BINARY_NAME"
    info "Removed $SYSTEM_INSTALL_DIR/$BINARY_NAME"
    removed=1
fi

if [ "$removed" -eq 0 ]; then
    warn "No bs3 binary found in $USER_INSTALL_DIR or $SYSTEM_INSTALL_DIR"
fi

# Optionally remove config
if [ -d "$CONFIG_DIR" ]; then
    printf "\033[33mRemove config directory %s? [y/N] \033[0m" "$CONFIG_DIR"
    read -r answer </dev/tty
    case "$answer" in
        [Yy]*)
            rm -rf "$CONFIG_DIR"
            info "Removed $CONFIG_DIR"
            ;;
        *)
            info "Config kept at $CONFIG_DIR"
            ;;
    esac
fi
