#!/usr/bin/env bash
set -euo pipefail

# Name of your Go binary
BINARY_NAME="bs3"

# List of OS/ARCH combinations
PLATFORMS=(
  "linux amd64"
  "linux arm64"
)

# Build output folder
OUTPUT_DIR=".builds"
mkdir -p "$OUTPUT_DIR"

for PLATFORM in "${PLATFORMS[@]}"; do
  OS=$(echo $PLATFORM | awk '{print $1}')
  ARCH=$(echo $PLATFORM | awk '{print $2}')

  # Set binary name for Windows
  BIN_NAME="$BINARY_NAME"
  if [[ "$OS" == "windows" ]]; then
    BIN_NAME="${BINARY_NAME}.exe"
  fi

  ZIP_NAME="${OUTPUT_DIR}/${OS}_${ARCH}.zip"

  echo "Building $OS/$ARCH -> $ZIP_NAME"

  # Build binary
  GOOS=$OS GOARCH=$ARCH go build -o "$BIN_NAME"

  # Zip the binary
  zip -j "$ZIP_NAME" "$BIN_NAME"

  # Remove the standalone binary
  rm "$BIN_NAME"
done

echo "All builds completed and zipped!"
