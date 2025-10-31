# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI tool that checks GitHub Actions self-hosted runner versions against GitHub's 30-day update policy. It fetches releases from the `actions/runner` repository via the GitHub API and analyzes whether a given runner version is current, critical, or expired based on semantic versioning and release dates.

## Build and Test Commands

### Building
```bash
# Build for current platform (output: bin/github-actions-runner-version)
make build

# Build for all platforms (darwin/linux/windows, amd64/arm64)
make build-all

# Install to GOPATH/bin
make install

# Build Docker image
make docker-build
```

### Testing
```bash
# Run all tests with race detection and coverage
make test

# Run tests and open HTML coverage report
make test-coverage

# Run specific test
go test -v ./internal/version -run TestAnalyse_ExpiredVersion
```

### Other Development Commands
```bash
# Format code and tidy dependencies
make fmt

# Run linter (installs golangci-lint if needed)
make lint

# Clean build artifacts
make clean

# Run example check
make run
```

## Architecture

### Three-Layer Architecture

1. **CLI Layer** (`cmd/root.go`)
   - Built with Cobra framework
   - Handles flags, output formatting (terminal/JSON/CI), and colourised display (British English)
   - Three output modes: terminal (human-readable), JSON (automation), CI (GitHub Actions annotations + markdown summaries)

2. **Core Logic Layer** (`internal/version/`)
   - `checker.go`: Core analysis engine - compares versions, calculates age from first newer release, determines status
   - `types.go`: Data structures (Analysis, Release, Status enum, CheckerConfig)
   - Status determination: current → warning → critical → expired based on days since first newer release
   - **Key insight**: Age is calculated from the FIRST newer release, not the latest

3. **GitHub Integration Layer** (`internal/github/client.go`)
   - Wraps `go-github/v57` library
   - Fetches from `actions/runner` repository
   - Includes MockClient for testing

### Key Design Patterns

- **Interface-based design**: `GitHubClient` interface allows easy mocking in tests
- **Semantic versioning**: Uses `Masterminds/semver/v3` for proper version comparison
- **Configuration validation**: `CheckerConfig.Validate()` ensures critical days < max days
- **No database**: Stateless tool that fetches fresh data on each run

### Version Analysis Algorithm

The checker uses a multi-step analysis:
1. Fetch latest release from GitHub API
2. If comparison version provided, parse it and fetch all releases
3. Filter releases newer than comparison version, sort oldest-first
4. Calculate days since FIRST newer release (not latest) - this is the key policy metric
5. Determine status based on thresholds:
   - `current`: on latest version
   - `warning`: behind but within critical threshold
   - `critical`: within critical age window (default 12-30 days)
   - `expired`: beyond max age (default 30 days)

### CI Output Features

When using `--ci` flag for GitHub Actions:
- Uses `::group::`/`::endgroup::` for collapsible sections
- Uses `::error::`/`::warning::`/`::notice::` for annotations
- Writes markdown summary to `$GITHUB_STEP_SUMMARY` file with status table and release links

## Repository and Module

- **Repository**: https://github.com/nickromney-org/github-actions-runner-version
- **Module Path**: `github.com/nickromney-org/github-actions-runner-version`
- **Binary Name**: `github-actions-runner-version`

Note: The current go.mod may still reference placeholder paths (`github.com/yourusername/runner-version-checker`) that should be updated to the correct module path above.

## Testing Strategy

Tests use `MockGitHubClient` to simulate various scenarios (current, warning, critical, expired versions). Test helper `newTestRelease()` creates releases with specific ages (days ago) for deterministic testing.

## Dependencies

- `spf13/cobra`: CLI framework
- `Masterminds/semver/v3`: Semantic version parsing and comparison
- `google/go-github/v57`: GitHub API client
- `fatih/color`: Terminal colourisation (imported as `colour`)
- `golang.org/x/oauth2`: GitHub authentication

## GitHub API Authentication

Tool accepts token via `-t` flag or `GITHUB_TOKEN` env var to avoid rate limiting (60 req/hour unauthenticated, 5000 req/hour authenticated).

## British English

This project uses British English spelling throughout the codebase:
- Method names: `Analyse()` (not `Analyze()`)
- Variable names: `colour` (not `color`)
- Comments and documentation use British spelling

The `golangci-lint` configuration enforces British spelling via the `misspell` linter with `locale: UK`.
