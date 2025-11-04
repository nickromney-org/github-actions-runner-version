# GitHub Actions Integration

Complete guide to using the GitHub Release Version Checker in GitHub Actions workflows.

## Table of Contents

- [Quick Start](#quick-start)
- [CI Output Format](#ci-output-format)
- [Self-Hosted Runners](#self-hosted-runners)
- [Multiple Repositories](#multiple-repositories)
- [Advanced Workflows](#advanced-workflows)
- [Examples](#examples)

## Quick Start

### Basic Workflow

Check a GitHub Actions runner version daily:

```yaml
name: Check Runner Version
on:
 schedule:
 - cron: "0 9 * * *" # Daily at 9 AM UTC
 workflow_dispatch:

jobs:
 check:
 runs-on: self-hosted
 steps:
 - name: Get runner version
 id: version
 run: |
 VERSION=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
 echo "version=$VERSION" >> $GITHUB_OUTPUT

 - name: Check version
 run: github-release-version-checker -c ${{ steps.version.outputs.version }} --ci
```

### Install Binary in Workflow

If the binary isn't already installed on your runner:

```yaml
- name: Download version checker
 run: |
 REPO="nickromney-org/github-release-version-checker"
 BINARY="github-release-version-checker-linux-amd64"
 URL="https://github.com/${REPO}/releases/latest/download"
 curl -LO "${URL}/${BINARY}"
 chmod +x "${BINARY}"
 sudo mv "${BINARY}" /usr/local/bin/github-release-version-checker
```

## CI Output Format

The `--ci` flag provides GitHub Actions-specific formatting:

### Collapsible Sections

```text
::group:: Runner Version Check
Latest version: v2.329.0
Your version: v2.328.0
Status: Behind
::endgroup::
```

### Annotations

Different status levels use appropriate annotation commands:

- **Current**: `::notice::` - Informational message
- **Warning**: `::notice::` - Update available
- **Critical**: `::warning::` - Update urgently
- **Expired**: `::error::` - Update immediately

### Job Summary

Automatically writes a markdown summary to `$GITHUB_STEP_SUMMARY`:

- Status badge
- Version comparison table
- Release timeline
- Clickable links to GitHub releases

## Self-Hosted Runners

### Detect Runner Version

```yaml
- name: Get runner version
 id: version
 run: |
 VERSION=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
 echo "version=$VERSION" >> $GITHUB_OUTPUT

- name: Check version
 run: github-release-version-checker -c ${{ steps.version.outputs.version }} --ci
```

### Create Issue on Expiration

```yaml
- name: Check version
 id: check
 run: github-release-version-checker -c ${{ steps.version.outputs.version }} --ci
 continue-on-error: true

- name: Create issue if expired
 if: steps.check.conclusion == 'failure'
 uses: actions/github-script@v7
 with:
 script: |
 const version = '${{ steps.version.outputs.version }}';
 const title = ` Self-hosted runner version ${version} is expired`;
 const body = `
 ## Runner Version Expired

 **Runner:** \`${{ runner.name }}\`
 **Current Version:** \`${version}\`

 The runner version has exceeded the 30-day update policy
 and will no longer receive jobs from GitHub Actions.

 ### Action Required
 Update the runner to the latest version immediately.

 See the [workflow run](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}) for details.
 `;

 // Check if issue already exists
 const issues = await github.rest.issues.listForRepo({
 owner: context.repo.owner,
 repo: context.repo.repo,
 state: 'open',
 labels: 'runner-version-expired'
 });

 const existingIssue = issues.data.find(issue =>
 issue.title.includes(version) && issue.title.includes('expired')
 );

 if (!existingIssue) {
 await github.rest.issues.create({
 owner: context.repo.owner,
 repo: context.repo.repo,
 title: title,
 body: body,
 labels: ['runner', 'runner-version-expired', 'critical']
 });
 }
```

### Send Slack Notification

```yaml
- name: Check version
 id: check
 run: |
 OUTPUT=$(github-release-version-checker -c ${{ steps.version.outputs.version }} --json)
 echo "output=$OUTPUT" >> $GITHUB_OUTPUT

 STATUS=$(echo "$OUTPUT" | jq -r '.status')
 if [ "$STATUS" = "expired" ] || [ "$STATUS" = "critical" ]; then
 exit 1
 fi
 continue-on-error: true

- name: Notify Slack on failure
 if: steps.check.conclusion == 'failure'
 uses: slackapi/slack-github-action@v1
 with:
 payload: |
 {
 "text": " Runner version check failed",
 "blocks": [
 {
 "type": "section",
 "text": {
 "type": "mrkdwn",
 "text": "*Runner Version Alert*\n\nRunner \`${{ runner.name }}\` version \`${{ steps.version.outputs.version }}\` needs attention.\n\nStatus: ${{ fromJSON(steps.check.outputs.output).status }}\n\n<${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}|View workflow run>"
 }
 }
 ]
 }
 env:
 SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
```

## Multiple Repositories

Check versions for multiple tools in a single workflow:

```yaml
name: Daily Version Checks
on:
 schedule:
 - cron: "0 9 * * *"
 workflow_dispatch:

jobs:
 check-versions:
 runs-on: ubuntu-latest
 steps:
 - name: Download version checker
 run: |
 REPO="nickromney-org/github-release-version-checker"
 BINARY="github-release-version-checker-linux-amd64"
 URL="https://github.com/${REPO}/releases/latest/download"
 curl -LO "${URL}/${BINARY}"
 chmod +x "${BINARY}"
 sudo mv "${BINARY}" /usr/local/bin/github-release-version-checker

 - name: Check GitHub Actions Runner
 run: |
 LATEST=$(github-release-version-checker)
 echo "::notice::GitHub Actions Runner latest: $LATEST"
 github-release-version-checker --ci

 - name: Check Kubernetes
 run: |
 LATEST=$(github-release-version-checker --repo k8s)
 echo "::notice::Kubernetes latest: $LATEST"
 github-release-version-checker --repo k8s --ci

 - name: Check Node.js
 run: |
 LATEST=$(github-release-version-checker --repo node)
 echo "::notice::Node.js latest: $LATEST"
 github-release-version-checker --repo node --ci

 - name: Check Terraform
 run: |
 LATEST=$(github-release-version-checker --repo hashicorp/terraform)
 echo "::notice::Terraform latest: $LATEST"
 github-release-version-checker --repo hashicorp/terraform --ci
```

## Advanced Workflows

### Matrix Strategy for Multiple Runners

Check multiple self-hosted runners:

```yaml
name: Check All Runners
on:
 schedule:
 - cron: "0 9 * * *"

jobs:
 check-runners:
 strategy:
 matrix:
 runner:
 - name: runner-1
 labels: [self-hosted, linux, x64, runner-1]
 - name: runner-2
 labels: [self-hosted, linux, x64, runner-2]
 - name: runner-3
 labels: [self-hosted, linux, arm64, runner-3]

 runs-on: ${{ matrix.runner.labels }}
 steps:
 - name: Get runner version
 id: version
 run: |
 VERSION=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
 echo "version=$VERSION" >> $GITHUB_OUTPUT

 - name: Check version for ${{ matrix.runner.name }}
 run: |
 echo "::group::Checking ${{ matrix.runner.name }}"
 github-release-version-checker -c ${{ steps.version.outputs.version }} --ci
 echo "::endgroup::"
```

### Conditional Checks with Version Comparison

```yaml
- name: Get versions
 id: versions
 run: |
 CURRENT=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
 LATEST=$(github-release-version-checker)
 echo "current=$CURRENT" >> $GITHUB_OUTPUT
 echo "latest=$LATEST" >> $GITHUB_OUTPUT

- name: Check if update needed
 id: check
 run: |
 OUTPUT=$(github-release-version-checker -c ${{ steps.versions.outputs.current }} --json)
 echo "output=$OUTPUT" >> $GITHUB_OUTPUT

 STATUS=$(echo "$OUTPUT" | jq -r '.status')
 echo "status=$STATUS" >> $GITHUB_OUTPUT

 if [ "$STATUS" = "current" ]; then
 echo "::notice::Runner is up to date"
 elif [ "$STATUS" = "warning" ]; then
 echo "::warning::Runner update available"
 elif [ "$STATUS" = "critical" ]; then
 echo "::warning::Runner update urgent"
 exit 1
 elif [ "$STATUS" = "expired" ]; then
 echo "::error::Runner version expired"
 exit 1
 fi

- name: Auto-update runner
 if: steps.check.outputs.status != 'current'
 run: |
 echo "Triggering runner update workflow..."
 # Your update logic here
```

### Scheduled Checks with Different Frequencies

```yaml
name: Runner Version Monitoring
on:
 schedule:
 # Quick check every 6 hours
 - cron: "0 */6 * * *"
 # Detailed check daily at 9 AM
 - cron: "0 9 * * *"
 workflow_dispatch:
 inputs:
 detailed:
 description: 'Run detailed check'
 type: boolean
 default: false

jobs:
 quick-check:
 if: github.event.schedule == '0 */6 * * *' || (github.event_name == 'workflow_dispatch' && !inputs.detailed)
 runs-on: self-hosted
 steps:
 - name: Quick status check
 run: |
 VERSION=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
 github-release-version-checker -c "$VERSION" --json | jq -r '.status'

 detailed-check:
 if: github.event.schedule == '0 9 * * *' || (github.event_name == 'workflow_dispatch' && inputs.detailed)
 runs-on: self-hosted
 steps:
 - name: Detailed version check
 run: |
 VERSION=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
 github-release-version-checker -c "$VERSION" --ci -v
```

### Integration with Terraform/Infrastructure

Check versions and trigger infrastructure updates:

```yaml
name: Infrastructure Version Checks
on:
 schedule:
 - cron: "0 10 * * MON" # Weekly on Monday

jobs:
 check-dependencies:
 runs-on: ubuntu-latest
 outputs:
 terraform-outdated: ${{ steps.terraform.outputs.outdated }}
 kubernetes-outdated: ${{ steps.kubernetes.outputs.outdated }}
 steps:
 - name: Check Terraform version
 id: terraform
 run: |
 CURRENT="1.5.0" # From your infrastructure
 OUTPUT=$(github-release-version-checker --repo hashicorp/terraform -c "$CURRENT" --json)
 STATUS=$(echo "$OUTPUT" | jq -r '.status')

 if [ "$STATUS" != "current" ]; then
 echo "outdated=true" >> $GITHUB_OUTPUT
 echo "::warning::Terraform version $CURRENT is outdated"
 else
 echo "outdated=false" >> $GITHUB_OUTPUT
 fi

 - name: Check Kubernetes version
 id: kubernetes
 run: |
 CURRENT="1.28.0" # From your cluster
 OUTPUT=$(github-release-version-checker --repo k8s -c "$CURRENT" --json)
 STATUS=$(echo "$OUTPUT" | jq -r '.status')

 if [ "$STATUS" != "current" ]; then
 echo "outdated=true" >> $GITHUB_OUTPUT
 echo "::warning::Kubernetes version $CURRENT is outdated"
 else
 echo "outdated=false" >> $GITHUB_OUTPUT
 fi

 update-infrastructure:
 needs: check-dependencies
 if: needs.check-dependencies.outputs.terraform-outdated == 'true' || needs.check-dependencies.outputs.kubernetes-outdated == 'true'
 runs-on: ubuntu-latest
 steps:
 - name: Create update issue
 uses: actions/github-script@v7
 with:
 script: |
 await github.rest.issues.create({
 owner: context.repo.owner,
 repo: context.repo.repo,
 title: 'Infrastructure version updates available',
 body: 'Automated check found outdated dependencies. Review and update.',
 labels: ['infrastructure', 'dependencies']
 });
```

## Examples

### Complete Self-Hosted Runner Check

See [.github/workflows/check-runner.yml](../.github/workflows/check-runner.yml) for a complete example with:

- Auto-detection of runner version
- Issue creation on expiration
- Slack notifications
- Failure handling
- Demonstration modes for GitHub-hosted runners

### JSON Processing Example

```yaml
- name: Check and process JSON output
 run: |
 OUTPUT=$(github-release-version-checker -c "$VERSION" --json)

 # Extract fields
 LATEST=$(echo "$OUTPUT" | jq -r '.latest_version')
 STATUS=$(echo "$OUTPUT" | jq -r '.status')
 RELEASES_BEHIND=$(echo "$OUTPUT" | jq -r '.releases_behind')
 DAYS_SINCE=$(echo "$OUTPUT" | jq -r '.days_since_update')

 # Build custom message
 echo "::notice::Runner: $VERSION"
 echo "::notice::Latest: $LATEST"
 echo "::notice::Status: $STATUS"
 echo "::notice::Releases behind: $RELEASES_BEHIND"
 echo "::notice::Days since update: $DAYS_SINCE"

 # Set outputs for later steps
 echo "status=$STATUS" >> $GITHUB_OUTPUT
 echo "latest=$LATEST" >> $GITHUB_OUTPUT
```

### Error Handling Example

```yaml
- name: Check version with error handling
 id: check
 run: |
 set +e # Don't exit on error

 OUTPUT=$(github-release-version-checker -c "$VERSION" --json 2>&1)
 EXIT_CODE=$?

 if [ $EXIT_CODE -eq 0 ]; then
 echo "::notice::Version check succeeded"
 echo "output=$OUTPUT" >> $GITHUB_OUTPUT
 else
 echo "::error::Version check failed: $OUTPUT"
 echo "failed=true" >> $GITHUB_OUTPUT
 fi

 exit $EXIT_CODE
 continue-on-error: true

- name: Handle failure
 if: steps.check.outputs.failed == 'true'
 run: |
 echo "::error::Unable to check runner version"
 # Notification logic here
```

## Best Practices

### 1. Use GitHub Token

Avoid rate limiting by providing `GITHUB_TOKEN`:

```yaml
- name: Check version
 env:
 GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
 run: github-release-version-checker -c "$VERSION" --ci
```

### 2. Cache the Binary

For workflows that run frequently, cache the binary:

```yaml
- name: Cache version checker
 uses: actions/cache@v4
 id: cache
 with:
 path: /usr/local/bin/github-release-version-checker
 key: version-checker-${{ runner.os }}

- name: Download version checker
 if: steps.cache.outputs.cache-hit != 'true'
 run: |
 # Download and install binary
```

### 3. Use continue-on-error

Allow workflows to complete even if checks fail:

```yaml
- name: Check version
 run: github-release-version-checker -c "$VERSION" --ci
 continue-on-error: true
```

### 4. Separate Concerns

Use multiple jobs for different responsibilities:

```yaml
jobs:
 check:
 # Just check versions

 notify:
 needs: check
 if: failure()
 # Handle notifications

 update:
 needs: check
 if: always()
 # Trigger updates
```

### 5. Use Workflow Outputs

Share version information between jobs:

```yaml
jobs:
 check:
 outputs:
 version: ${{ steps.check.outputs.version }}
 status: ${{ steps.check.outputs.status }}
 steps:
 - id: check
 run: |
 OUTPUT=$(github-release-version-checker -c "$VERSION" --json)
 echo "version=$(echo "$OUTPUT" | jq -r '.comparison_version')" >> $GITHUB_OUTPUT
 echo "status=$(echo "$OUTPUT" | jq -r '.status')" >> $GITHUB_OUTPUT

 process:
 needs: check
 runs-on: ubuntu-latest
 steps:
 - run: echo "Version ${{ needs.check.outputs.version }} is ${{ needs.check.outputs.status }}"
```

## Troubleshooting

### Binary Not Found

Ensure the binary is installed or download it in the workflow:

```yaml
- name: Verify binary exists
 run: which github-release-version-checker || echo "Binary not found"
```

### Rate Limiting

Provide a GitHub token to avoid rate limits:

```yaml
env:
 GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Self-Hosted Runner Issues

Verify `$RUNNER_HOME` is set:

```yaml
- name: Debug runner environment
 run: |
 echo "RUNNER_HOME: $RUNNER_HOME"
 ls -la "$RUNNER_HOME/.runner"
```

## Next Steps

- [CLI Usage Guide](CLI-USAGE.md) - Learn CLI commands
- [Installation Guide](INSTALLATION.md) - Install the tool
- [Library Usage](LIBRARY-USAGE.md) - Use as a Go library
