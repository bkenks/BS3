#!/usr/bin/env sh
# Install the BS3 CLI from a released version.
#
#   curl -fsSL https://raw.githubusercontent.com/bkenks/BS3/main/cli-tool/scripts/install.sh | sh
#
# By default this installs the `cli/stable` channel. To pin a specific
# released version, set BS3_CLI_VERSION (a bare semver, e.g. 0.4.0):
#
#   BS3_CLI_VERSION=0.4.0 curl -fsSL .../install.sh | sh
#
# The installer prefers a prebuilt binary from the GitHub release matching the
# host OS/arch. If no matching asset exists (older release, unsupported
# platform, or network/API trouble) it falls back to cloning the tag and
# building from source — which requires Go and git.
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

warn() {
    printf "\033[33m%s\033[0m\n" "$1" >&2
}

# fetch <url> <outfile> — download to a file with curl or wget.
fetch() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1" -o "$2"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        return 127
    fi
}

# fetch_stdout <url> — download to stdout with curl or wget.
fetch_stdout() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$1"
    else
        return 127
    fi
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

# ── Detect host platform (Go-style GOOS/GOARCH) ──────────────
# Left empty for anything we don't publish a prebuilt binary for, which routes
# the install through the build-from-source fallback.
case "$(uname -s)" in
    Linux)  PLATFORM_OS="linux" ;;
    Darwin) PLATFORM_OS="darwin" ;;
    *)      PLATFORM_OS="" ;;
esac
case "$(uname -m)" in
    x86_64 | amd64)   PLATFORM_ARCH="amd64" ;;
    arm64 | aarch64)  PLATFORM_ARCH="arm64" ;;
    *)                PLATFORM_ARCH="" ;;
esac

# ── Resolve the version to install ───────────────────────────
# A pinned BS3_CLI_VERSION maps straight to cli/vX.Y.Z. For the stable channel
# we resolve the newest non-prerelease cli/v* release via the GitHub API; the
# download URL needs a concrete version (the cli/stable tag carries no assets).
if [ -n "$BS3_CLI_VERSION" ]; then
    VERSION="$BS3_CLI_VERSION"
    TAG="cli/v$BS3_CLI_VERSION"
    info "Installing BS3 CLI version $BS3_CLI_VERSION ($TAG)..."
else
    TAG="cli/stable"
    info "Installing BS3 CLI from the stable channel..."
    # GitHub lists releases newest-first; tag_name precedes prerelease in each
    # object, so remember the last cli/v* tag seen and emit it at the matching
    # "prerelease": false line.
    VERSION="$(fetch_stdout "https://api.github.com/repos/${REPO}/releases?per_page=100" 2>/dev/null | awk '
        /"tag_name":/ {
            if (match($0, /cli\/v[0-9]+\.[0-9]+\.[0-9]+/)) {
                tag = substr($0, RSTART + 5, RLENGTH - 5)
            } else {
                tag = ""
            }
        }
        /"prerelease":/ && tag != "" {
            if ($0 ~ /false/) { print tag; exit }
            tag = ""
        }
    ' || true)"
fi

# ── Try a prebuilt binary first ──────────────────────────────
install_binary() {
    src="$1"
    mkdir -p "$USER_INSTALL_DIR"
    cp "$src" "$USER_INSTALL_DIR/$BINARY_NAME"
    chmod +x "$USER_INSTALL_DIR/$BINARY_NAME"
    info "Installed $BINARY_NAME to $USER_INSTALL_DIR/$BINARY_NAME"
}

PREBUILT_INSTALLED=false
if [ -n "$VERSION" ] && [ -n "$PLATFORM_OS" ] && [ -n "$PLATFORM_ARCH" ]; then
    ASSET="bs3_v${VERSION}_${PLATFORM_OS}_${PLATFORM_ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/cli/v${VERSION}/${ASSET}"
    info "Downloading prebuilt binary ($PLATFORM_OS/$PLATFORM_ARCH) for v$VERSION..."
    if fetch "$URL" "$TMP_DIR/$ASSET" 2>/dev/null; then
        if tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR" "$BINARY_NAME" 2>/dev/null; then
            install_binary "$TMP_DIR/$BINARY_NAME"
            PREBUILT_INSTALLED=true
        else
            warn "Downloaded archive could not be extracted; falling back to source build."
        fi
    else
        warn "No prebuilt binary for v$VERSION ($PLATFORM_OS/$PLATFORM_ARCH); falling back to source build."
    fi
fi

# ── Fallback: clone the tag and build from source ────────────
if [ "$PREBUILT_INSTALLED" = false ]; then
    command -v go >/dev/null 2>&1 || die "Go is required to build from source but is not installed. Install it from https://go.dev/dl/"
    command -v git >/dev/null 2>&1 || die "git is required to build from source but is not installed."

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

    info "Building BS3 CLI from source..."
    # Build cli-tool in isolation (GOWORK=off) so it resolves dependencies from
    # cli-tool's own go.mod/go.sum, independent of the repo-root go.work workspace.
    cd "$TMP_DIR/src/cli-tool"
    CGO_ENABLED=0 GOWORK=off go build -o "$TMP_DIR/$BINARY_NAME" . 2>&1 || die "Build failed."
    install_binary "$TMP_DIR/$BINARY_NAME"
fi

case ":$PATH:" in
    *":$USER_INSTALL_DIR:"*) ;;
    *)
        printf "\n\033[33mAdd this to your shell config (~/.bashrc, ~/.zshrc, etc.):\033[0m\n"
        printf '  export PATH="%s:$PATH"\n\n' "$USER_INSTALL_DIR"
        ;;
esac
