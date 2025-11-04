#!/usr/bin/env bash
#
# Update the Daily Version Checks section in README.md
#
# Usage: update-readme-status.sh <runner_version> <runner_status> <runner_badge> <runner_output> \
#                                 <terraform_version> <terraform_status> <terraform_badge> \
#                                 <nodejs_version> <nodejs_status> <nodejs_badge>

set -euo pipefail

# Parse arguments
RUNNER_VERSION="${1:-unknown}"
RUNNER_STATUS="${2:-unknown}"
RUNNER_BADGE="${3:-grey}"
RUNNER_OUTPUT="${4:-}"
TERRAFORM_VERSION="${5:-unknown}"
TERRAFORM_STATUS="${6:-unknown}"
TERRAFORM_BADGE="${7:-grey}"
NODEJS_VERSION="${8:-unknown}"
NODEJS_STATUS="${9:-unknown}"
NODEJS_BADGE="${10:-grey}"

# Function to add 'v' prefix only if not already present
add_version_prefix() {
  local version="$1"
  if [[ "$version" =~ ^v ]]; then
    echo "$version"
  else
    echo "v$version"
  fi
}

# Add 'v' prefix to versions if not already present
RUNNER_VERSION=$(add_version_prefix "$RUNNER_VERSION")
TERRAFORM_VERSION=$(add_version_prefix "$TERRAFORM_VERSION")
NODEJS_VERSION=$(add_version_prefix "$NODEJS_VERSION")

# Get current timestamp
TIMESTAMP=$(date -u +"%d %b %Y %H:%M UTC")

# Create a temporary file for the new status section
STATUS_FILE=$(mktemp)
TMP_README=$(mktemp)

# Set up cleanup trap
trap 'rm -f "$STATUS_FILE" "$TMP_README"' EXIT

# Generate release URLs (strip 'v' prefix for URL construction where needed)
RUNNER_URL="https://github.com/actions/runner/releases/tag/${RUNNER_VERSION}"
TERRAFORM_URL="https://github.com/hashicorp/terraform/releases/tag/${TERRAFORM_VERSION}"
NODEJS_URL="https://github.com/nodejs/node/releases/tag/${NODEJS_VERSION}"

cat > "$STATUS_FILE" <<EOF
## Daily Version Checks

**Last updated:** ${TIMESTAMP}

| Repository | Status | Latest Version | Command |
|------------|--------|----------------|---------|
| [GitHub Actions Runner](${RUNNER_URL}) | ![Status](https://img.shields.io/badge/${RUNNER_STATUS}-${RUNNER_BADGE}) | \`${RUNNER_VERSION}\` | \`github-release-version-checker\` |
| [Terraform](${TERRAFORM_URL}) | ![Status](https://img.shields.io/badge/${TERRAFORM_STATUS}-${TERRAFORM_BADGE}) | \`${TERRAFORM_VERSION}\` | \`github-release-version-checker --repo hashicorp/terraform\` |
| [Node.js](${NODEJS_URL}) | ![Status](https://img.shields.io/badge/${NODEJS_STATUS}-${NODEJS_BADGE}) | \`${NODEJS_VERSION}\` | \`github-release-version-checker --repo node\` |

### GitHub Actions Runner Release Timeline

\`\`\`text
${RUNNER_OUTPUT}
\`\`\`

EOF

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

# Validate the temporary file before replacing
if [[ -s "$TMP_README" ]] && grep -q "## Daily Version Checks" "$TMP_README"; then
  mv "$TMP_README" README.md
  echo "✅ Updated README.md with version status"
else
  echo "❌ Error: Temporary README file is empty or malformed. Aborting update." >&2
  exit 1
fi
