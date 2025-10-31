# 🚀 Quick Start: 5 Minutes to Runner Monitoring

Get up and running with GitHub Actions runner version monitoring in 5 minutes.

## ⚡ Step 1: Install the Tool (30 seconds)

### Option A: Download Binary (Fastest)

```bash
# Linux (x64)
curl -LO https://github.com/yourusername/runner-version-checker/releases/latest/download/runner-version-check-linux-amd64
chmod +x runner-version-check-linux-amd64
sudo mv runner-version-check-linux-amd64 /usr/local/bin/runner-version-check

# macOS (Intel)
curl -LO https://github.com/yourusername/runner-version-checker/releases/latest/download/runner-version-check-darwin-amd64
chmod +x runner-version-check-darwin-amd64
sudo mv runner-version-check-darwin-amd64 /usr/local/bin/runner-version-check

# macOS (Apple Silicon)
curl -LO https://github.com/yourusername/runner-version-checker/releases/latest/download/runner-version-check-darwin-arm64
chmod +x runner-version-check-darwin-arm64
sudo mv runner-version-check-darwin-arm64 /usr/local/bin/runner-version-check
```

### Option B: Build from Source

```bash
git clone https://github.com/yourusername/runner-version-checker.git
cd runner-version-checker
make build
sudo cp bin/runner-version-check /usr/local/bin/
```

### Verify Installation

```bash
runner-version-check --help
```

---

## 📝 Step 2: Test It Locally (1 minute)

Find your runner version:

```bash
# If runner is at default location
cat /opt/actions-runner/.runner | jq -r '.agentVersion'

# Or wherever your runner is installed
cat $RUNNER_HOME/.runner | jq -r '.agentVersion'
```

Run the check:

```bash
runner-version-check -c 2.328.0
```

You should see something like:

```text
2.329.0

ℹ️  Version 2.328.0: 1 release behind

   📦 Update available: v2.329.0
      Released: Oct 14, 2024 (3 days ago)
   🎯 Latest version: v2.329.0
```

---

## 🔄 Step 3: Add to GitHub Actions (2 minutes)

Create `.github/workflows/check-runner.yml`:

```yaml
name: Check Runner Version

on:
  schedule:
    - cron: "0 9 * * *" # Daily at 9 AM UTC
  workflow_dispatch: # Manual trigger

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
        run: runner-version-check -c ${{ steps.version.outputs.version }} --ci
```

Commit and push:

```bash
git add .github/workflows/check-runner.yml
git commit -m "Add runner version check workflow"
git push
```

---

## ✅ Step 4: Test the Workflow (1 minute)

### Trigger Manually

1. Go to your repo on GitHub
2. Click **Actions** tab
3. Select **Check Runner Version** workflow
4. Click **Run workflow**

### View Results

- Check the logs for colorful output
- Scroll to bottom for the **Job Summary** with a nice table

---

## 🎉 Done!

You now have:

- ✅ Daily automated checks
- ✅ Beautiful formatted output in GitHub Actions
- ✅ Job summaries with tables and links
- ✅ A manual trigger option

## 🔥 Next Steps (Optional)

### Add Issue Creation on Expiration

Add this step to your workflow:

```yaml
- name: Create issue if expired
  if: failure()
  uses: actions/github-script@v7
  with:
    script: |
      await github.rest.issues.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title: '🚨 Runner version expired',
        body: 'The runner version has expired. Update immediately.',
        labels: ['runner', 'critical']
      });
```

### Add Slack Notifications

```yaml
- name: Notify Slack
  if: failure()
  uses: slackapi/slack-github-action@v1
  with:
    payload: |
      {
        "text": "🚨 Runner version check failed! Update needed."
      }
  env:
    SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
```

### Check Multiple Runners

If you have multiple self-hosted runners, create separate workflows or use matrix strategy:

```yaml
strategy:
  matrix:
    runner: [runner-01, runner-02, runner-03]
jobs:
  check:
    runs-on: ${{ matrix.runner }}
    # ... rest of the workflow
```

---

## 💡 Pro Tips

### Tip 1: Use GITHUB_TOKEN

The tool respects `GITHUB_TOKEN` environment variable to avoid rate limiting:

```yaml
- name: Check version
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: runner-version-check -c ${{ steps.version.outputs.version }} --ci
```

### Tip 2: Customize Thresholds

```bash
# Warn at 10 days, expire at 25 days (instead of defaults 12/30)
runner-version-check -c 2.328.0 --critical-days 10 --max-days 25 --ci
```

### Tip 3: Run Locally for Debugging

```bash
# Verbose output
runner-version-check -c 2.328.0 -v

# JSON for scripting
runner-version-check -c 2.328.0 --json | jq .
```

---

## 🆘 Troubleshooting

### "Failed to fetch latest release"

**Problem:** Rate limit or network issue
**Solution:** Add `GITHUB_TOKEN` environment variable

### "Could not detect runner version"

**Problem:** `.runner` file not found
**Solution:** Check your `$RUNNER_HOME` path and adjust in workflow

### "Job summary not appearing"

**Problem:** Not using `--ci` flag
**Solution:** Make sure you're using `--ci` flag in GitHub Actions

### "Binary not found after install"

**Problem:** `/usr/local/bin` not in PATH
**Solution:**

```bash
export PATH="/usr/local/bin:$PATH"
# Or install to a different location in your PATH
```

---

## 📚 More Information

- **Full Documentation:** [README.md](README.md)
- **Output Modes:** [OUTPUT-MODES.md](OUTPUT-MODES.md)
- **CI Examples:** [CI-OUTPUT.md](CI-OUTPUT.md)
- **Complete Workflow:** [.github/workflows/check-runner.yml](.github/workflows/check-runner.yml)

---

## 🎯 Summary

| Time    | Step           |
| ------- | -------------- |
| 30s     | Install binary |
| 1m      | Test locally   |
| 2m      | Add workflow   |
| 1m      | Test workflow  |
| **~5m** | **Total**      |

Now you'll get daily checks of your runner version with beautiful formatted output! 🎉
