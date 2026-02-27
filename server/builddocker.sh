#!/bin/sh
# Build the bs3-server Docker image.
# Can be run from any directory — always uses the repo root as the build context.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

docker build -f "$SCRIPT_DIR/Dockerfile" -t bs3-server "$REPO_ROOT"
