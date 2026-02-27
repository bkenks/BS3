#!/usr/bin/env bash
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
TEST_DIR="$SCRIPT_DIR/.testing"
BINARY_NAME="bs3"

mkdir -p "$TEST_DIR"
go build -o "$TEST_DIR/$BINARY_NAME"
cp -f "$TEST_DIR/bs3" "$HOME/.local/bin"
