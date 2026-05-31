#!/usr/bin/env bash
# Release the BS3 CLI: create a component-prefixed GitHub release
# (cli/vX.Y.Z) and, for stable releases, move the cli/stable pointer tag.
#
# Run `gh auth login` once first, then:
#   ./dev/scripts/release-cli.sh <version> [--prerelease]
#
# <version>      bare semver X.Y.Z (no leading v, no prefix)
# --prerelease   optional; marks the GitHub release as a prerelease and
#                does NOT move the cli/stable tag
set -e

usage() {
  echo "Usage: $0 <version> [--prerelease]" >&2
  echo "  <version>  bare semver X.Y.Z (e.g. 0.4.0)" >&2
  exit 1
}

# devtui passes inputs as env vars (VERSION, CHANNEL); the bs3dev Go hub and
# manual runs pass positional args. When invoked with no positional args, map
# the env vars onto the positional interface below so devtui.toml can point
# straight at this script (no wrapper needed).
if [ "$#" -eq 0 ] && [ -n "${VERSION:-}" ]; then
  set -- "$VERSION"
  if [ "${CHANNEL:-}" = "prerelease" ]; then
    set -- "$@" "--prerelease"
  fi
fi

VERSION="$1"
PRERELEASE_ARG="$2"

# Validate the version argument.
if [ -z "$VERSION" ]; then
  echo "Error: <version> is required." >&2
  usage
fi
if ! echo "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: version '$VERSION' is not a bare semver X.Y.Z." >&2
  usage
fi

PRERELEASE=false
if [ -n "$PRERELEASE_ARG" ]; then
  if [ "$PRERELEASE_ARG" = "--prerelease" ]; then
    PRERELEASE=true
  else
    echo "Error: unexpected argument '$PRERELEASE_ARG'." >&2
    usage
  fi
fi

# This script lives in dev/scripts/, so the repo root is two levels up.
cd "$(dirname "$0")/../.."

# Warn (but continue) if the working tree is dirty — the release will be cut
# from the current HEAD commit, not from the uncommitted changes.
if [ -n "$(git status --porcelain)" ]; then
  echo "Warning: working tree has uncommitted changes; releasing from HEAD anyway." >&2
fi

# ── Cross-compile release binaries ───────────────────────────
# Build a static `bs3` binary for each target platform and archive it. The
# archives are uploaded as GitHub release assets so users can grab a prebuilt
# binary instead of building from source via install.sh.
#
# Build conventions match the bs3dev hub's "CLI › Cross-Compile Build":
# CGO_ENABLED=0 + GOWORK=off so cli-tool resolves against its own
# go.mod/go.sum, independent of the repo-root go.work workspace.
PLATFORMS="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64"

BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT

ASSETS=""
echo "Building release binaries for: $PLATFORMS"
for platform in $PLATFORMS; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"

  bin="bs3"
  [ "$GOOS" = "windows" ] && bin="bs3.exe"

  stage="$BUILD_DIR/bs3_v${VERSION}_${GOOS}_${GOARCH}"
  mkdir -p "$stage"

  echo "  → $GOOS/$GOARCH"
  ( cd cli-tool && CGO_ENABLED=0 GOWORK=off GOOS="$GOOS" GOARCH="$GOARCH" \
      go build -o "$stage/$bin" . ) || {
    echo "Error: build failed for $GOOS/$GOARCH." >&2
    exit 1
  }

  # Archive: .zip for Windows, .tar.gz elsewhere (idiomatic per platform).
  if [ "$GOOS" = "windows" ]; then
    archive="$BUILD_DIR/bs3_v${VERSION}_${GOOS}_${GOARCH}.zip"
    ( cd "$stage" && zip -qj "$archive" "$bin" )
  else
    archive="$BUILD_DIR/bs3_v${VERSION}_${GOOS}_${GOARCH}.tar.gz"
    tar -czf "$archive" -C "$stage" "$bin"
  fi
  ASSETS="$ASSETS $archive"
done

# Checksums for all archives, so users can verify downloads.
CHECKSUMS="$BUILD_DIR/bs3_v${VERSION}_checksums.txt"
( cd "$BUILD_DIR" && shasum -a 256 bs3_v${VERSION}_*.tar.gz bs3_v${VERSION}_*.zip > "$CHECKSUMS" )
ASSETS="$ASSETS $CHECKSUMS"

# ── GitHub release ───────────────────────────────────────────
TAG="cli/v$VERSION"
TARGET="$(git rev-parse HEAD)"
INSTALL_ONELINER="curl -fsSL https://raw.githubusercontent.com/bkenks/BS3/main/cli-tool/scripts/install.sh | sh"

if [ "$PRERELEASE" = false ]; then
  NOTES="BS3 CLI v$VERSION

Install (stable channel):
\`\`\`
$INSTALL_ONELINER
\`\`\`

This release is now the \`cli/stable\` channel. To pin this exact version:
\`\`\`
BS3_CLI_VERSION=$VERSION $INSTALL_ONELINER
\`\`\`"
else
  NOTES="BS3 CLI v$VERSION (prerelease)

This is a prerelease; the \`cli/stable\` channel was not updated.

To install this exact version:
\`\`\`
BS3_CLI_VERSION=$VERSION $INSTALL_ONELINER
\`\`\`"
fi

# `gh release create` creates the cli/v$VERSION tag itself, and uploads the
# cross-compiled archives + checksums as release assets ($ASSETS, unquoted so
# each path is a separate positional arg).
if [ "$PRERELEASE" = true ]; then
  gh release create "$TAG" \
    --title "CLI v$VERSION" \
    --target "$TARGET" \
    --notes "$NOTES" \
    --prerelease \
    $ASSETS
else
  gh release create "$TAG" \
    --title "CLI v$VERSION" \
    --target "$TARGET" \
    --notes "$NOTES" \
    $ASSETS
fi

RELEASE_URL="$(gh release view "$TAG" --json url --jq .url 2>/dev/null || echo "$TAG")"

# ── Move the stable pointer (stable releases only) ───────────
if [ "$PRERELEASE" = false ]; then
  git tag -f cli/stable "$TARGET"
  git push -f origin cli/stable
  echo "──────────────────────────────────────────────────────────"
  echo "CLI release done: GitHub release $RELEASE_URL; cli/stable now points at $TARGET"
else
  echo "──────────────────────────────────────────────────────────"
  echo "CLI prerelease done: GitHub release $RELEASE_URL; cli/stable unchanged"
fi
