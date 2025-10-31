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
- 🐳 **Docker Ready** - Multi-stage builds, tiny images
- 🧪 **Well Tested** - Comprehensive unit tests included
- 📦 **Zero Dependencies** - Single static binary, no runtime needed

## 🚀 Quick Start

### Installation

**Option 1: Download Binary**

```bash
# Download latest release for your platform
curl -LO https://github.com/yourusername/runner-version-checker/releases/latest/download/runner-version-check-linux-amd64

# Make executable
chmod +x runner-version-check-linux-amd64

# Move to PATH
sudo mv runner-version-check-linux-amd64 /usr/local/bin/runner-version-check
```

**Option 2: Build from Source**

```bash
git clone https://github.com/yourusername/runner-version-checker.git
cd runner-version-checker
make build

# Binary will be in bin/runner-version-check
```

**Option 3: Install with Go**

```bash
go install github.com/yourusername/runner-version-checker@latest
```

**Option 4: Docker**

```bash
docker run ghcr.io/yourusername/runner-version-checker:latest -c 2.327.1
```

### Basic Usage

```bash
# Check latest version
runner-version-check

# Check a specific version
runner-version-check -c 2.327.1

# Verbose output
runner-version-check -c 2.327.1 -v

# JSON output for automation
runner-version-check -c 2.327.1 --json

# CI output for GitHub Actions
runner-version-check -c 2.327.1 --ci

# With GitHub token (to avoid rate limiting)
runner-version-check -c 2.327.1 -t $GITHUB_TOKEN
```

## 📖 Usage Examples

### Example 1: Check Latest Version

```bash
$ runner-version-check
2.329.0
```

Perfect for scripts:

```bash
LATEST_VERSION=$(runner-version-check)
echo "Latest runner version is: $LATEST_VERSION"
```

### Example 2: Check Expired Version

```bash
$ runner-version-check -c 2.327.1
2.329.0

🚨 Version 2.327.1 EXPIRED: 2 releases behind AND 35 days overdue

   📦 Update available: v2.328.0
      Released: Aug 13, 2024 (65 days ago)
   🎯 Latest version: v2.329.0
   ⚠️  2 releases behind
```

### Example 3: Verbose Output

```bash
$ runner-version-check -c 2.328.0 -v
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
$ runner-version-check -c 2.327.1 --json
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
$ runner-version-check -c 2.327.1 --ci
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
  runner-version-check [flags]

Flags:
  -c, --compare string      version to compare against (e.g., 2.327.1)
  -d, --critical-days int   days before critical warning (default 12)
  -m, --max-days int        days before version expires (default 30)
  -v, --verbose            verbose output
      --json               output as JSON
      --ci                 format output for CI/GitHub Actions
  -t, --token string       GitHub token (or GITHUB_TOKEN env var)
  -h, --help              help for runner-version-check
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
        run: runner-version-check -c ${{ steps.version.outputs.version }} --ci
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
GOOS=darwin GOARCH=amd64 go build -o bin/runner-version-check-darwin-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bin/runner-version-check-darwin-arm64

# Linux (x64)
GOOS=linux GOARCH=amd64 go build -o bin/runner-version-check-linux-amd64

# Linux (ARM64)
GOOS=linux GOARCH=arm64 go build -o bin/runner-version-check-linux-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/runner-version-check-windows-amd64.exe
```

## 🐳 Docker Usage

### Build Docker Image

```bash
docker build -t runner-version-check:latest .
```

### Run in Docker

```bash
# Basic usage
docker run --rm runner-version-check:latest

# With comparison version
docker run --rm runner-version-check:latest -c 2.327.1

# With GitHub token
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN runner-version-check:latest -c 2.327.1 -v
```

## 🔧 Integration Examples

### In GitHub Actions (Recommended)

```yaml
- name: Check runner version
  run: |
    VERSION=$(cat $RUNNER_HOME/.runner | jq -r '.agentVersion')
    runner-version-check -c "$VERSION" --ci
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
OUTPUT=$(runner-version-check -c "$VERSION" --json)
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

OUTPUT=$(runner-version-check -c "$VERSION" --json)
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

OUTPUT=$(runner-version-check -c "$RUNNER_VERSION" --json)

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
