# Installation Guide

This guide covers all methods for installing the GitHub Release Version Checker.

## Option 1: Download Pre-built Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/nickromney-org/github-release-version-checker/releases/latest):

### Linux (x64)

```bash
curl -L -o github-release-version-checker https://github.com/nickromney-org/github-release-version-checker/releases/latest/download/github-release-version-checker-linux-amd64
chmod +x github-release-version-checker
sudo mv github-release-version-checker /usr/local/bin/
```

### Linux (ARM64)

```bash
curl -L -o github-release-version-checker https://github.com/nickromney-org/github-release-version-checker/releases/latest/download/github-release-version-checker-linux-arm64
chmod +x github-release-version-checker
sudo mv github-release-version-checker /usr/local/bin/
```

### macOS (Intel)

```bash
curl -L -o github-release-version-checker https://github.com/nickromney-org/github-release-version-checker/releases/latest/download/github-release-version-checker-darwin-amd64
chmod +x github-release-version-checker
sudo mv github-release-version-checker /usr/local/bin/
```

### macOS (Apple Silicon)

```bash
curl -L -o github-release-version-checker https://github.com/nickromney-org/github-release-version-checker/releases/latest/download/github-release-version-checker-darwin-arm64
chmod +x github-release-version-checker
sudo mv github-release-version-checker /usr/local/bin/
```

### Windows (x64)

```powershell
# Download from GitHub Releases page
# https://github.com/nickromney-org/github-release-version-checker/releases/latest

# Or using PowerShell:
$url = "https://github.com/nickromney-org/github-release-version-checker/releases/latest/download/github-release-version-checker-windows-amd64.exe"
Invoke-WebRequest -Uri $url -OutFile "github-release-version-checker.exe"

# Add to PATH or move to a directory in your PATH
```

### macOS Security Note

This binary is not code-signed with an Apple Developer certificate. If you download via a web browser instead of `curl`, macOS may add a quarantine attribute that blocks execution.

If macOS prevents you from running the binary, remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine github-release-version-checker
```

Alternatively, use Option 2 (build from source) or Option 3 (`go install`) to avoid this issue entirely.

## Option 2: Build from Source

Building from source bypasses any code-signing issues and ensures you're running code you've verified:

```bash
git clone https://github.com/nickromney-org/github-release-version-checker.git
cd github-release-version-checker
make build

# Binary will be in bin/github-release-version-checker
sudo mv bin/github-release-version-checker /usr/local/bin/
```

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Build Commands

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install to GOPATH/bin
make install
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

## Option 3: Install with Go

If you have Go installed, you can install directly from source:

```bash
go install github.com/nickromney-org/github-release-version-checker@latest
```

This installs the binary to `$GOPATH/bin` (usually `~/go/bin`). Make sure this directory is in your PATH.

## Option 4: Docker

Docker images are not automatically published with releases, but you can build your own:

```bash
# Build Docker image locally
docker build -t github-release-version-checker:latest .

# Run in Docker
docker run --rm github-release-version-checker:latest

# With comparison version
docker run --rm github-release-version-checker:latest -c 2.327.1

# With GitHub token
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN github-release-version-checker:latest -c 2.327.1 -v
```

## Verify Installation

After installation, verify it works:

```bash
# Check version
github-release-version-checker --version

# Get latest runner version
github-release-version-checker
```

## GitHub Token (Optional but Recommended)

To avoid GitHub API rate limiting (60 requests/hour unauthenticated, 5000 requests/hour authenticated), provide a GitHub token:

```bash
# Via flag
github-release-version-checker -t $GITHUB_TOKEN -c 2.328.0

# Via environment variable (recommended)
export GITHUB_TOKEN="your_token_here"
github-release-version-checker -c 2.328.0
```

Create a token at: https://github.com/settings/tokens (no scopes required for public repositories)

## Troubleshooting

### macOS: "cannot be opened because it is from an unidentified developer"

Remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine github-release-version-checker
```

### Linux: "Permission denied"

Make the binary executable:

```bash
chmod +x github-release-version-checker
```

### Command not found

Ensure `/usr/local/bin` is in your PATH:

```bash
echo $PATH | grep -q /usr/local/bin || echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## Next Steps

- [CLI Usage Guide](CLI-USAGE.md) - Learn how to use the CLI
- [GitHub Actions Integration](GITHUB-ACTIONS.md) - Use in CI/CD
- [Library Usage](LIBRARY-USAGE.md) - Import as a Go library
