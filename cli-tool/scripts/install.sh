#!/usr/bin/env sh
# Install the BS3 CLI from a released version.
#
#   curl -fsSL https://raw.githubusercontent.com/bkenks/BS3/main/cli-tool/scripts/install.sh | sh
#
# By default this installs the `cli/stable` channel. To pin a specific
# released version, set BS3_CLI_VERSION (a bare semver, e.g. 0.4.0):
#
#   BS3_CLI_VERSION=0.4.0 curl -fsSL .../install.sh | sh
set -e

REPO="bkenks/BS3"
BINARY_NAME="bs3"
USER_INSTALL_DIR="$HOME/.local/bin"

die() {
    printf "\033[31mError: %s\033[0m\n" "$1" >&2
    exit 1
}

info() {
    printf "\033[32m%s\033[0m\n" "$1"
}

command -v go >/dev/null 2>&1 || die "Go is required but not installed. Install it from https://go.dev/dl/"
command -v git >/dev/null 2>&1 || die "git is required but not installed."

# Pick the git tag to install. Default to the moving cli/stable channel;
# BS3_CLI_VERSION pins an exact released version (cli/vX.Y.Z).
if [ -n "$BS3_CLI_VERSION" ]; then
    TAG="cli/v$BS3_CLI_VERSION"
    info "Installing BS3 CLI version $BS3_CLI_VERSION ($TAG)..."
else
    TAG="cli/stable"
    info "Installing BS3 CLI from the stable channel ($TAG)..."
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

# Shallow-clone the repo at the requested tag. Clone into "src" rather than
# "BS3" — on case-insensitive filesystems (macOS default) a "BS3" directory
# and the "bs3" build output would collide.
if ! git clone --depth 1 --branch "$TAG" "https://github.com/${REPO}.git" "$TMP_DIR/src" >/dev/null 2>&1; then
    if [ -n "$BS3_CLI_VERSION" ]; then
        die "No released CLI version '$BS3_CLI_VERSION' found (tag $TAG does not exist)."
    else
        die "No stable CLI release found yet (tag $TAG does not exist). Set BS3_CLI_VERSION to install a specific version."
    fi
fi

info "Building BS3 CLI..."
# The repo's root go.work only includes ./dev, so building cli-tool must run
# with GOWORK=off — otherwise the build fails resolving the workspace.
cd "$TMP_DIR/src/cli-tool"
CGO_ENABLED=0 GOWORK=off go build -o "$TMP_DIR/$BINARY_NAME" . 2>&1 || die "Build failed."

# Install to ~/.local/bin.
mkdir -p "$USER_INSTALL_DIR"
cp "$TMP_DIR/$BINARY_NAME" "$USER_INSTALL_DIR/$BINARY_NAME"
chmod +x "$USER_INSTALL_DIR/$BINARY_NAME"
info "Installed $BINARY_NAME to $USER_INSTALL_DIR/$BINARY_NAME"

case ":$PATH:" in
    *":$USER_INSTALL_DIR:"*) ;;
    *)
        printf "\n\033[33mAdd this to your shell config (~/.bashrc, ~/.zshrc, etc.):\033[0m\n"
        printf '  export PATH="%s:$PATH"\n\n' "$USER_INSTALL_DIR"
        ;;
esac
