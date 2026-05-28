#!/usr/bin/env bash
set -euo pipefail
cd server
GOWORK=off go build -o bs3-server ./cmd
echo "Built server/bs3-server"
