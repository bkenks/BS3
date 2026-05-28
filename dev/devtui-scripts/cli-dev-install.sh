#!/usr/bin/env bash
set -euo pipefail
cd cli-tool
mkdir -p .testing
GOWORK=off go build -o .testing/bs3 .
echo "Built binary at cli-tool/.testing/bs3"
mkdir -p "$HOME/.local/bin"
cp .testing/bs3 "$HOME/.local/bin/bs3"
chmod +x "$HOME/.local/bin/bs3"
echo "Copied to $HOME/.local/bin/bs3"
