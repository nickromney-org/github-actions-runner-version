# Library Usage Guide

Use the GitHub Release Version Checker as a library in your own Go applications. The public API is exposed via the `/pkg` directory.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [API Packages](#api-packages)
- [Examples](#examples)
- [API Reference](#api-reference)

## Installation

```bash
go get github.com/nickromney-org/github-release-version-checker
```

## Quick Start

```go
package main

import (
 "context"
 "fmt"
 "os"

 "github.com/nickromney-org/github-release-version-checker/pkg/checker"
 "github.com/nickromney-org/github-release-version-checker/pkg/client"
 "github.com/nickromney-org/github-release-version-checker/pkg/policy"
)

func main() {
 // Create GitHub client
 ghClient := client.NewClient(os.Getenv("GITHUB_TOKEN"), "actions", "runner")

 // Create days-based policy (12 days critical, 30 days expired)
 pol := policy.NewDaysPolicy(12, 30)

 // Create checker with policy
 versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 NoCache: false,
 }, pol)

 // Analyse a version
 analysis, err := versionChecker.Analyse(context.Background(), "2.328.0")
 if err != nil {
 panic(err)
 }

 // Check results
 fmt.Printf("Status: %s\n", analysis.Status())
 fmt.Printf("Releases behind: %d\n", analysis.ReleasesBehind)
 fmt.Printf("Is expired: %v\n", analysis.IsExpired)
}
```

## API Packages

### `pkg/client` - GitHub API Client

Create clients to fetch releases from any GitHub repository:

```go
import "github.com/nickromney-org/github-release-version-checker/pkg/client"

// Create a client
ghClient := client.NewClient(token, "owner", "repo")

// Fetch releases
releases, err := ghClient.GetAllReleases(ctx)
latest, err := ghClient.GetLatestRelease(ctx)
recent, err := ghClient.GetRecentReleases(ctx, 5)
```

#### Functions

**`NewClient(token, owner, repo string) GitHubClient`**

Creates a new GitHub API client. Token is optional but recommended to avoid rate limiting.

**`GetLatestRelease(ctx context.Context) (types.Release, error)`**

Fetches the latest release.

**`GetAllReleases(ctx context.Context) ([]types.Release, error)`**

Fetches all releases (paginated).

**`GetRecentReleases(ctx context.Context, limit int) ([]types.Release, error)`**

Fetches the N most recent releases.

### `pkg/policy` - Expiry Policies

Two policy types are available:

#### Days-Based Policy

For time-based expiry (e.g., GitHub Actions runners):

```go
import "github.com/nickromney-org/github-release-version-checker/pkg/policy"

// Warn after 12 days, expire after 30 days
daysPolicy := policy.NewDaysPolicy(12, 30)
```

**Use cases:**

- GitHub Actions runners (30-day policy)
- Time-sensitive dependencies
- Security-critical updates

#### Version-Based Policy

For semantic versioning-based expiry (e.g., Kubernetes):

```go
import "github.com/nickromney-org/github-release-version-checker/pkg/policy"

// Support up to 3 minor versions behind
versionPolicy := policy.NewVersionsPolicy(3)
```

**Use cases:**

- Kubernetes (N-3 minor version support)
- Node.js (N-3 major version support)
- Libraries following semantic versioning

#### Policy Interface

Both policies implement the `VersionPolicy` interface:

```go
type VersionPolicy interface {
 Evaluate(
 comparisonVersion *semver.Version,
 comparisonDate time.Time,
 latestVersion *semver.Version,
 latestDate time.Time,
 newerReleases []types.Release,
 ) PolicyResult

 Type() string
 GetCriticalAgeDays() int
 GetMaxAgeDays() int
 GetMaxVersionsBehind() int
}
```

### `pkg/checker` - Version Analysis

Analyse versions against policies:

```go
import "github.com/nickromney-org/github-release-version-checker/pkg/checker"

// Create checker with policy
versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 NoCache: false, // Use embedded cache
}, pol)

// Analyse a version
analysis, err := versionChecker.Analyse(ctx, "2.328.0")

// Access results
switch analysis.Status() {
case checker.StatusCurrent:
 fmt.Println(" Up to date")
case checker.StatusWarning:
 fmt.Println(" Update available")
case checker.StatusCritical:
 fmt.Println(" Update urgently")
case checker.StatusExpired:
 fmt.Println(" Expired - update immediately")
}
```

#### Config Options

```go
type Config struct {
 CriticalAgeDays int // Days before critical warning
 MaxAgeDays int // Days before version expires
 NoCache bool // Bypass embedded cache
}
```

#### Analysis Result

The `Analysis` struct provides comprehensive information:

```go
type Analysis struct {
 LatestVersion *semver.Version // Latest available version
 ComparisonVersion *semver.Version // Version being checked
 ComparisonReleasedAt *time.Time // When comparison version was released
 IsLatest bool // Is on latest version
 IsExpired bool // Beyond max age threshold
 IsCritical bool // Within critical age window
 ReleasesBehind int // Number of newer releases
 DaysSinceUpdate int // Days since first newer release
 FirstNewerVersion *semver.Version // First newer version available
 FirstNewerReleaseDate *time.Time // When first newer release was published
 NewerReleases []types.Release // All newer releases
 RecentReleases []ReleaseExpiry // Recent releases for timeline
 Message string // Human-readable status message
 CriticalAgeDays int // Critical threshold (days)
 MaxAgeDays int // Expiry threshold (days)
 PolicyType string // "days" or "versions"
 MinorVersionsBehind int // For version-based policies
}
```

#### Status Method

```go
func (a *Analysis) Status() string
```

Returns one of:

- `"current"` - On latest version
- `"warning"` - Behind but not critical
- `"critical"` - Within critical window
- `"expired"` - Beyond expiry threshold

### `pkg/types` - Shared Types

```go
import "github.com/nickromney-org/github-release-version-checker/pkg/types"

type Release struct {
 Version *semver.Version
 PublishedAt time.Time
 URL string
}
```

## Examples

### Example 1: Basic Usage

```go
package main

import (
 "context"
 "fmt"
 "os"

 "github.com/nickromney-org/github-release-version-checker/pkg/checker"
 "github.com/nickromney-org/github-release-version-checker/pkg/client"
 "github.com/nickromney-org/github-release-version-checker/pkg/policy"
)

func main() {
 // Create client
 ghClient := client.NewClient(os.Getenv("GITHUB_TOKEN"), "actions", "runner")

 // Create policy
 pol := policy.NewDaysPolicy(12, 30)

 // Create checker
 versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 }, pol)

 // Analyse
 analysis, err := versionChecker.Analyse(context.Background(), "2.328.0")
 if err != nil {
 fmt.Printf("Error: %v\n", err)
 return
 }

 // Display results
 fmt.Printf("Latest: %s\n", analysis.LatestVersion.String())
 fmt.Printf("Your version: %s\n", analysis.ComparisonVersion.String())
 fmt.Printf("Status: %s\n", analysis.Status())
 fmt.Printf("Releases behind: %d\n", analysis.ReleasesBehind)
 fmt.Printf("Days since update: %d\n", analysis.DaysSinceUpdate)
}
```

### Example 2: Version-Based Policy (Kubernetes)

```go
package main

import (
 "context"
 "fmt"
 "os"

 "github.com/nickromney-org/github-release-version-checker/pkg/checker"
 "github.com/nickromney-org/github-release-version-checker/pkg/client"
 "github.com/nickromney-org/github-release-version-checker/pkg/policy"
)

func main() {
 // Create client for Kubernetes
 ghClient := client.NewClient(os.Getenv("GITHUB_TOKEN"), "kubernetes", "kubernetes")

 // Create version-based policy (3 minor versions)
 pol := policy.NewVersionsPolicy(3)

 // Create checker
 versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{}, pol)

 // Analyse
 analysis, err := versionChecker.Analyse(context.Background(), "1.28.0")
 if err != nil {
 fmt.Printf("Error: %v\n", err)
 return
 }

 // Display results
 fmt.Printf("Latest: %s\n", analysis.LatestVersion.String())
 fmt.Printf("Your version: %s\n", analysis.ComparisonVersion.String())
 fmt.Printf("Status: %s\n", analysis.Status())
 fmt.Printf("Minor versions behind: %d\n", analysis.MinorVersionsBehind)
}
```

### Example 3: Custom Repository

```go
package main

import (
 "context"
 "fmt"
 "os"

 "github.com/nickromney-org/github-release-version-checker/pkg/checker"
 "github.com/nickromney-org/github-release-version-checker/pkg/client"
 "github.com/nickromney-org/github-release-version-checker/pkg/policy"
)

func main() {
 // Create client for any repository
 ghClient := client.NewClient(os.Getenv("GITHUB_TOKEN"), "hashicorp", "terraform")

 // Create version-based policy
 pol := policy.NewVersionsPolicy(3)

 // Create checker
 versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{}, pol)

 // Analyse
 analysis, err := versionChecker.Analyse(context.Background(), "1.5.0")
 if err != nil {
 fmt.Printf("Error: %v\n", err)
 return
 }

 // Display results
 fmt.Printf("Repository: hashicorp/terraform\n")
 fmt.Printf("Latest: %s\n", analysis.LatestVersion.String())
 fmt.Printf("Your version: %s\n", analysis.ComparisonVersion.String())
 fmt.Printf("Status: %s\n", analysis.Status())
}
```

### Example 4: JSON Output

```go
package main

import (
 "context"
 "encoding/json"
 "fmt"
 "os"

 "github.com/nickromney-org/github-release-version-checker/pkg/checker"
 "github.com/nickromney-org/github-release-version-checker/pkg/client"
 "github.com/nickromney-org/github-release-version-checker/pkg/policy"
)

func main() {
 // Create client and checker
 ghClient := client.NewClient(os.Getenv("GITHUB_TOKEN"), "actions", "runner")
 pol := policy.NewDaysPolicy(12, 30)
 versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 }, pol)

 // Analyse
 analysis, err := versionChecker.Analyse(context.Background(), "2.328.0")
 if err != nil {
 fmt.Printf("Error: %v\n", err)
 return
 }

 // Marshal to JSON
 jsonData, err := json.MarshalIndent(analysis, "", " ")
 if err != nil {
 fmt.Printf("JSON error: %v\n", err)
 return
 }

 fmt.Println(string(jsonData))
}
```

### Example 5: Error Handling

```go
package main

import (
 "context"
 "errors"
 "fmt"
 "os"

 "github.com/nickromney-org/github-release-version-checker/pkg/checker"
 "github.com/nickromney-org/github-release-version-checker/pkg/client"
 "github.com/nickromney-org/github-release-version-checker/pkg/policy"
)

func main() {
 ghClient := client.NewClient(os.Getenv("GITHUB_TOKEN"), "actions", "runner")
 pol := policy.NewDaysPolicy(12, 30)
 versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 }, pol)

 analysis, err := versionChecker.Analyse(context.Background(), "invalid-version")
 if err != nil {
 // Handle specific error types
 if errors.Is(err, checker.ErrInvalidVersion) {
 fmt.Println("Invalid version format")
 } else if errors.Is(err, checker.ErrVersionNotFound) {
 fmt.Println("Version not found in releases")
 } else {
 fmt.Printf("Error: %v\n", err)
 }
 return
 }

 fmt.Printf("Status: %s\n", analysis.Status())
}
```

## Working Examples

See the [examples/](../examples/) directory for complete working examples:

- **`examples/basic/`** - Basic usage with days-based policy
- **`examples/version-based-policy/`** - Version-based policy (Kubernetes)
- **`examples/custom-repository/`** - Check any GitHub repository
- **`examples/json-output/`** - Using JSON marshalling

Run an example:

```bash
cd examples/basic
export GITHUB_TOKEN="your_token"
go run main.go
```

## API Reference

Complete API documentation is available at:

**[pkg.go.dev/github.com/nickromney-org/github-release-version-checker](https://pkg.go.dev/github.com/nickromney-org/github-release-version-checker)**

## Best Practices

### 1. Always Use Context

Pass a context for cancellation and timeout control:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

analysis, err := versionChecker.Analyse(ctx, "2.328.0")
```

### 2. Handle Errors Properly

Check for specific error types:

```go
analysis, err := versionChecker.Analyse(ctx, version)
if err != nil {
 if errors.Is(err, checker.ErrVersionNotFound) {
 // Handle version not found
 } else {
 // Handle other errors
 }
 return
}
```

### 3. Use GitHub Tokens

Avoid rate limiting by providing a token:

```go
token := os.Getenv("GITHUB_TOKEN")
ghClient := client.NewClient(token, "owner", "repo")
```

### 4. Cache Results

For repeated checks, consider caching results:

```go
type CachedChecker struct {
 checker checker.Checker
 cache map[string]*checker.Analysis
 mu sync.RWMutex
}
```

### 5. Choose the Right Policy

- **Days-based**: For time-sensitive updates (security patches, runner compliance)
- **Version-based**: For semantic versioning support windows (Kubernetes, Node.js)

## Next Steps

- [CLI Usage Guide](CLI-USAGE.md) - Use as a command-line tool
- [GitHub Actions Integration](GITHUB-ACTIONS.md) - Use in CI/CD
- [Development Guide](DEVELOPMENT.md) - Contribute to the project
