# 🏃 GitHub Actions Runner Version Checker (Go)

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A blazingly fast, type-safe CLI tool to check if your GitHub Actions self-hosted runner version complies with GitHub's [30-day update policy](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners/about-self-hosted-runners#about-self-hosted-runner-updates).

> **GitHub's Policy**: Any updates released for the software, including major, minor, or patch releases, are considered as an available update. If you do not perform a software update within 30 days, the GitHub Actions service will not queue jobs to your runner.

## ✨ Features

- ⚡ **Lightning Fast** - Single binary, ~10ms startup time
- 🔒 **Type Safe** - Written in Go with strong typing throughout
- 🎨 **Beautiful Output** - Colorized terminal output with emojis
- 📊 **Multiple Formats** - Terminal UI or JSON for automation
- 🔢 **Semantic Versioning** - Proper major.minor.patch comparison
- 🕐 **Accurate Age Tracking** - Calculates from first newer release
- 💾 **Embedded Cache** - Historical releases built-in, minimal API calls
- 🐳 **Docker Ready** - Multi-stage builds, tiny images
- 🧪 **Well Tested** - Comprehensive unit tests included
- 📦 **Zero Dependencies** - Single static binary, no runtime needed

## 🚀 Quick Start

### Installation

**Option 1: Download Binary**

Download the latest release for your platform from [GitHub Releases](https://github.com/nickromney-org/github-actions-runner-version/releases/latest):

```bash
# Linux (x64)
curl -LO https://github.com/nickromney-org/github-actions-runner-version/releases/latest/download/github-release-version-checker-linux-amd64
chmod +x github-release-version-checker-linux-amd64
sudo mv github-release-version-checker-linux-amd64 /usr/local/bin/github-release-version-checker

# macOS (Intel)
curl -LO https://github.com/nickromney-org/github-actions-runner-version/releases/latest/download/github-release-version-checker-darwin-amd64
chmod +x github-release-version-checker-darwin-amd64
xattr -d com.apple.quarantine github-release-version-checker-darwin-amd64  # Remove macOS quarantine
sudo mv github-release-version-checker-darwin-amd64 /usr/local/bin/github-release-version-checker

# macOS (Apple Silicon)
curl -LO https://github.com/nickromney-org/github-actions-runner-version/releases/latest/download/github-release-version-checker-darwin-arm64
chmod +x github-release-version-checker-darwin-arm64
xattr -d com.apple.quarantine github-release-version-checker-darwin-arm64  # Remove macOS quarantine
sudo mv github-release-version-checker-darwin-arm64 /usr/local/bin/github-release-version-checker
```

> **Note for macOS users**: Downloaded binaries are not code-signed with an Apple Developer certificate. The `xattr -d com.apple.quarantine` command removes the Gatekeeper quarantine attribute. Alternatively, you can build from source (see Option 2 below).

**Option 2: Build from Source**

Building from source bypasses any code-signing issues and ensures you're running code you've verified:

```bash
git clone https://github.com/nickromney-org/github-actions-runner-version.git
cd github-actions-runner-version
make build

# Binary will be in bin/github-release-version-checker
sudo mv bin/github-release-version-checker /usr/local/bin/
```

**Option 3: Install with Go**

```bash
go install github.com/nickromney-org/github-actions-runner-version@latest
```

### Basic Usage

**GitHub Actions Runner (default - days-based policy)**

```bash
# Check latest version
github-release-version-checker

# Check a specific version
github-release-version-checker -c 2.328.0

# Verbose output
github-release-version-checker -c 2.328.0 -v

# JSON output for automation
github-release-version-checker -c 2.328.0 --json

# CI output for GitHub Actions
github-release-version-checker -c 2.328.0 --ci
```

**Kubernetes (version-based policy: 3 minor versions)**

```bash
github-release-version-checker --repo kubernetes/kubernetes -c 1.31.12
github-release-version-checker --repo kubernetes/kubernetes -c 1.28.0
```

**Other Popular Tools**

```bash
# Pulumi (version-based policy)
github-release-version-checker --repo pulumi/pulumi -c 3.204.0

# HashiCorp Terraform (version-based policy)
github-release-version-checker --repo hashicorp/terraform -c 1.11.1

# Arkade (version-based policy)
github-release-version-checker --repo alexellis/arkade -c 0.11.50
```

**Advanced Options**

```bash
# Bypass embedded cache (always fetch from API)
github-release-version-checker -c 2.328.0 --no-cache

# With GitHub token (to avoid rate limiting)
github-release-version-checker -c 2.328.0 -t $GITHUB_TOKEN

# Quiet mode (suppress timeline table)
github-release-version-checker -c 2.328.0 -q
```

## 📖 Usage Examples

### Example 1: Check Latest Version

```bash
$ github-actions-runner-version
2.329.0
```

Perfect for scripts:

```bash
LATEST_VERSION=$(github-actions-runner-version)
echo "Latest runner version is: $LATEST_VERSION"
```

### Example 2: Check Expired Version

```bash
$ github-release-version-checker -c 2.327.1
2.329.0

🚨 Version 2.327.1 (25 Jul 2025) EXPIRED 12 Sep 2025: Update to v2.329.0 (Released 14 Oct 2025)

📅 Release Expiry Timeline
─────────────────────────────────────────────────────
Version    Release Date   Expiry Date    Status
2.327.0    22 Jul 2025    24 Aug 2025    ❌ Expired 67 days ago
2.327.1    25 Jul 2025    12 Sep 2025    ❌ Expired 48 days ago  ← Your version
2.328.0    13 Aug 2025    13 Nov 2025    ✅ Valid (13 days left)
2.329.0    14 Oct 2025    -              ✅ Latest (16 days ago)
```

### Example 2a: Quiet Output (suppress expiry table)

```bash
$ github-release-version-checker -c 2.327.1 -q
2.329.0

🚨 Version 2.327.1 (25 Jul 2025) EXPIRED 12 Sep 2025: Update to v2.329.0 (Released 14 Oct 2025)
```

### Example 3: Verbose Output

```bash
$ github-release-version-checker -c 2.328.0 -v
2.329.0

⚠️  Version 2.328.0 Warning: 1 release behind

   📦 Update available: v2.329.0
      Released: Oct 14, 2024 (3 days ago)
   🎯 Latest version: v2.329.0

📊 Detailed Analysis
─────────────────────────────────────
  Current version:      v2.328.0
  Latest version:       v2.329.0
  Status:               warning
  Releases behind:      1
  First newer release:  v2.329.0
  Released on:          2024-10-14
  Days since update:    3
  Days until expired:   27

📋 Available Updates
─────────────────────────────────────
  • v2.329.0 (2024-10-14, 3 days ago)
```

### Example 4: JSON Output

```bash
$ github-release-version-checker -c 2.327.1 --json
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

### Example 5: CI/GitHub Actions Output

```bash
$ github-release-version-checker -c 2.327.1 --ci
2.329.0

::group::📊 Runner Version Check
Latest version: v2.329.0
Your version: v2.327.1
Status: Expired
::endgroup::

::error title=Runner Version Expired::🚨 Version 2.327.1 EXPIRED! (2 releases behind AND 35 days overdue)
::error::Update required: v2.328.0 was released 65 days ago
::error::Latest version: v2.329.0

::group::📋 Available Updates
  • v2.329.0 (2024-10-14, 3 days ago) [Latest]
  • v2.328.0 (2024-08-13, 65 days ago) [First newer release]
::endgroup::
```

**Plus** a beautiful markdown summary in the GitHub Actions job summary! See [CI-OUTPUT.md](CI-OUTPUT.md) for more examples.

## 🎯 Command Line Options

```bash
Usage:
  github-release-version-checker [flags]

Flags:
  -c, --compare string      version to compare against (e.g., 2.327.1)
  -d, --critical-days int   days before critical warning (default 12)
  -m, --max-days int        days before version expires (default 30)
  -v, --verbose            verbose output
      --json               output as JSON
      --ci                 format output for CI/GitHub Actions
  -q, --quiet              quiet output (suppress expiry table)
  -n, --no-cache           bypass embedded cache and always fetch from GitHub API
  -t, --token string       GitHub token (or GITHUB_TOKEN env var)
      --version            show version information
  -h, --help              help for github-actions-runner-version
```

## 🔄 Using in GitHub Actions

The `--ci` flag formats output perfectly for GitHub Actions with:

- Collapsible sections using `::group::`
- Error/warning annotations
- Beautiful markdown job summaries
- Clickable links to releases

### Quick Start

```yaml
name: Check Runner Version
on:
  schedule:
    - cron: "0 9 * * *" # Daily at 9 AM

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

See [.github/workflows/check-runner.yml](.github/workflows/check-runner.yml) for a complete example with:

- Auto-detection of runner version
- Issue creation on expiration
- Slack notifications
- Failure handling

## 🏗️ Building

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Build Commands

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt

# Install locally
make install

# Clean build artifacts
make clean
```

### Cross-Platform Builds

```bash
# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o bin/github-release-version-checker-darwin-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bin/github-release-version-checker-darwin-arm64

# Linux (x64)
GOOS=linux GOARCH=amd64 go build -o bin/github-release-version-checker-linux-amd64

# Linux (ARM64)
GOOS=linux GOARCH=arm64 go build -o bin/github-release-version-checker-linux-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/github-release-version-checker-windows-amd64.exe
```

## 🐳 Docker Usage

Docker images are not automatically published with releases, but you can build your own:

```bash
# Build Docker image locally
docker build -t github-actions-runner-version:latest .

# Run in Docker
docker run --rm github-actions-runner-version:latest

# With comparison version
docker run --rm github-actions-runner-version:latest -c 2.327.1

# With GitHub token
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN github-actions-runner-version:latest -c 2.327.1 -v
```

## 🔧 Integration Examples

### In GitHub Actions (Recommended)

```yaml
- name: Check runner version
  run: |
    VERSION=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
    github-release-version-checker -c "$VERSION" --ci
```

This gives you:

- ✅ Beautiful formatted output in logs
- ✅ Error/warning annotations
- ✅ Markdown summary table
- ✅ Clickable links to releases

See [CI-OUTPUT.md](CI-OUTPUT.md) for examples of what the output looks like.

### In Shell Scripts

```bash
#!/bin/bash
set -e

VERSION=$(cat /opt/actions-runner/.runner | jq -r '.agentVersion')
OUTPUT=$(github-release-version-checker -c "$VERSION" --json)
STATUS=$(echo "$OUTPUT" | jq -r '.status')

if [ "$STATUS" = "expired" ]; then
    echo "❌ Runner version is expired! Please update immediately."
    exit 1
elif [ "$STATUS" = "critical" ]; then
    echo "⚠️  Runner version is critical. Update soon."
    exit 0
else
    echo "✅ Runner version is current."
fi
```

### In CI/CD (Other Systems)

For non-GitHub CI systems, use JSON output:

```bash
#!/bin/bash
# Jenkins, GitLab CI, CircleCI, etc.

OUTPUT=$(github-release-version-checker -c "$VERSION" --json)
IS_EXPIRED=$(echo "$OUTPUT" | jq -r '.is_expired')

if [ "$IS_EXPIRED" = "true" ]; then
    echo "Runner version check FAILED"
    exit 1
fi
```

### In Monitoring (Prometheus)

If you need metrics export, you can parse JSON output:

```bash
#!/bin/bash
# Export metrics for Prometheus node_exporter textfile collector

OUTPUT=$(github-release-version-checker -c "$RUNNER_VERSION" --json)

cat > /var/lib/node_exporter/textfile/runner.prom <<EOF
# HELP runner_is_expired Whether the runner version is expired
# TYPE runner_is_expired gauge
runner_is_expired $(echo $OUTPUT | jq -r '.is_expired | if . then 1 else 0 end')

# HELP runner_releases_behind Number of releases behind
# TYPE runner_releases_behind gauge
runner_releases_behind $(echo $OUTPUT | jq -r '.releases_behind')

# HELP runner_days_since_update Days since a newer version was released
# TYPE runner_days_since_update gauge
runner_days_since_update $(echo $OUTPUT | jq -r '.days_since_update')
EOF
```

## 📊 Project Structure

```text
.
├── main.go                          # Entry point
├── cmd/
│   └── root.go                     # CLI commands (Cobra)
├── internal/
│   ├── version/
│   │   ├── types.go                # Type definitions
│   │   ├── checker.go              # Core analysis logic
│   │   └── checker_test.go         # Unit tests
│   └── github/
│       └── client.go               # GitHub API client
├── go.mod                          # Dependencies
├── go.sum                          # Dependency checksums
├── Makefile                        # Build automation
├── Dockerfile                      # Container build
└── README.md                       # This file
```

## 🧪 Testing

Run the comprehensive test suite:

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test -v ./internal/version -run TestAnalyze_ExpiredVersion
```

Example test output:

```text
=== RUN   TestAnalyze_ExpiredVersion
--- PASS: TestAnalyze_ExpiredVersion (0.00s)
=== RUN   TestAnalyze_CriticalVersion
--- PASS: TestAnalyze_CriticalVersion (0.00s)
PASS
coverage: 87.3% of statements
ok      github.com/yourusername/runner-version-checker/internal/version    0.234s
```

## 🎨 Why Go?

This tool is written in Go for several compelling reasons:

1. **Single Binary** - No runtime dependencies, ship one file
2. **Fast Startup** - ~10ms vs Node's ~500ms vs Python's ~200ms
3. **Cross-Compilation** - Build for any OS/arch from anywhere
4. **Strong Typing** - Catch bugs at compile time
5. **Excellent Standard Library** - HTTP, JSON, time handling built-in
6. **Perfect for CLI Tools** - This is Go's sweet spot (kubectl, helm, gh)
7. **Easy Distribution** - Just download and run
8. **Low Memory** - ~3-5MB resident memory
9. **Great Ecosystem** - Cobra, semver, go-github libraries

### Performance Comparison

| Implementation    | Startup Time | Binary Size | Memory Usage |
| ----------------- | ------------ | ----------- | ------------ |
| **Go**            | ~10ms        | 8MB         | 5MB          |
| Bash + jq         | ~5ms         | N/A         | 2MB          |
| TypeScript (Node) | ~500ms       | N/A         | 40MB         |
| Python            | ~200ms       | N/A         | 20MB         |

## 💾 Release Data Caching

This tool embeds historical release data to minimize GitHub API calls:

- **Embedded cache**: All releases from v2.159.0 to build time
- **API calls**: Always fetches 5 most recent releases (1 API call)
- **Validation**: Checks if embedded cache is current (within top 5)
- **Fallback**: Full API query if cache is more than 5 releases behind
- **Updates**: Automated daily checks trigger new tool builds

**Result**: Reduces API calls from 2 per invocation to 1, improving speed and rate limit usage.

### Cache Architecture

1. **Bootstrap Process**: `scripts/update-releases.sh` fetches all releases from GitHub API
2. **Embedded Data**: `data/releases.json` is embedded in binary via `go:embed`
3. **Runtime Logic**:
   - Load embedded releases (instant, no API call)
   - Fetch 5 most recent from API (1 API call)
   - If latest embedded is in top 5: merge datasets (optimal path)
   - If cache is stale: fall back to full API query (2 API calls total)

### Maintaining the Cache

The cache is automatically updated by GitHub Actions:

- **Check**: `cmd/check-releases` validates embedded data currency
- **Update**: Daily workflow runs at 6 AM UTC
- **Commit**: Auto-commits new releases to trigger rebuild

Manual update:

```bash
# Update releases.json with latest data
./scripts/update-releases.sh

# Rebuild binary with new embedded cache
make build
```

## 🛠️ Development

This project uses British English spelling throughout:
- `Analyse` (not Analyze)
- `colour` (not color)

Linting enforces British spelling via `golangci-lint` with UK locale.

### Commands

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Run with local build
./bin/github-release-version-checker -c 2.327.1
```

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure tests pass (`make test`)
5. Format code (`make fmt`)
6. Run linter (`make lint`)
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🔗 Related Projects

- [GitHub Actions Runner](https://github.com/actions/runner)
- [GitHub CLI](https://github.com/cli/cli) - Another excellent Go CLI tool
- [Semantic Versioning](https://semver.org/)

## 🙏 Acknowledgments

Built with:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [semver](https://github.com/Masterminds/semver) - Semantic versioning
- [go-github](https://github.com/google/go-github) - GitHub API client
- [color](https://github.com/fatih/color) - Terminal colors

---

Made with ❤️ and ☕ - ensuring your runners stay compliant!
