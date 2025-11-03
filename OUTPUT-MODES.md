# Output Modes Quick Reference

This tool has three output modes, each designed for different use cases.

## Quick Comparison

| Mode | Flag | Best For | Features |
| ------------ | ----------- | -------------------- | ------------------------------------- |
| **Terminal** | _(default)_ | Local use, debugging | Colors, emojis, human-readable |
| **CI** | `--ci` | GitHub Actions | Annotations, job summary, collapsible |
| **JSON** | `--json` | Automation, scripts | Structured, parseable |

---

## 1⃣ Terminal Mode (Default)

**When to use:** Running locally to check status

**Command:**

```bash
runner-version-check -c 2.327.1
```

**Output:**

```
1.329.0

 Version 2.327.1 EXPIRED: 2 releases behind AND 35 days overdue

 Update available: v2.328.0
 Released: Aug 13, 2024 (65 days ago)
 Latest version: v2.329.0
 2 releases behind
```

**Features:**

- Colorized output (red/yellow/green)
- Emojis for quick visual scanning
- Concise and readable
- Works in any terminal

**With verbose flag (`-v`):**

```
1.329.0

 Version 2.327.1 EXPIRED: 2 releases behind AND 35 days overdue

 Update available: v2.328.0
 Released: Aug 13, 2024 (65 days ago)
 Latest version: v2.329.0
 2 releases behind

 Detailed Analysis
─────────────────────────────────────
 Current version: v2.327.1
 Latest version: v2.329.0
 Status: expired
 Releases behind: 2
 First newer release: v2.328.0
 Released on: 2024-08-13
 Days since update: 65
 Days overdue: 35

 Available Updates
─────────────────────────────────────
 • v2.329.0 (2024-10-14, 3 days ago)
 • v2.328.0 (2024-08-13, 65 days ago)
```

---

## 2⃣ CI Mode

**When to use:** In GitHub Actions workflows

**Command:**

```bash
runner-version-check -c 2.327.1 --ci
```

**Output in Logs:**

```
1.329.0

::group:: Runner Version Check
Latest version: v2.329.0
Your version: v2.327.1
Status: Expired
::endgroup::

::error title=Runner Version Expired:: Version 2.327.1 EXPIRED! (2 releases behind AND 35 days overdue)
::error::Update required: v2.328.0 was released 65 days ago
::error::Latest version: v2.329.0

::group:: Available Updates
 • v2.329.0 (2024-10-14, 3 days ago) [Latest]
 • v2.328.0 (2024-08-13, 65 days ago) [First newer release]
::endgroup::
```

**Features:**

- Uses GitHub Actions workflow commands
- Creates collapsible log sections
- Error/warning annotations appear in UI
- Writes markdown summary to job summary
- Clickable links to releases

**Job Summary (at bottom of workflow run):**

```markdown
## Runner Version Status: Expired

| Metric | Value |
| --------------- | ---------- |
| Current Version | v2.327.1 |
| Latest Version | v2.329.0 |
| Status | Expired |
| Releases Behind | 2 |
| Days Overdue | 35 |

### Action Required

**Update to v2.328.0 or later immediately.** GitHub will not queue jobs to runners with expired versions.

### Available Updates

- [v2.329.0](https://github.com/actions/runner/releases/tag/v2.329.0) - Released Oct 14, 2024 (3 days ago)
- [v2.328.0](https://github.com/actions/runner/releases/tag/v2.328.0) - Released Aug 13, 2024 (65 days ago)
```

**Usage in workflow:**

```yaml
- name: Check runner version
 run: runner-version-check -c ${{ steps.version.outputs.version }} --ci
```

---

## 3⃣ JSON Mode

**When to use:** Automation, scripting, parsing

**Command:**

```bash
runner-version-check -c 2.327.1 --json
```

**Output:**

```json
{
 "latest_version": "2.329.0",
 "comparison_version": "2.327.1",
 "is_latest": false,
 "is_expired": true,
 "is_critical": false,
 "releases_behind": 2,
 "days_since_update": 65,
 "first_newer_version": "2.328.0",
 "first_newer_release_date": "2024-08-13T10:30:00Z",
 "status": "expired",
 "message": "Version 2.327.1 EXPIRED: 2 releases behind AND 35 days overdue",
 "critical_age_days": 12,
 "max_age_days": 30
}
```

**Features:**

- Machine-readable structured data
- Easy to parse with `jq`
- All metrics available
- Stable schema

**Usage in scripts:**

```bash
# Parse with jq
OUTPUT=$(runner-version-check -c 2.327.1 --json)
IS_EXPIRED=$(echo "$OUTPUT" | jq -r '.is_expired')
STATUS=$(echo "$OUTPUT" | jq -r '.status')
DAYS_OVERDUE=$(echo "$OUTPUT" | jq -r '(.days_since_update - .max_age_days)')

# Use in conditionals
if [ "$IS_EXPIRED" = "true" ]; then
 echo "ERROR: Runner expired!"
 exit 1
fi
```

**Schema:**

```typescript
{
 latest_version: string; // Latest available version
 comparison_version: string; // Version being checked
 is_latest: boolean; // True if on latest
 is_expired: boolean; // True if > max_age_days old
 is_critical: boolean; // True if > critical_age_days old
 releases_behind: number; // How many releases behind
 days_since_update: number; // Days since first newer release
 first_newer_version: string; // First version after current
 first_newer_release_date: string; // ISO 8601 timestamp
 status: "current" | "warning" | "critical" | "expired";
 message: string; // Human-readable summary
 critical_age_days: number; // Config: critical threshold
 max_age_days: number; // Config: expiry threshold
}
```

---

## Decision Tree

```
Need to use in GitHub Actions?
│
├─ Yes → Use --ci flag
│ Gets you nice logs + job summary
│
└─ No → Are you automating/scripting?
 │
 ├─ Yes → Use --json flag
 │ Parse with jq or similar
 │
 └─ No → Use default (or -v for details)
 Human-readable terminal output
```

---

## Pro Tips

### Tip 1: Combine Modes

You can use both `--ci` and store JSON output:

```yaml
- name: Check version
 run: |
 # Store JSON for later use
 OUTPUT=$(runner-version-check -c $VERSION --json)
 echo "status=$(echo $OUTPUT | jq -r .status)" >> $GITHUB_OUTPUT

 # Also show nice CI output
 runner-version-check -c $VERSION --ci
```

### Tip 2: CI Mode Works Everywhere

Even though `--ci` uses GitHub Actions commands, it works fine in any environment:

- GitHub Actions: Commands render as collapsible sections and annotations
- Other CI systems: Commands are ignored, output is still readable
- Local terminal: Commands are ignored, looks like normal output

### Tip 3: First Line is Always Version

For script compatibility, the first line of output (in all modes) is always the latest version:

```bash
LATEST=$(runner-version-check | head -n1)
echo "Latest version is: $LATEST"
```

### Tip 4: Verbose Works With All Modes

You can combine `-v` with other flags:

```bash
runner-version-check -c 2.327.1 --ci -v # CI output with extra details
runner-version-check -c 2.327.1 --json -v # JSON output + stderr verbose logs
```

---

## Real-World Examples

### Example 1: Daily CI Check (GitHub Actions)

```yaml
name: Check Runner
on:
 schedule:
 - cron: "0 9 * * *"

jobs:
 check:
 runs-on: self-hosted
 steps:
 - run: |
 VERSION=$(cat $RUNNER_HOME/.runner | jq -r .agentVersion)
 runner-version-check -c $VERSION --ci
```

### Example 2: Fail on Expired

```bash
#!/bin/bash
OUTPUT=$(runner-version-check -c "$VERSION" --json)
IS_EXPIRED=$(echo "$OUTPUT" | jq -r '.is_expired')

if [ "$IS_EXPIRED" = "true" ]; then
 # Show detailed info
 runner-version-check -c "$VERSION" -v
 exit 1
fi
```

### Example 3: Monitor Multiple Runners

```bash
#!/bin/bash
for runner in runner-{01..10}; do
 echo "Checking $runner..."
 ssh $runner 'runner-version-check -c $(cat .runner | jq -r .agentVersion)'
done
```

### Example 4: Send to Slack

```bash
#!/bin/bash
OUTPUT=$(runner-version-check -c "$VERSION" --json)
STATUS=$(echo "$OUTPUT" | jq -r '.status')
MESSAGE=$(echo "$OUTPUT" | jq -r '.message')

if [ "$STATUS" = "expired" ] || [ "$STATUS" = "critical" ]; then
 curl -X POST "$SLACK_WEBHOOK" \
 -H 'Content-Type: application/json' \
 -d "{\"text\":\"$MESSAGE\"}"
fi
```

---

## Troubleshooting

### Issue: Colors not showing

**Solution:** Some terminals don't support colors. The tool auto-detects this and falls back to plain text.

### Issue: CI commands visible in logs

**Solution:** This is normal outside GitHub Actions. The commands are ignored by other systems.

### Issue: Job summary not appearing

**Solution:** Check that `$GITHUB_STEP_SUMMARY` environment variable is set (only in GitHub Actions).

### Issue: JSON parsing error

**Solution:** Make sure you're using the `--json` flag and piping to `jq` correctly.

---

**See also:**

- [CI-OUTPUT.md](CI-OUTPUT.md) - Detailed CI output examples
- [README.md](README.md) - Full documentation
- [.github/workflows/check-runner.yml](.github/workflows/check-runner.yml) - Complete workflow example
