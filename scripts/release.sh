#!/usr/bin/env bash
# Build the BS3 server image for multiple platforms and push it to Docker Hub.
# Run `docker login` once first. Edit the variables below, then run:
#   ./scripts/release.sh
set -e

# ── Edit these ───────────────────────────────────────────────
DOCKER_USER="bkenks"
IMAGE_NAME="bs3-server"
VERSION="0.1.0"
PLATFORMS="linux/amd64,linux/arm64"
# ─────────────────────────────────────────────────────────────

IMAGE="$DOCKER_USER/$IMAGE_NAME"
BUILDER="bs3-builder"

# Build context must be the repo root (Dockerfile pulls in ../logger).
# This script lives in scripts/, so the repo root is one level up.
cd "$(dirname "$0")/.."

# Multi-platform builds need a buildx builder with the docker-container driver;
# the default builder can only build for the host architecture. Created once,
# reused on every later run.
if ! docker buildx inspect "$BUILDER" >/dev/null 2>&1; then
  docker buildx create --name "$BUILDER" >/dev/null
fi
docker buildx use "$BUILDER"

# --push is required: a multi-platform image cannot be loaded into the local
# daemon, only published to a registry.
docker buildx build \
  --platform "$PLATFORMS" \
  -f server/Dockerfile \
  -t "$IMAGE:$VERSION" \
  -t "$IMAGE:latest" \
  --push \
  .

echo "pushed $IMAGE:$VERSION and $IMAGE:latest ($PLATFORMS)"
