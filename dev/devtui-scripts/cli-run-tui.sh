#!/usr/bin/env bash
set -euo pipefail
cd cli-tool
exec env GOWORK=off go run . tui
