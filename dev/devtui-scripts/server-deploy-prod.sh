#!/usr/bin/env bash
set -euo pipefail
docker compose -f server/compose/compose.prod.yml pull
docker compose -f server/compose/compose.prod.yml up -d
