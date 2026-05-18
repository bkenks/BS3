#!/usr/bin/env bash
# This file opens the Development Tools menu.
# Runs from the repo root (one level up from this script) regardless of where
# it is invoked — the dev hub must be started from the repository root.

cd "$(dirname "$0")/.." && go run ./dev/
