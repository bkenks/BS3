#!/usr/bin/env sh
set -e

REPO="bkenks/bs3-cli"
BINARY_NAME="bs3-cli"
VERSION=$(curl -fsSL https://api.github.com/repos/bkenks/bs3-cli/releases/latest |
    grep tag_name |
    cut -d '"' -f4)

die() {
    printf "\033[31m%s\033[0m\n" "$1" >&2
    exit 1
}

detect_platform() {
    OS="$(uname | tr '[:upper:]' '[:lower:]')"

    case "$(uname -m)" in
    x86_64 | amd64) ARCH="amd64" ;;
    arm64 | aarch64) ARCH="arm64" ;;
    *) die "Unsupported architecture: $(uname -m)" ;;
    esac
}

detect_fetch() {
    if command -v curl >/dev/null 2>&1; then
        FETCH="curl -fL"
    elif command -v wget >/dev/null 2>&1; then
        FETCH="wget -qO-"
    else
        die "curl or wget is required"
    fi
}

install_binary() {
    DEST="$HOME/.local/bin"
    mkdir -p "$DEST"

    FILE="${OS}_${ARCH}.zip"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE}"

    TMP_DIR="$(mktemp -d)"
    ZIP_PATH="$TMP_DIR/$FILE"

    printf "Downloading %s...\n" "$URL"

    if command -v curl >/dev/null 2>&1; then
        curl -fL "$URL" -o "$ZIP_PATH"
    else
        wget "$URL" -O "$ZIP_PATH"
    fi

    printf "Extracting...\n"
    unzip -q "$ZIP_PATH" -d "$TMP_DIR"

    # assumes zip contains the binary directly
    if [ ! -f "$TMP_DIR/$BINARY_NAME" ]; then
        die "Binary not found in archive."
    fi

    mv "$TMP_DIR/$BINARY_NAME" "$DEST/$BINARY_NAME"
    chmod +x "$DEST/$BINARY_NAME"

    rm -rf "$TMP_DIR"

    printf "\n✅ Installed to %s/%s\n" "$DEST" "$BINARY_NAME"

    if ! echo "$PATH" | grep -q "$HOME/.local/bin"; then
        printf "\n⚠️  Add this to your shell config if command not found:\n"
        printf "export PATH="$HOME/.local/bin:$PATH"\n\n"
    fi
}

main() {
    detect_platform
    detect_fetch
    install_binary
}

main "$@"
