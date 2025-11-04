# Generic Version Checker Implementation Plan

**Document Version**: 1.0
**Date**: 2025-11-03
**Author**: Implementation plan for expanding github-release-version-checker into a generic GitHub release version checker

## Executive Summary

This document outlines the implementation plan to transform the current GitHub Actions runner-specific version checker into a **generic GitHub release version checker** that supports multiple projects with different versioning policies, while maintaining backward compatibility and the current tool name.

### Key Goals

1. Support multiple GitHub projects (actions/runner, kubernetes/kubernetes, pulumi/pulumi, ubuntu releases, etc.)
1. Support both **time-based** expiry policies (days) and **version-based** policies (N minor versions)
1. Maintain embedded caches for common projects
1. Allow custom cache files for other projects
1. Default to current behavior (actions/runner with 30-day policy) for backward compatibility
1. Keep the current tool name: `github-release-version-checker`

## Current Architecture Limitations

### Hardcoded Dependencies

**Location**: `internal/github/client.go:14-17`

```go
const (
 owner = "actions"
 repo = "runner"
)
```

**Impact**: Cannot check other repositories

### Single Embedded Cache

**Location**: `data/releases.json`
**Content**: Only actions/runner releases
**Impact**: Must hit GitHub API for all other projects

### Time-Based Policy Only

**Location**: `internal/version/checker.go`
**Current**: Only supports days-based thresholds (CriticalAgeDays, MaxAgeDays)
**Impact**: Cannot express "3 minor versions behind" policy

### Hardcoded Workflows

**Location**: `.github/workflows/update-releases.yml`
**Impact**: Only updates actions/runner cache

## Proposed Architecture

### 1. Repository Configuration System

#### New Type: RepositoryConfig

**Location**: `internal/config/repository.go` (new package)

```go
package config

import "github.com/Masterminds/semver/v3"

// PolicyType defines the type of expiry policy
type PolicyType string

const (
 PolicyTypeDays PolicyType = "days" // Time-based: expires after N days
 PolicyTypeVersions PolicyType = "versions" // Version-based: expires after N minor versions
)

// RepositoryConfig defines a GitHub repository and its version policy
type RepositoryConfig struct {
 Owner string // GitHub owner (e.g., "actions", "kubernetes")
 Repo string // GitHub repo (e.g., "runner", "kubernetes")

 // Policy configuration
 PolicyType PolicyType
 CriticalDays int // For PolicyTypeDays
 MaxDays int // For PolicyTypeDays
 MaxVersionsBehind int // For PolicyTypeVersions

 // Cache configuration
 CachePath string // Path to embedded cache file
 CacheEnabled bool // Whether to use embedded cache
}

// Predefined repository configurations
var (
 ConfigActionsRunner = RepositoryConfig{
 Owner: "actions",
 Repo: "runner",
 PolicyType: PolicyTypeDays,
 CriticalDays: 12,
 MaxDays: 30,
 CachePath: "data/actions-runner.json",
 CacheEnabled: true,
 }

 ConfigKubernetes = RepositoryConfig{
 Owner: "kubernetes",
 Repo: "kubernetes",
 PolicyType: PolicyTypeVersions,
 MaxVersionsBehind: 3, // Support last 3 minor versions
 CachePath: "data/kubernetes.json",
 CacheEnabled: true,
 }

 ConfigPulumi = RepositoryConfig{
 Owner: "pulumi",
 Repo: "pulumi",
 PolicyType: PolicyTypeVersions,
 MaxVersionsBehind: 3,
 CachePath: "data/pulumi.json",
 CacheEnabled: true,
 }

 ConfigUbuntu = RepositoryConfig{
 Owner: "ubuntu",
 Repo: "ubuntu", // May need special handling
 PolicyType: PolicyTypeDays,
 CriticalDays: 180, // 6 months
 MaxDays: 365, // 1 year
 CachePath: "data/ubuntu.json",
 CacheEnabled: true,
 }
)

// GetPredefinedConfig returns a predefined config by name
func GetPredefinedConfig(name string) (*RepositoryConfig, error) {
 configs := map[string]RepositoryConfig{
 "actions-runner": ConfigActionsRunner,
 "github-runner": ConfigActionsRunner, // Alias
 "runner": ConfigActionsRunner, // Alias
 "kubernetes": ConfigKubernetes,
 "k8s": ConfigKubernetes, // Alias
 "pulumi": ConfigPulumi,
 "ubuntu": ConfigUbuntu,
 }

 config, ok := configs[name]
 if !ok {
 return nil, fmt.Errorf("unknown repository: %s", name)
 }
 return &config, nil
}

// ParseRepositoryString parses "owner/repo" format
func ParseRepositoryString(repoStr string) (*RepositoryConfig, error) {
 parts := strings.Split(repoStr, "/")
 if len(parts) != 2 {
 return nil, fmt.Errorf("invalid repository format: %s (expected: owner/repo)", repoStr)
 }

 // Default to version-based policy with conservative defaults
 return &RepositoryConfig{
 Owner: parts[0],
 Repo: parts[1],
 PolicyType: PolicyTypeVersions,
 MaxVersionsBehind: 3,
 CacheEnabled: false, // No embedded cache for custom repos
 }, nil
}
```

### 2. Version Policy System

#### New Interface: VersionPolicy

**Location**: `internal/policy/policy.go` (new package)

```go
package policy

import (
 "time"
 "github.com/Masterminds/semver/v3"
 "github.com/nickromney-org/github-release-version-checker/internal/version"
)

// PolicyResult contains the result of a policy evaluation
type PolicyResult struct {
 IsExpired bool
 IsCritical bool
 IsWarning bool
 Message string
 DaysOld int // For days-based policies
 VersionsBehind int // For version-based policies
}

// VersionPolicy defines the interface for version expiry policies
type VersionPolicy interface {
 // Evaluate checks if a version is expired/critical
 Evaluate(
 comparison *semver.Version,
 comparisonDate time.Time,
 latest *semver.Version,
 latestDate time.Time,
 newerReleases []version.Release,
 ) PolicyResult

 // Type returns the policy type
 Type() string
}

// DaysPolicy implements time-based expiry
type DaysPolicy struct {
 CriticalDays int
 MaxDays int
}

func (p *DaysPolicy) Evaluate(
 comparison *semver.Version,
 comparisonDate time.Time,
 latest *semver.Version,
 latestDate time.Time,
 newerReleases []version.Release,
) PolicyResult {
 if len(newerReleases) == 0 {
 return PolicyResult{IsExpired: false, IsCritical: false}
 }

 // Calculate days since FIRST newer release
 firstNewer := newerReleases[0]
 daysSinceFirstNewer := int(time.Since(firstNewer.PublishedAt).Hours() / 24)

 isExpired := daysSinceFirstNewer >= p.MaxDays
 isCritical := daysSinceFirstNewer >= p.CriticalDays && !isExpired
 isWarning := daysSinceFirstNewer > 0 && !isCritical && !isExpired

 return PolicyResult{
 IsExpired: isExpired,
 IsCritical: isCritical,
 IsWarning: isWarning,
 DaysOld: daysSinceFirstNewer,
 VersionsBehind: len(newerReleases),
 Message: fmt.Sprintf("%d days old, %d versions behind", daysSinceFirstNewer, len(newerReleases)),
 }
}

func (p *DaysPolicy) Type() string { return "days" }

// VersionsPolicy implements version-based expiry
type VersionsPolicy struct {
 MaxMinorVersionsBehind int
}

func (p *VersionsPolicy) Evaluate(
 comparison *semver.Version,
 comparisonDate time.Time,
 latest *semver.Version,
 latestDate time.Time,
 newerReleases []version.Release,
) PolicyResult {
 if comparison.Equal(latest) {
 return PolicyResult{IsExpired: false, IsCritical: false}
 }

 // Count how many minor versions behind
 minorVersionsBehind := 0
 currentMinor := comparison.Minor()
 seenMinors := make(map[uint64]bool)

 for _, rel := range newerReleases {
 // Only count distinct minor versions
 if rel.Version.Major() == comparison.Major() && rel.Version.Minor() > currentMinor {
 if !seenMinors[rel.Version.Minor()] {
 minorVersionsBehind++
 seenMinors[rel.Version.Minor()] = true
 }
 }

 // Stop if major version changed
 if rel.Version.Major() > comparison.Major() {
 break
 }
 }

 isExpired := minorVersionsBehind > p.MaxMinorVersionsBehind
 isCritical := minorVersionsBehind == p.MaxMinorVersionsBehind
 isWarning := minorVersionsBehind > 0 && minorVersionsBehind < p.MaxMinorVersionsBehind

 return PolicyResult{
 IsExpired: isExpired,
 IsCritical: isCritical,
 IsWarning: isWarning,
 VersionsBehind: minorVersionsBehind,
 Message: fmt.Sprintf("%d minor versions behind", minorVersionsBehind),
 }
}

func (p *VersionsPolicy) Type() string { return "versions" }

// NewPolicy creates a policy from config
func NewPolicy(config *config.RepositoryConfig) VersionPolicy {
 switch config.PolicyType {
 case config.PolicyTypeDays:
 return &DaysPolicy{
 CriticalDays: config.CriticalDays,
 MaxDays: config.MaxDays,
 }
 case config.PolicyTypeVersions:
 return &VersionsPolicy{
 MaxMinorVersionsBehind: config.MaxVersionsBehind,
 }
 default:
 // Default to days-based
 return &DaysPolicy{CriticalDays: 12, MaxDays: 30}
 }
}
```

### 3. Multi-Cache System

#### Cache Manager

**Location**: `internal/cache/manager.go` (new package)

```go
package cache

import (
 "embed"
 "encoding/json"
 "fmt"
 "os"
 "github.com/nickromney-org/github-release-version-checker/internal/version"
)

//go:embed data/*.json
var embeddedCaches embed.FS

// CacheManager handles multiple embedded and custom caches
type CacheManager struct {
 customCachePath string // Optional custom cache file
}

func NewCacheManager(customPath string) *CacheManager {
 return &CacheManager{customCachePath: customPath}
}

// LoadCache loads releases for a repository
func (m *CacheManager) LoadCache(repoConfig *config.RepositoryConfig) ([]version.Release, error) {
 // Priority: custom cache > embedded cache > no cache

 if m.customCachePath != "" {
 return m.loadCustomCache(m.customCachePath)
 }

 if repoConfig.CacheEnabled && repoConfig.CachePath != "" {
 return m.loadEmbeddedCache(repoConfig.CachePath)
 }

 return nil, nil // No cache available
}

func (m *CacheManager) loadEmbeddedCache(path string) ([]version.Release, error) {
 data, err := embeddedCaches.ReadFile(path)
 if err != nil {
 return nil, fmt.Errorf("failed to read embedded cache %s: %w", path, err)
 }

 var cacheData struct {
 GeneratedAt time.Time `json:"generated_at"`
 Repository string `json:"repository"`
 Releases []version.Release `json:"releases"`
 }

 if err := json.Unmarshal(data, &cacheData); err != nil {
 return nil, fmt.Errorf("failed to parse cache: %w", err)
 }

 return cacheData.Releases, nil
}

func (m *CacheManager) loadCustomCache(path string) ([]version.Release, error) {
 data, err := os.ReadFile(path)
 if err != nil {
 return nil, fmt.Errorf("failed to read custom cache %s: %w", path, err)
 }

 // Support same format as embedded caches
 var cacheData struct {
 Releases []version.Release `json:"releases"`
 }

 if err := json.Unmarshal(data, &cacheData); err != nil {
 return nil, fmt.Errorf("failed to parse custom cache: %w", err)
 }

 return cacheData.Releases, nil
}
```

### 4. Updated CLI Interface

#### New Flags

**Location**: `cmd/root.go`

```go
var (
 // Existing flags
 comparisonVersion string
 criticalAgeDays int
 maxAgeDays int
 verbose bool
 jsonOutput bool
 ciOutput bool
 quiet bool
 githubToken string
 showVersion bool
 noCache bool

 // NEW FLAGS
 repository string // Repository to check (default: "actions/runner")
 customCachePath string // Path to custom cache file
 policyType string // "days" or "versions"
 maxVersionsBehind int // For version-based policy
)

func init() {
 // Existing flags...

 // Repository selection
 rootCmd.Flags().StringVarP(&repository, "repo", "r", "actions/runner",
 "repository to check (name, owner/repo, or URL). Examples: kubernetes, pulumi/pulumi, https://github.com/ubuntu/ubuntu")

 // Custom cache
 rootCmd.Flags().StringVar(&customCachePath, "cache", "",
 "path to custom cache JSON file (overrides embedded cache)")

 // Policy override
 rootCmd.Flags().StringVar(&policyType, "policy", "",
 "policy type: 'days' or 'versions' (overrides repository default)")
 rootCmd.Flags().IntVar(&maxVersionsBehind, "max-versions", 0,
 "max minor versions behind before expiry (for version-based policy)")
}
```

#### Usage Examples

```bash
# Default: actions/runner with 30-day policy (backward compatible)
github-release-version-checker -c 2.327.1

# Check Kubernetes version (uses predefined config)
github-release-version-checker --repo kubernetes -c v1.32.0

# Check Pulumi with custom repository
github-release-version-checker --repo pulumi/pulumi -c v3.200.0

# Use custom cache file
github-release-version-checker --repo myorg/mytool -c v1.2.3 --cache ./my-cache.json

# Override policy type
github-release-version-checker --repo actions/runner -c 2.327.1 --policy versions --max-versions 5

# Check from GitHub URL
github-release-version-checker --repo https://github.com/kubernetes/kubernetes -c v1.32.0
```

### 5. Directory Structure Changes

```
github-release-version-checker/
├── cmd/
│ └── root.go # Updated with new flags
├── internal/
│ ├── config/ # NEW PACKAGE
│ │ ├── repository.go # Repository configurations
│ │ └── repository_test.go
│ ├── policy/ # NEW PACKAGE
│ │ ├── policy.go # Policy interface
│ │ ├── days.go # Days-based policy
│ │ ├── versions.go # Version-based policy
│ │ └── policy_test.go
│ ├── cache/ # NEW PACKAGE
│ │ ├── manager.go # Multi-cache management
│ │ └── manager_test.go
│ ├── github/
│ │ └── client.go # MODIFIED: Remove hardcoded owner/repo
│ ├── version/
│ │ ├── checker.go # MODIFIED: Use policy system
│ │ └── types.go
│ └── data/
│ └── loader.go # MODIFIED: Load from cache manager
├── data/ # EXPANDED
│ ├── actions-runner.json # Renamed from releases.json
│ ├── kubernetes.json # NEW
│ ├── pulumi.json # NEW
│ └── ubuntu.json # NEW
├── scripts/
│ ├── update-releases.sh # MODIFIED: Support multiple repos
│ └── bootstrap-cache.sh # NEW: Bootstrap new cache
├── cmd/bootstrap-releases/
│ └── main.go # MODIFIED: Accept --repo flag
└── cmd/check-releases/
 └── main.go # MODIFIED: Accept --repo flag
```

## Implementation Phases

### Phase 1: Core Architecture Refactoring

**Estimated Time**: 2-3 days
**PR Title**: `feat: add multi-repository support infrastructure`

**Tasks**:

1. Create `internal/config` package with `RepositoryConfig`
1. Create `internal/policy` package with policy interface
1. Create `internal/cache` package with cache manager
1. Modify `internal/github/client.go` to accept owner/repo parameters
1. Update `internal/version/checker.go` to use policy system
1. Add tests for all new packages

**Deliverables**:

- New packages with 80%+ test coverage
- GitHub client accepts dynamic repositories
- Policy system handles both days and versions
- Cache manager loads from multiple sources

### Phase 2: CLI Integration

**Estimated Time**: 1-2 days
**PR Title**: `feat: add CLI flags for repository and cache selection`

**Tasks**:

1. Add `--repo`, `--cache`, `--policy`, `--max-versions` flags to CLI
1. Implement repository name resolution (aliases)
1. Implement URL parsing (extract owner/repo from GitHub URLs)
1. Maintain backward compatibility (default to actions/runner)
1. Update help text and examples

**Deliverables**:

- New CLI flags working
- Backward compatibility verified
- Help documentation updated

### Phase 3: Embedded Caches

**Estimated Time**: 2-3 days
**PR Title**: `feat: add embedded caches for kubernetes, pulumi, ubuntu`

**Tasks**:

1. Fetch initial release data for kubernetes, pulumi, ubuntu
1. Create cache JSON files in `data/` directory
1. Update `cmd/bootstrap-releases` to accept `--repo` flag
1. Update `cmd/check-releases` to accept `--repo` flag
1. Create `scripts/bootstrap-cache.sh` for new repositories
1. Update `.github/workflows/update-releases.yml` to handle multiple repos

**Deliverables**:

- 4 embedded cache files (actions-runner, kubernetes, pulumi, ubuntu)
- Bootstrap scripts support multiple repositories
- CI/CD workflow updates all caches

### Phase 4: Testing & Documentation

**Estimated Time**: 1-2 days
**PR Title**: `docs: update documentation for generic version checker`

**Tasks**:

1. Integration tests for all predefined repositories
1. End-to-end tests for custom cache files
1. Update README.md with new capabilities
1. Update CLAUDE.md with architecture changes
1. Add migration guide for existing users
1. Add examples for each supported repository

**Deliverables**:

- Comprehensive test suite
- Updated documentation
- Migration guide

## Breaking Changes

### Minimal Breaking Changes

This design **avoids breaking changes** through:

1. **Default behavior unchanged**: No flags = actions/runner with 30-day policy
1. **Existing flags preserved**: All current flags continue to work
1. **Binary name unchanged**: Still `github-release-version-checker`
1. **Cache path migration**: `data/releases.json` → `data/actions-runner.json` (symlink for compatibility)

### Migration Strategy

For existing automation/scripts:

```bash
# Old command (still works):
github-release-version-checker -c 2.327.1

# New explicit form (equivalent):
github-release-version-checker --repo actions/runner -c 2.327.1
```

## Testing Strategy

### Unit Tests

- `internal/config`: Repository configuration parsing
- `internal/policy`: Both policy types with various scenarios
- `internal/cache`: Embedded and custom cache loading

### Integration Tests

```go
func TestMultiRepositorySupport(t *testing.T) {
 repos := []struct {
 name string
 repo string
 version string
 expect string
 }{
 {"actions-runner", "actions/runner", "2.327.1", "expired"},
 {"kubernetes", "kubernetes/kubernetes", "v1.29.0", "expired"}, // 4 minor behind
 {"pulumi", "pulumi/pulumi", "v3.200.0", "warning"},
 }

 for _, tt := range repos {
 t.Run(tt.name, func(t *testing.T) {
 // Test repository selection and policy evaluation
 })
 }
}
```

### End-to-End Tests

- Test with real GitHub API (limited in CI)
- Test custom cache files
- Test all predefined repositories
- Test backward compatibility

## Future Extensibility

### Custom Policy Plugins

Future enhancement: Allow users to define custom policy logic

```yaml
# .github-version-policy.yaml
repository: myorg/mytool
policy:
 type: custom
 script: ./policy.sh
```

### Additional Repository Metadata

Support for:

- LTS releases
- Security-only releases
- Custom release channels (stable, beta, alpha)

### Web Service Mode

Run as HTTP service for centralized version checking:

```bash
github-release-version-checker serve --port 8080
curl http://localhost:8080/check?repo=kubernetes/kubernetes&version=v1.32.0
```

## Risk Assessment

### Low Risk

- Backward compatibility maintained
- Existing functionality unchanged
- Progressive enhancement approach

### Medium Risk

- Cache file size growth (4 embedded caches instead of 1)
- **Mitigation**: Limit cache depth to last 100 releases per repo
- Version-based policy complexity
- **Mitigation**: Comprehensive tests for edge cases

### High Risk

- None identified

## Success Criteria

1. Support checking versions for 4+ different repositories
1. Both time-based and version-based policies working
1. Custom cache files supported
1. Zero breaking changes for existing users
1. Test coverage remains above 45%
1. Binary size remains under 12MB
1. Documentation complete and clear

## Appendices

### Appendix A: Cache File Format

```json
{
 "generated_at": "2025-11-03T12:00:00Z",
 "repository": "kubernetes/kubernetes",
 "policy_type": "versions",
 "releases": [
 {
 "version": "1.34.0",
 "published_at": "2025-10-15T00:00:00Z",
 "url": "https://github.com/kubernetes/kubernetes/releases/tag/v1.34.0"
 }
 ]
}
```

### Appendix B: Repository Aliases

```
actions/runner → actions-runner, github-runner, runner
kubernetes/kubernetes → kubernetes, k8s
pulumi/pulumi → pulumi
ubuntu → ubuntu
```

### Appendix C: Version Policy Comparison

| Repository | Policy Type | Threshold | Rationale |
|------------|-------------|-----------|-----------|
| actions/runner | days | 30 days | GitHub's official policy |
| kubernetes | versions | 3 minor | Kubernetes support policy |
| pulumi | versions | 3 minor | Similar to k8s |
| ubuntu | days | 365 days | 1 year LTS support cycle |

---

**Document Status**: Ready for Implementation
**Next Steps**: Review and approve before starting Phase 1
