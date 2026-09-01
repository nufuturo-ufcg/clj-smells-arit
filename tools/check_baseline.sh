#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <baseline-normal.json> <current-normal.json> <baseline-cross.json> <current-cross.json>" >&2
  exit 2
fi

reportdiff_cache="${GOCACHE:-/tmp/arit-reportdiff-cache}"

echo "normal"
GOCACHE="$reportdiff_cache" go run ./tools/reportdiff "$1" "$2"

echo "cross-namespace"
GOCACHE="$reportdiff_cache" go run ./tools/reportdiff "$3" "$4"
