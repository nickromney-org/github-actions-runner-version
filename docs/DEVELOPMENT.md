# Development Guide

Guide for developers who want to build, test, and contribute to the GitHub Release Version Checker.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Building](#building)
- [Testing](#testing)
- [Code Style](#code-style)
- [Release Data Caching](#release-data-caching)
- [Contributing](#contributing)

## Prerequisites

- **Go 1.21 or later** - [Install Go](https://golang.org/doc/install)
- **Make** (optional) - For using Makefile targets
- **Git** - For version control
- **GitHub CLI** (optional) - For creating PRs and releases

### Recommended Tools

- **golangci-lint** - Installed automatically by `make lint`
- **jq** - For JSON processing in scripts
- **curl** - For downloading releases

## Getting Started

### Clone the Repository

```bash
git clone https://github.com/nickromney-org/github-release-version-checker.git
cd github-release-version-checker
```

### Install Dependencies

```bash
go mod download
```

### Build the Project

```bash
make build
```

The binary will be created at `bin/github-release-version-checker`.

### Run Tests

```bash
make test
```

### Run the Binary

```bash
./bin/github-release-version-checker
```

## Project Structure

```text
.
├── main.go # Entry point
├── cmd/
│ ├── root.go # CLI commands (Cobra)
│ ├── format.go # Output formatting
│ ├── format_test.go # Format tests
│ ├── bootstrap-releases/ # Cache bootstrap utility
│ └── check-releases/ # Cache validation utility
├── pkg/ # Public API (importable)
│ ├── checker/ # Version analysis engine
│ │ ├── checker.go # Core analysis logic
│ │ ├── types.go # Analysis and Config types
│ │ └── checker_test.go # Unit tests
│ ├── client/ # GitHub API client
│ │ ├── client.go # Client implementation
│ │ └── client_test.go # Client tests
│ ├── policy/ # Expiry policies
│ │ ├── policy.go # Policy implementations
│ │ └── policy_test.go # Policy tests
│ └── types/ # Shared types
│ └── release.go # Release type
├── internal/ # Private implementation
│ ├── version/ # Legacy wrapper around pkg/checker
│ ├── github/ # Legacy wrapper around pkg/client
│ ├── policy/ # Config adapter for policies
│ ├── config/ # Repository configurations
│ │ ├── repository.go # Predefined configs
│ │ └── repository_test.go
│ ├── data/ # Embedded cache loader
│ │ ├── loader.go
│ │ └── releases.json # Embedded release cache
│ ├── cache/ # Cache management
│ │ ├── manager.go
│ │ └── manager_test.go
│ └── types/ # Internal types (aliases pkg/types)
├── examples/ # Library usage examples
│ ├── basic/ # Basic usage example
│ ├── version-based-policy/ # Version policy example
│ ├── custom-repository/ # Custom repo example
│ └── json-output/ # JSON output example
├── docs/ # Documentation
│ ├── INSTALLATION.md
│ ├── CLI-USAGE.md
│ ├── LIBRARY-USAGE.md
│ ├── GITHUB-ACTIONS.md
│ └── DEVELOPMENT.md
├── scripts/ # Utility scripts
│ └── update-releases.sh # Update embedded cache
├── .github/
│ └── workflows/ # GitHub Actions workflows
│ ├── check-runner.yml
│ ├── update-releases.yml
│ └── release.yml
├── go.mod # Dependencies
├── go.sum # Dependency checksums
├── Makefile # Build automation
├── Dockerfile # Container build
├── .golangci.yml # Linter configuration
├── CLAUDE.md # Project instructions for Claude Code
└── README.md # Main documentation
```

### Package Organization

#### Public API (`pkg/`)

These packages are designed to be imported by external Go applications:

- `pkg/checker` - Core version checking logic
- `pkg/client` - GitHub API client
- `pkg/policy` - Policy implementations
- `pkg/types` - Shared data types

#### Internal Implementation (`internal/`)

These packages are for internal use only:

- `internal/version` - Legacy adapter (backward compatibility)
- `internal/github` - Legacy client adapter
- `internal/config` - Repository configuration management
- `internal/data` - Embedded cache loading
- `internal/cache` - Cache manager
- `internal/policy` - Policy factory from config

## Building

### Development Build

```bash
make build
```

Creates `bin/github-release-version-checker` for your current platform.

### Build for All Platforms

```bash
make build-all
```

Creates binaries for:

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Build with Version Information

```bash
make build VERSION=1.2.3
```

Embeds version information in the binary.

### Manual Build

```bash
go build -o bin/github-release-version-checker .
```

### Build Optimization

The Makefile uses these optimizations:

```bash
CGO_ENABLED=0 go build \
 -trimpath \
 -ldflags "-w -s -X main.Version=$VERSION" \
 -o bin/github-release-version-checker
```

- `CGO_ENABLED=0` - Static compilation without C dependencies
- `-trimpath` - Remove absolute file paths for reproducible builds
- `-w -s` - Strip debug information and symbol table
- `-X main.Version=$VERSION` - Embed version information

### Docker Build

```bash
make docker-build
```

Or manually:

```bash
docker build -t github-release-version-checker:latest .
```

## Testing

### Run All Tests

```bash
make test
```

Runs tests with race detection and coverage reporting.

### Run Specific Package Tests

```bash
go test -v ./pkg/checker
go test -v ./pkg/client
go test -v ./pkg/policy
```

### Run Specific Test

```bash
# Note: Using British spelling as per project convention
go test -v ./pkg/checker -run TestAnalyse_ExpiredVersion
```

### Test with Coverage

```bash
make test-coverage
```

Opens HTML coverage report in your browser.

### Coverage by Package

As of the latest build:

- `pkg/checker`: 57.3%
- `pkg/client`: 45.2%
- `pkg/policy`: 94.1%
- `internal/version`: 63.8%
- `internal/config`: 92.0%
- `internal/data`: 80.0%
- `internal/cache`: 92.9%

### Run Benchmarks

```bash
go test -bench=. -benchmem ./pkg/checker/
```

### Test Example Output

```text
=== RUN TestAnalyse_ExpiredVersion
--- PASS: TestAnalyse_ExpiredVersion (0.01s)
=== RUN TestAnalyse_CriticalVersion
--- PASS: TestAnalyse_CriticalVersion (0.00s)
=== RUN TestAnalyse_CurrentVersion
--- PASS: TestAnalyse_CurrentVersion (0.00s)
PASS
coverage: 57.3% of statements
ok github.com/nickromney-org/github-release-version-checker/pkg/checker 1.504s
```

## Code Style

### British English Convention

This project uses British English spelling throughout:

- `Analyse` (not Analyze)
- `colour` (not color)
- `Initialise` (not Initialize)

Linting enforces British spelling via `golangci-lint` with UK locale:

```yaml
# .golangci.yml
linters-settings:
 misspell:
 locale: UK
```

### Format Code

```bash
make fmt
```

Runs `gofmt` and `goimports` on all Go files.

### Run Linter

```bash
make lint
```

Runs `golangci-lint` with project configuration. If `golangci-lint` is not installed, it will be installed automatically.

### Linter Configuration

The project uses `.golangci.yml` with:

- `gofmt` - Standard formatting
- `govet` - Static analysis
- `staticcheck` - Advanced static analysis
- `misspell` (UK locale) - Spelling checks
- `ineffassign` - Detect ineffectual assignments
- `unused` - Find unused code

### Code Comments

Use British English in comments:

```go
// Analyse checks if a version complies with the policy
func Analyse(ctx context.Context, version string) (*Analysis, error) {
 // ...
}
```

### Test Naming

Tests use British spelling:

```go
func TestAnalyse_ExpiredVersion(t *testing.T) {
 // ...
}
```

## Release Data Caching

### Overview

The tool embeds historical release data to minimize GitHub API calls:

- **Embedded cache**: All releases from v2.159.0 to build time
- **API calls**: Always fetches 5 most recent releases (1 API call)
- **Validation**: Checks if embedded cache is current (within top 5)
- **Fallback**: Full API query if cache is more than 5 releases behind

**Result**: Reduces API calls from 2 per invocation to 1 in the common case.

### Cache Architecture

1. **Bootstrap Process**: `scripts/update-releases.sh` fetches all releases from GitHub API
1. **Embedded Data**: `internal/data/releases.json` is embedded in binary via `go:embed`
1. **Runtime Logic**:
   - Load embedded releases (instant, no API call)
   - Fetch 5 most recent from API (1 API call)
   - If latest embedded is in top 5: merge datasets (optimal path)
   - If cache is stale: fall back to full API query (2 API calls total)

### Maintaining the Cache

#### Automated Updates

GitHub Actions workflow (`.github/workflows/update-releases.yml`):

- Runs daily at 6 AM UTC (3 hours before runner compliance checks)
- Executes `check-releases` to validate cache
- If stale: runs `update-releases.sh` and commits changes
- Commit triggers semantic-release for new binary build
- Includes `[skip ci]` to avoid workflow recursion

#### Manual Update

```bash
# Update releases.json with latest data
./scripts/update-releases.sh

# Rebuild binary with new embedded cache
make build
```

#### Check Cache Status

```bash
go run cmd/check-releases/main.go
```

Exit codes:

- `0` - Cache is current
- `1` - Cache is stale (needs update)

#### Bootstrap Cache

```bash
go run cmd/bootstrap-releases/main.go
```

Fetches all releases from GitHub API and updates `internal/data/releases.json`.

## Contributing

### Workflow

1. **Fork the repository**

 ```bash
 gh repo fork nickromney-org/github-release-version-checker
 ```

1. **Create a feature branch**

 ```bash
 git checkout -b feature/amazing-feature
 ```

1. **Make your changes**

   - Write code following the [Code Style](#code-style) guidelines
   - Use British English spelling
   - Add tests for new functionality

1. **Run tests**

 ```bash
 make test
 ```

1. **Format and lint**

 ```bash
 make fmt
 make lint
 ```

1. **Commit your changes**

   Use [semantic commit messages](https://www.conventionalcommits.org/):

   ```bash
   git commit -m "feat: add support for custom policies"
   git commit -m "fix: correct version comparison for pre-release tags"
   git commit -m "docs: update installation instructions"
   ```

   Commit types:

   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `test:` - Test additions or updates
   - `refactor:` - Code refactoring
   - `chore:` - Build process or auxiliary tool changes

1. **Push to your fork**

 ```bash
 git push origin feature/amazing-feature
 ```

1. **Open a Pull Request**

 ```bash
 gh pr create --title "feat: add support for custom policies" --body "Description of changes"
 ```

### Pull Request Guidelines

- Provide a clear description of the changes
- Reference any related issues
- Ensure all tests pass
- Update documentation if needed
- Follow the project's code style
- Keep changes focused and atomic

### Adding New Repository Configurations

To add a new predefined repository configuration:

1. **Add config in `internal/config/repository.go`**:

 ```go
 ConfigMyRepo = RepositoryConfig{
 Owner: "owner",
 Repo: "repo",
 PolicyType: PolicyTypeVersions,
 MaxVersionsBehind: 3,
 CachePath: "data/myrepo.json",
 CacheEnabled: false,
 }
 ```

1. **Add aliases in `GetPredefinedConfig()`**:

 ```go
 "myrepo": ConfigMyRepo,
 "mr": ConfigMyRepo,
 ```

1. **Add tests in `internal/config/repository_test.go`**:

 ```go
 func TestGetPredefinedConfig_MyRepo(t *testing.T) {
 // ...
 }
 ```

1. **Update documentation** in `docs/CLI-USAGE.md`

### Adding New Policy Types

To add a new policy type:

1. **Implement the `VersionPolicy` interface in `pkg/policy/policy.go`**:

 ```go
 type MyPolicy struct {
 // fields
 }

 func (p *MyPolicy) Evaluate(...) PolicyResult {
 // implementation
 }
 ```

1. **Add constructor**:

 ```go
 func NewMyPolicy(param int) *MyPolicy {
 return &MyPolicy{...}
 }
 ```

1. **Add tests in `pkg/policy/policy_test.go`**

1. **Update documentation** in `docs/LIBRARY-USAGE.md`

## Why Go?

This tool is written in Go for several compelling reasons:

1. **Single Binary** - No runtime dependencies, ship one file
1. **Fast Startup** - ~10ms vs Node's ~500ms vs Python's ~200ms
1. **Cross-Compilation** - Build for any OS/arch from anywhere
1. **Strong Typing** - Catch bugs at compile time
1. **Excellent Standard Library** - HTTP, JSON, time handling built-in
1. **Perfect for CLI Tools** - This is Go's sweet spot (kubectl, helm, gh)
1. **Easy Distribution** - Just download and run
1. **Low Memory** - ~3-5MB resident memory
1. **Great Ecosystem** - Cobra, semver, go-github libraries

### Performance Comparison

| Implementation | Startup Time | Binary Size | Memory Usage |
| ----------------- | ------------ | ----------- | ------------ |
| **Go** | ~10ms | 8MB | 5MB |
| Bash + jq | ~5ms | N/A | 2MB |
| TypeScript (Node) | ~500ms | N/A | 40MB |
| Python | ~200ms | N/A | 20MB |

## Dependencies

The project uses minimal external dependencies:

- **[spf13/cobra](https://github.com/spf13/cobra)** - CLI framework
- **[Masterminds/semver/v3](https://github.com/Masterminds/semver)** - Semantic version parsing
- **[google/go-github/v57](https://github.com/google/go-github)** - GitHub API client
- **[fatih/color](https://github.com/fatih/color)** - Terminal colourisation
- **[golang.org/x/oauth2](https://golang.org/x/oauth2)** - GitHub authentication

## License

MIT License - see [LICENSE](../LICENSE) file for details.

## Resources

- **Repository**: https://github.com/nickromney-org/github-release-version-checker
- **Issues**: https://github.com/nickromney-org/github-release-version-checker/issues
- **Discussions**: https://github.com/nickromney-org/github-release-version-checker/discussions
- **API Documentation**: https://pkg.go.dev/github.com/nickromney-org/github-release-version-checker

## Getting Help

- Check the [documentation](../docs/)
- Search [existing issues](https://github.com/nickromney-org/github-release-version-checker/issues)
- Ask in [discussions](https://github.com/nickromney-org/github-release-version-checker/discussions)
- Open a [new issue](https://github.com/nickromney-org/github-release-version-checker/issues/new)
