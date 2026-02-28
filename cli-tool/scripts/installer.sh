#!/usr/bin/env sh
set -e

REPO="bkenks/BS3"
BINARY_NAME="bs3"
INSTALL_DIR="$HOME/.local/bin"

die() {
    printf "\033[31mError: %s\033[0m\n" "$1" >&2
    exit 1
}

info() {
    printf "\033[32m%s\033[0m\n" "$1"
}

command -v go >/dev/null 2>&1 || die "Go is required but not installed. Install it from https://go.dev/dl/"
command -v git >/dev/null 2>&1 || die "git is required but not installed."

info "Cloning BS3 repository..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

git clone --depth=1 "https://github.com/${REPO}.git" "$TMP_DIR/BS3" >/dev/null 2>&1

info "Building bs3 CLI..."
rm -f "$TMP_DIR/BS3/go.work" "$TMP_DIR/BS3/go.work.sum"
cd "$TMP_DIR/BS3/cli-tool"
go build -o "$TMP_DIR/$BINARY_NAME" . 2>&1 || die "Build failed."

mkdir -p "$INSTALL_DIR"
mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

info "Installed to $INSTALL_DIR/$BINARY_NAME"

case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        printf "\n\033[33mAdd this to your shell config (~/.bashrc, ~/.zshrc, etc.):\033[0m\n"
        printf '  export PATH="$HOME/.local/bin:$PATH"\n\n'
        ;;
esac
