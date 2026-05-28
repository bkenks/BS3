#!/usr/bin/env bash
set -euo pipefail
docker compose -f server/compose/compose.dev.yml down --volumes --rmi local --remove-orphans
