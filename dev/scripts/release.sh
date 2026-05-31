#!/usr/bin/env bash
# Release the BS3 SERVER: build the multi-platform image, push it to the GitHub
# Container Registry (GHCR), and create a component-prefixed GitHub release
# (server/vX.Y.Z).
#
# Run `docker login ghcr.io` (use a PAT with write:packages) and
# `gh auth login` once first, then:
#   ./dev/scripts/release.sh <version> [--prerelease]
#
# <version>      bare semver X.Y.Z (no leading v, no prefix)
# --prerelease   optional; marks the GitHub release as a prerelease and
#                skips tagging the image :latest
set -e

# ── Edit these ───────────────────────────────────────────────
REGISTRY="ghcr.io"
OWNER="bkenks"
IMAGE_NAME="bs3-server"
PLATFORMS="linux/amd64,linux/arm64"
# ─────────────────────────────────────────────────────────────

usage() {
  echo "Usage: $0 <version> [--prerelease]" >&2
  echo "  <version>  bare semver X.Y.Z (e.g. 3.3.0)" >&2
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

IMAGE="$REGISTRY/$OWNER/$IMAGE_NAME"
BUILDER="bs3-builder"

# Build context must be the repo root (Dockerfile pulls in ../logger).
# This script lives in dev/scripts/, so the repo root is two levels up.
cd "$(dirname "$0")/../.."

# Warn (but continue) if the working tree is dirty — the release will be cut
# from the current HEAD commit, not from the uncommitted changes.
if [ -n "$(git status --porcelain)" ]; then
  echo "Warning: working tree has uncommitted changes; releasing from HEAD anyway." >&2
fi

# Multi-platform builds need a buildx builder with the docker-container driver;
# the default builder can only build for the host architecture. Created once,
# reused on every later run.
if ! docker buildx inspect "$BUILDER" >/dev/null 2>&1; then
  docker buildx create --name "$BUILDER" >/dev/null
fi
docker buildx use "$BUILDER"

# Build the image tag list. :latest is only published for stable releases.
BUILD_TAGS="-t $IMAGE:$VERSION"
if [ "$PRERELEASE" = false ]; then
  BUILD_TAGS="$BUILD_TAGS -t $IMAGE:latest"
fi

# --push is required: a multi-platform image cannot be loaded into the local
# daemon, only published to a registry.
docker buildx build \
  --platform "$PLATFORMS" \
  -f server/Dockerfile \
  $BUILD_TAGS \
  --push \
  .

if [ "$PRERELEASE" = false ]; then
  echo "pushed $IMAGE:$VERSION and $IMAGE:latest ($PLATFORMS)"
else
  echo "pushed $IMAGE:$VERSION ($PLATFORMS)"
fi

# ── GitHub release ───────────────────────────────────────────
TAG="server/v$VERSION"
TARGET="$(git rev-parse HEAD)"

if [ "$PRERELEASE" = false ]; then
  NOTES="BS3 Server v$VERSION

Docker image: \`docker pull $IMAGE:$VERSION\`
The \`$IMAGE:latest\` tag also points at this release.

Platforms: $PLATFORMS"
else
  NOTES="BS3 Server v$VERSION (prerelease)

Docker image: \`docker pull $IMAGE:$VERSION\`
This is a prerelease; the \`:latest\` tag was not updated.

Platforms: $PLATFORMS"
fi

# `gh release create` creates the server/v$VERSION tag itself.
if [ "$PRERELEASE" = true ]; then
  gh release create "$TAG" \
    --title "Server v$VERSION" \
    --target "$TARGET" \
    --notes "$NOTES" \
    --prerelease
else
  gh release create "$TAG" \
    --title "Server v$VERSION" \
    --target "$TARGET" \
    --notes "$NOTES"
fi

RELEASE_URL="$(gh release view "$TAG" --json url --jq .url 2>/dev/null || echo "$TAG")"

echo "──────────────────────────────────────────────────────────"
echo "Server release done: image $IMAGE:$VERSION pushed, GitHub release $RELEASE_URL"
