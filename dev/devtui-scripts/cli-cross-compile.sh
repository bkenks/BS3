#!/usr/bin/env bash
set -euo pipefail
cd cli-tool
mkdir -p .builds
echo "Created cli-tool/.builds"
for target in "linux amd64" "linux arm64"; do
  read -r GOOS GOARCH <<<"$target"
  GOWORK=off GOOS="$GOOS" GOARCH="$GOARCH" go build -o bs3 .
  zip -j ".builds/${GOOS}_${GOARCH}.zip" bs3 >/dev/null
  rm -f bs3
  echo "Built and zipped ${GOOS}/${GOARCH} → ${GOOS}_${GOARCH}.zip"
done
