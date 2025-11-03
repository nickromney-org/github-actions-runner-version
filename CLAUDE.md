# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI tool that checks GitHub release versions against configurable expiry policies. It supports multiple repositories with both time-based (days) and version-based (semantic versioning) policies.

**Multi-Repository Support**: Checks versions for any GitHub repository including:
- GitHub Actions runners (default, 30-day time-based policy)
- Kubernetes (version-based: 3 minor versions behind)
- Pulumi (version-based: 3 minor versions behind)
- Ubuntu releases (time-based: 365 days)
- Custom repositories via owner/repo format or URLs

**Policy Types**:
- **Days Policy**: Time-based expiry (e.g., 30 days for actions/runner)
- **Versions Policy**: Semantic versioning-based (e.g., N minor versions behind)

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

**Build Optimizations**:
- `CGO_ENABLED=0`: Static compilation without C dependencies
- `-trimpath`: Removes absolute file paths for reproducible builds
- `-w -s`: Strips debug information and symbol table
- Binary size: ~9.8MB (darwin/arm64)

### Testing
```bash
# Run all tests with race detection and coverage
make test

# Run tests and open HTML coverage report
make test-coverage

# Run specific test
go test -v ./internal/version -run TestAnalyse_ExpiredVersion

# Run benchmarks
go test -bench=. -benchmem ./internal/version/
```

**Test Coverage** (as of 2025-11-03):
- Overall: 46.3%
- `internal/version`: 89.6%
- `internal/data`: 80.0%
- `internal/github`: 46.9%
- `cmd`: 31.8%

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

### Multi-Repository Architecture

1. **CLI Layer** (`cmd/root.go`)
   - Built with Cobra framework
   - Handles repository selection (`--repo`), policy override (`--policy`), and cache paths
   - Three output modes: terminal (human-readable), JSON (automation), CI (GitHub Actions annotations)
   - Repository resolution: predefined names, owner/repo format, GitHub URLs

2. **Configuration Layer** (`internal/config/`)
   - `repository.go`: RepositoryConfig with predefined configs (actions-runner, kubernetes, pulumi, ubuntu)
   - Repository aliases: "k8s" → kubernetes, "runner" → actions/runner
   - Policy types: days (time-based) and versions (semantic versioning)
   - `ParseRepositoryString()`: Handles all repository input formats

3. **Policy Layer** (`internal/policy/`)
   - `policy.go`: Pluggable policy system via VersionPolicy interface
   - **DaysPolicy**: Time-based expiry (e.g., 30 days)
   - **VersionsPolicy**: Semantic versioning-based (N minor versions behind)
   - PolicyResult: IsExpired, IsCritical, IsWarning, DaysOld, VersionsBehind

4. **Cache Layer** (`internal/cache/`)
   - `manager.go`: Manages embedded and custom caches
   - Priority: custom cache > embedded cache > no cache
   - `go:embed data/*.json`: Multiple embedded caches per repository
   - JSON parsing with intermediate types for proper unmarshaling

5. **Core Logic Layer** (`internal/version/`)
   - `checker.go`: Core analysis engine with policy integration
   - `NewCheckerWithPolicy()`: Creates checker with custom policy
   - Status determination delegated to policy when present
   - **Key insight**: Age calculated from FIRST newer release

6. **GitHub Integration Layer** (`internal/github/`)
   - Dynamic owner/repo support via `NewClient(token, owner, repo)`
   - Supports any GitHub repository
   - MockClient for testing

7. **Type Layer** (`internal/types/`)
   - `release.go`: Shared Release type
   - Breaks import cycles between version and policy packages

### Key Design Patterns

- **Pluggable policies**: VersionPolicy interface for different expiry strategies
- **Repository abstraction**: Single tool for multiple repositories
- **Interface-based design**: GitHubClient, VersionPolicy interfaces
- **Type aliases**: version.Release aliases types.Release for backward compatibility
- **Embedded multi-cache**: Multiple cache files via go:embed
- **No database**: Stateless with embedded data

### Version Analysis Algorithm

The checker uses a multi-step analysis with embedded cache optimization:

1. Load embedded releases from `data/releases.json` (instant, no API call)
2. Fetch 5 most recent releases from GitHub API (1 API call)
3. Validate cache: check if latest embedded release is in top 5
   - If current: merge embedded + recent releases (optimal path, 1 API call total)
   - If stale (>5 releases behind): fall back to `GetAllReleases()` (2 API calls total)
4. If comparison version provided, parse it and filter releases
5. Filter releases newer than comparison version, sort oldest-first
6. Calculate days since FIRST newer release (not latest) - this is the key policy metric
7. Determine status based on thresholds:
   - `current`: on latest version
   - `warning`: behind but within critical threshold
   - `critical`: within critical age window (default 12-30 days)
   - `expired`: beyond max age (default 30 days)

**Cache Strategy**: Reduces API calls from 2 per invocation to 1 in the common case, improving speed and rate limit usage.

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

## Embedded Release Cache

The tool embeds historical release data to minimize API calls during runtime:

### Cache Files and Structure

- **Cache file**: `data/releases.json` - JSON file with all releases from v2.159.0 onwards
- **Embedded**: Via `go:embed` directive in `internal/data/loader.go`
- **Format**:
  ```json
  {
    "generated_at": "2025-10-31T12:00:00Z",
    "releases": [
      {
        "version": "2.329.0",
        "published_at": "2024-10-14T10:30:00Z",
        "url": "https://github.com/actions/runner/releases/tag/v2.329.0"
      }
    ]
  }
  ```

### Maintenance Tools

Three commands manage the cache:

1. **`cmd/bootstrap-releases`**: Fetches all releases from GitHub API and generates `data/releases.json`
   - Used during initial setup and manual updates
   - Run via `scripts/update-releases.sh`

2. **`cmd/check-releases`**: Validates cache currency
   - Checks if latest embedded release is in top 5 recent releases
   - Exit 0 if current, exit 1 if stale
   - Used by automation to trigger updates

3. **`scripts/update-releases.sh`**: Shell wrapper for bootstrap command
   - Sets up environment and runs bootstrap
   - Reports release count after update

### Automated Updates

GitHub Actions workflow (`.github/workflows/update-releases.yml`):
- Runs daily at 6 AM UTC (3 hours before runner compliance checks)
- Executes `check-releases` to validate cache
- If stale: runs `update-releases.sh` and commits changes
- Commit triggers semantic-release for new binary build
- Includes `[skip ci]` to avoid workflow recursion

### Runtime Optimization

The `Analyse()` method in `internal/version/checker.go`:
- Loads embedded cache (instant, no API)
- Fetches 5 most recent releases (1 API call)
- Calls `isEmbeddedCurrent()` to validate cache
- If current: calls `mergeReleases()` to combine datasets
- If stale: falls back to `GetAllReleases()` (additional API call)

## British English

This project uses British English spelling throughout the codebase:
- Method names: `Analyse()` (not `Analyze()`)
- Variable names: `colour` (not `color`)
- Comments and documentation use British spelling

The `golangci-lint` configuration enforces British spelling via the `misspell` linter with `locale: UK`.
