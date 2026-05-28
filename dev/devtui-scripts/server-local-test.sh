#!/usr/bin/env bash
exec docker compose -f server/compose/compose.dev.yml up --build
