#!/usr/bin/env bash
# Release the BS3 CLI: create a component-prefixed GitHub release
# (cli/vX.Y.Z) and, for stable releases, move the cli/stable pointer tag.
#
# Run `gh auth login` once first, then:
#   ./dev/scripts/release-cli.sh <version> [--prerelease]
#
# <version>      bare semver X.Y.Z (no leading v, no prefix)
# --prerelease   optional; marks the GitHub release as a prerelease and
#                does NOT move the cli/stable tag
set -e

usage() {
  echo "Usage: $0 <version> [--prerelease]" >&2
  echo "  <version>  bare semver X.Y.Z (e.g. 0.4.0)" >&2
  exit 1
}

VERSION="$1"
PRERELEASE_ARG="$2"

# Validate the version argument.
if [ -z "$VERSION" ]; then
  echo "Error: <version> is required." >&2
  usage
fi
if ! echo "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: version '$VERSION' is not a bare semver X.Y.Z." >&2
  usage
fi

PRERELEASE=false
if [ -n "$PRERELEASE_ARG" ]; then
  if [ "$PRERELEASE_ARG" = "--prerelease" ]; then
    PRERELEASE=true
  else
    echo "Error: unexpected argument '$PRERELEASE_ARG'." >&2
    usage
  fi
fi

# This script lives in dev/scripts/, so the repo root is two levels up.
cd "$(dirname "$0")/../.."

# Warn (but continue) if the working tree is dirty — the release will be cut
# from the current HEAD commit, not from the uncommitted changes.
if [ -n "$(git status --porcelain)" ]; then
  echo "Warning: working tree has uncommitted changes; releasing from HEAD anyway." >&2
fi

# ── GitHub release ───────────────────────────────────────────
TAG="cli/v$VERSION"
TARGET="$(git rev-parse HEAD)"
INSTALL_ONELINER="curl -fsSL https://raw.githubusercontent.com/bkenks/BS3/main/cli-tool/scripts/install.sh | sh"

if [ "$PRERELEASE" = false ]; then
  NOTES="BS3 CLI v$VERSION

Install (stable channel):
\`\`\`
$INSTALL_ONELINER
\`\`\`

This release is now the \`cli/stable\` channel. To pin this exact version:
\`\`\`
BS3_CLI_VERSION=$VERSION $INSTALL_ONELINER
\`\`\`"
else
  NOTES="BS3 CLI v$VERSION (prerelease)

This is a prerelease; the \`cli/stable\` channel was not updated.

To install this exact version:
\`\`\`
BS3_CLI_VERSION=$VERSION $INSTALL_ONELINER
\`\`\`"
fi

# `gh release create` creates the cli/v$VERSION tag itself.
if [ "$PRERELEASE" = true ]; then
  gh release create "$TAG" \
    --title "CLI v$VERSION" \
    --target "$TARGET" \
    --notes "$NOTES" \
    --prerelease
else
  gh release create "$TAG" \
    --title "CLI v$VERSION" \
    --target "$TARGET" \
    --notes "$NOTES"
fi

RELEASE_URL="$(gh release view "$TAG" --json url --jq .url 2>/dev/null || echo "$TAG")"

# ── Move the stable pointer (stable releases only) ───────────
if [ "$PRERELEASE" = false ]; then
  git tag -f cli/stable "$TARGET"
  git push -f origin cli/stable
  echo "──────────────────────────────────────────────────────────"
  echo "CLI release done: GitHub release $RELEASE_URL; cli/stable now points at $TARGET"
else
  echo "──────────────────────────────────────────────────────────"
  echo "CLI prerelease done: GitHub release $RELEASE_URL; cli/stable unchanged"
fi
