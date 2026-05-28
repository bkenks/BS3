#!/usr/bin/env bash
set -euo pipefail
cd server
exec env GOWORK=off go run ./cmd/
