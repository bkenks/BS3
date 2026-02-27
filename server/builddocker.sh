#!/bin/sh
# Build the bs3-server Docker image for multiple platforms.
# Can be run from any directory — always uses the repo root as the build context.
#
# Usage:
#   ./builddocker.sh              # multi-platform build (linux/amd64 + linux/arm64), loads to local daemon
#   ./builddocker.sh --push       # multi-platform build and push to registry
#   PLATFORMS=linux/arm64 ./builddocker.sh  # single platform only
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
TAG="${TAG:-bs3-server}"

# Ensure a buildx builder with multi-platform support exists.
if ! docker buildx inspect bs3-builder > /dev/null 2>&1; then
    docker buildx create --name bs3-builder --use
else
    docker buildx use bs3-builder
fi

if [ "$1" = "--push" ]; then
    docker buildx build \
        --platform "$PLATFORMS" \
        -f "$SCRIPT_DIR/Dockerfile" \
        -t "$TAG" \
        --push \
        "$REPO_ROOT"
else
    # --load only works for a single platform; for multi-platform use --push or export manually.
    if [ "$PLATFORMS" = "linux/amd64,linux/arm64" ]; then
        echo "Note: multi-platform builds cannot be loaded into the local daemon."
        echo "Use --push to push to a registry, or set PLATFORMS=linux/arm64 to build a single platform."
        docker buildx build \
            --platform "$PLATFORMS" \
            -f "$SCRIPT_DIR/Dockerfile" \
            -t "$TAG" \
            "$REPO_ROOT"
    else
        docker buildx build \
            --platform "$PLATFORMS" \
            -f "$SCRIPT_DIR/Dockerfile" \
            -t "$TAG" \
            --load \
            "$REPO_ROOT"
    fi
fi
