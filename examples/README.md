# Examples

This directory contains example programs demonstrating how to use the `github-actions-runner-version` library in your own Go applications.

## Prerequisites

All examples require a GitHub token to avoid rate limiting:

```bash
export GITHUB_TOKEN="your_github_token_here"
```

## Running Examples

Each example is a standalone Go program. To run an example:

```bash
cd examples/basic
go run main.go
```

## Examples

### 1. Basic Usage (`basic/`)

Demonstrates the fundamental usage of the library:

- Creating a GitHub client
- Setting up a days-based policy
- Analysing a version
- Interpreting results

```bash
cd examples/basic
go run main.go
```

### 2. Version-Based Policy (`version-based-policy/`)

Shows how to use version-based policies (e.g., for Kubernetes):

- Creating a version-based policy (support N minor versions)
- Checking if a version is within the support window
- Handling version-based expiry

```bash
cd examples/version-based-policy
go run main.go
```

### 3. Custom Repository (`custom-repository/`)

Demonstrates checking any GitHub repository:

- Parameterized owner/repo/version
- Disabling cache for custom repos
- Displaying newer releases

```bash
cd examples/custom-repository
go run main.go hashicorp terraform 1.5.0
```

### 4. JSON Output (`json-output/`)

Shows how to use the built-in JSON marshalling:

- Encoding analysis results as JSON
- Error handling with JSON output
- Structured data for automation

```bash
cd examples/json-output
go run main.go
```

## Library Import

To use this library in your own project:

```bash
go get github.com/nickromney-org/github-release-version-checker
```

Then import the packages you need:

```go
import (
 "github.com/nickromney-org/github-release-version-checker/pkg/checker"
 "github.com/nickromney-org/github-release-version-checker/pkg/client"
 "github.com/nickromney-org/github-release-version-checker/pkg/policy"
)
```

## API Overview

### Client Package (`pkg/client`)

```go
// Create a GitHub client
ghClient := client.NewClient(token, owner, repo)

// Fetch releases
releases, err := ghClient.GetAllReleases(ctx)
recentReleases, err := ghClient.GetRecentReleases(ctx, 5)
latest, err := ghClient.GetLatestRelease(ctx)
```

### Policy Package (`pkg/policy`)

```go
// Days-based policy (e.g., GitHub Actions runners)
daysPolicy := policy.NewDaysPolicy(criticalDays, maxDays)

// Version-based policy (e.g., Kubernetes)
versionPolicy := policy.NewVersionsPolicy(maxMinorVersionsBehind)
```

### Checker Package (`pkg/checker`)

```go
// Create checker with policy
versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 NoCache: false,
}, pol)

// Analyse a version
analysis, err := versionChecker.Analyse(ctx, "2.328.0")

// Check status
switch analysis.Status() {
case checker.StatusCurrent:
 // Up to date
case checker.StatusWarning:
 // Behind but within policy
case checker.StatusCritical:
 // Approaching expiry
case checker.StatusExpired:
 // Beyond policy threshold
}
```

## See Also

- [Main README](../README.md) - Full project documentation
- [API Documentation](https://pkg.go.dev/github.com/nickromney-org/github-release-version-checker) - Go package documentation
