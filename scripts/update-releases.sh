#!/bin/bash
set -euo pipefail

GITHUB_TOKEN="${GITHUB_TOKEN:-}"
OUTPUT_FILE="data/releases.json"

echo "Fetching all releases from GitHub..."

# Use go run to bootstrap
go run ./cmd/bootstrap-releases \
  ${GITHUB_TOKEN:+--token "$GITHUB_TOKEN"} \
  --output "$OUTPUT_FILE"

RELEASE_COUNT=$(jq '.releases | length' "$OUTPUT_FILE")
echo "✅ Updated $OUTPUT_FILE with $RELEASE_COUNT releases"
