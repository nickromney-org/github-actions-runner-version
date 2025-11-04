#!/usr/bin/env bash
#
# Update the Daily Version Checks section in README.md
#
# Usage: update-readme-status.sh <runner_version> <runner_status> <runner_badge> \
#                                 <terraform_version> <terraform_status> <terraform_badge> \
#                                 <nodejs_version> <nodejs_status> <nodejs_badge>

set -euo pipefail

# Parse arguments
RUNNER_VERSION="${1:-unknown}"
RUNNER_STATUS="${2:-unknown}"
RUNNER_BADGE="${3:-grey}"
TERRAFORM_VERSION="${4:-unknown}"
TERRAFORM_STATUS="${5:-unknown}"
TERRAFORM_BADGE="${6:-grey}"
NODEJS_VERSION="${7:-unknown}"
NODEJS_STATUS="${8:-unknown}"
NODEJS_BADGE="${9:-grey}"

# Get current timestamp
TIMESTAMP=$(date -u +"%d %b %Y %H:%M UTC")

# Create a temporary file for the new status section
STATUS_FILE=$(mktemp)
cat > "$STATUS_FILE" <<EOF
## Daily Version Checks

**Last updated:** ${TIMESTAMP}

### GitHub Actions Runner

![Status](https://img.shields.io/badge/status-${RUNNER_STATUS}-${RUNNER_BADGE})

**Latest version:** \`v${RUNNER_VERSION}\`

### Terraform

![Status](https://img.shields.io/badge/status-${TERRAFORM_STATUS}-${TERRAFORM_BADGE})

**Latest version:** \`v${TERRAFORM_VERSION}\`

### Node.js

![Status](https://img.shields.io/badge/status-${NODEJS_STATUS}-${NODEJS_BADGE})

**Latest version:** \`v${NODEJS_VERSION}\`
EOF

# Create a temporary file for the new README
TMP_README=$(mktemp)

# Process the README: replace the Daily Version Checks section
if grep -q "## Daily Version Checks" README.md; then
  # Section exists, replace it
  awk '
    /^## Daily Version Checks/ {
      # Insert new status file
      while ((getline line < "'"$STATUS_FILE"'") > 0) {
        print line
      }
      close("'"$STATUS_FILE"'")

      # Skip old content until next ## heading
      while (getline > 0) {
        if (/^## /) {
          print
          break
        }
      }
      next
    }
    {print}
  ' README.md > "$TMP_README"
else
  # Section doesn't exist, insert after badges (before first ## heading after title)
  awk '
    /^## / && !inserted {
      print ""
      # Insert new status file
      while ((getline line < "'"$STATUS_FILE"'") > 0) {
        print line
      }
      close("'"$STATUS_FILE"'")
      print ""
      inserted=1
    }
    {print}
  ' README.md > "$TMP_README"
fi

# Replace the original file
mv "$TMP_README" README.md

# Clean up
rm -f "$STATUS_FILE"

echo "✅ Updated README.md with version status"
