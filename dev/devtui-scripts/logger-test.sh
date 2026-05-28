#!/usr/bin/env bash
set -euo pipefail
cd logger
CLICOLOR_FORCE=1 GOWORK=off go run ./cmd/
