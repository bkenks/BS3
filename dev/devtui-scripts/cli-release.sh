#!/usr/bin/env bash
# Wrap dev/scripts/release-cli.sh — translate devtui env vars to positional args.
#   $VERSION  bare semver X.Y.Z
#   $CHANNEL  stable | prerelease
set -euo pipefail
ARGS=("$VERSION")
if [ "${CHANNEL:-stable}" = "prerelease" ]; then
  ARGS+=("--prerelease")
fi
exec bash dev/scripts/release-cli.sh "${ARGS[@]}"
