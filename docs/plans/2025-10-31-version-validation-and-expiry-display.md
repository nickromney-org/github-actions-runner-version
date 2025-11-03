# Version Validation and Expiry Display Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add version validation, expiry timeline table, and clearer status messages with British English throughout.

**Architecture:** Extend the Analysis struct with expiry timeline data, add validation in Checker.Analyse(), update output formatting in cmd/root.go.

**Tech Stack:** Go 1.21+, go-github/v57, Masterminds/semver, fatih/color (aliased to colour)

---

## Task 1: Add ReleaseExpiry Type and Update Analysis Struct

**Files:**

- Modify: `internal/version/types.go`

**Step 1: Write failing test for ReleaseExpiry JSON marshaling**

Add to `internal/version/types_test.go` (create new file):

```go
package version

import (
 "encoding/json"
 "testing"
 "time"

 "github.com/Masterminds/semver/v3"
)

func TestReleaseExpiry_JSONMarshaling(t *testing.T) {
 releaseDate := time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC)
 expiryDate := time.Date(2024, 8, 24, 0, 0, 0, 0, time.UTC)

 expiry := ReleaseExpiry{
 Version: semver.MustParse("2.327.1"),
 ReleasedAt: releaseDate,
 ExpiresAt: &expiryDate,
 DaysUntilExpiry: -77,
 IsExpired: true,
 IsLatest: false,
 }

 data, err := json.Marshal(expiry)
 if err != nil {
 t.Fatalf("failed to marshal: %v", err)
 }

 var result map[string]interface{}
 if err := json.Unmarshal(data, &result); err != nil {
 t.Fatalf("failed to unmarshal: %v", err)
 }

 if result["version"] != "2.327.1" {
 t.Errorf("expected version 2.327.1, got %v", result["version"])
 }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/version -v -run TestReleaseExpiry_JSONMarshaling`

Expected: FAIL with "undefined: ReleaseExpiry"

**Step 3: Add ReleaseExpiry struct to types.go**

Add after the Release struct in `internal/version/types.go`:

```go
// ReleaseExpiry represents expiry information for a single release
type ReleaseExpiry struct {
 Version *semver.Version `json:"version"`
 ReleasedAt time.Time `json:"released"`
 ExpiresAt *time.Time `json:"expires"`
 DaysUntilExpiry int `json:"days_until_expiry"`
 IsExpired bool `json:"is_expired"`
 IsLatest bool `json:"is_latest"`
}

// MarshalJSON implements custom JSON marshaling for ReleaseExpiry
func (r *ReleaseExpiry) MarshalJSON() ([]byte, error) {
 return json.Marshal(&struct {
 Version string `json:"version"`
 ReleasedAt string `json:"released"`
 ExpiresAt *string `json:"expires"`
 DaysUntilExpiry int `json:"days_until_expiry"`
 IsExpired bool `json:"is_expired"`
 IsLatest bool `json:"is_latest"`
 }{
 Version: r.Version.String(),
 ReleasedAt: r.ReleasedAt.Format(time.RFC3339),
 ExpiresAt: timeString(r.ExpiresAt),
 DaysUntilExpiry: r.DaysUntilExpiry,
 IsExpired: r.IsExpired,
 IsLatest: r.IsLatest,
 })
}
```

**Step 4: Add RecentReleases field to Analysis struct**

In `internal/version/types.go`, add to Analysis struct:

```go
type Analysis struct {
 LatestVersion *semver.Version `json:"latest_version"`
 ComparisonVersion *semver.Version `json:"comparison_version,omitempty"`
 ComparisonReleasedAt *time.Time `json:"comparison_released_at,omitempty"`
 IsLatest bool `json:"is_latest"`
 IsExpired bool `json:"is_expired"`
 IsCritical bool `json:"is_critical"`
 ReleasesBehind int `json:"releases_behind"`
 DaysSinceUpdate int `json:"days_since_update"`
 FirstNewerVersion *semver.Version `json:"first_newer_version,omitempty"`
 FirstNewerReleaseDate *time.Time `json:"first_newer_release_date,omitempty"`
 NewerReleases []Release `json:"newer_releases,omitempty"`
 RecentReleases []ReleaseExpiry `json:"recent_releases,omitempty"`
 Message string `json:"message"`

 // Configuration used
 CriticalAgeDays int `json:"critical_age_days"`
 MaxAgeDays int `json:"max_age_days"`
}
```

**Step 5: Update Analysis.MarshalJSON to include ComparisonReleasedAt**

In `internal/version/types.go`, update the MarshalJSON method:

```go
func (a *Analysis) MarshalJSON() ([]byte, error) {
 type Alias Analysis
 return json.MarshalIndent(&struct {
 LatestVersion string `json:"latest_version"`
 ComparisonVersion string `json:"comparison_version,omitempty"`
 ComparisonReleasedAt *string `json:"comparison_released_at,omitempty"`
 FirstNewerVersion string `json:"first_newer_version,omitempty"`
 FirstNewerReleaseDate *string `json:"first_newer_release_date,omitempty"`
 Status Status `json:"status"`
 *Alias
 }{
 LatestVersion: a.LatestVersion.String(),
 ComparisonVersion: versionString(a.ComparisonVersion),
 ComparisonReleasedAt: timeString(a.ComparisonReleasedAt),
 FirstNewerVersion: versionString(a.FirstNewerVersion),
 FirstNewerReleaseDate: timeString(a.FirstNewerReleaseDate),
 Status: a.Status(),
 Alias: (*Alias)(a),
 }, "", " ")
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/version -v -run TestReleaseExpiry_JSONMarshaling`

Expected: PASS

**Step 7: Commit**

```bash
git add internal/version/types.go internal/version/types_test.go
git commit -m "feat: add ReleaseExpiry type and extend Analysis struct

- Add ReleaseExpiry struct with JSON marshaling
- Add RecentReleases field to Analysis
- Add ComparisonReleasedAt field to track comparison version release date
- Test JSON marshaling for new types"
```

---

## Task 2: Add Version Validation

**Files:**

- Modify: `internal/version/checker.go`
- Modify: `internal/version/checker_test.go`

**Step 1: Write failing test for version validation**

Add to `internal/version/checker_test.go`:

```go
func TestAnalyse_NonExistentVersion(t *testing.T) {
 latest := newTestRelease("2.329.0", 3)
 older := newTestRelease("2.328.0", 20)

 client := &MockGitHubClient{
 LatestRelease: &latest,
 AllReleases: []Release{latest, older},
 }

 checker := NewChecker(client, CheckerConfig{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 })

 ctx := context.Background()
 _, err := checker.Analyse(ctx, "2.327.99")

 if err == nil {
 t.Fatal("expected error for non-existent version, got nil")
 }

 expectedMsg := "version 2.327.99 does not exist in GitHub releases"
 if !strings.Contains(err.Error(), expectedMsg) {
 t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
 }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/version -v -run TestAnalyse_NonExistentVersion`

Expected: FAIL with "expected error for non-existent version, got nil"

**Step 3: Add versionExists helper method**

Add to `internal/version/checker.go` after the NewChecker function:

```go
// versionExists checks if a version exists in the releases list
func (c *Checker) versionExists(releases []Release, version *semver.Version) bool {
 for _, release := range releases {
 if release.Version.Equal(version) {
 return true
 }
 }
 return false
}
```

**Step 4: Add validation in Analyse method**

In `internal/version/checker.go`, after fetching all releases and before finding newer releases:

```go
// Fetch all releases to find newer versions
allReleases, err := c.client.GetAllReleases(ctx)
if err != nil {
 return nil, fmt.Errorf("failed to fetch all releases: %w", err)
}

// NEW: Validate version exists
if !c.versionExists(allReleases, comparisonVersion) {
 return nil, fmt.Errorf("version %s does not exist in GitHub releases (latest: %s)",
 comparisonVersion, latestRelease.Version)
}

// Find releases newer than comparison version
newerReleases := c.findNewerReleases(allReleases, comparisonVersion)
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/version -v -run TestAnalyse_NonExistentVersion`

Expected: PASS

**Step 6: Commit**

```bash
git add internal/version/checker.go internal/version/checker_test.go
git commit -m "feat: validate version exists before processing

- Add versionExists helper method
- Return error for non-existent versions with helpful message
- Test non-existent version rejection"
```

---

## Task 3: Rename Analyze to Analyse (British English)

**Files:**

- Modify: `internal/version/checker.go`
- Modify: `internal/version/checker_test.go`
- Modify: `cmd/root.go`

**Step 1: Rename method in checker.go**

In `internal/version/checker.go`, change method name:

```go
// Analyse performs the version analysis
func (c *Checker) Analyse(ctx context.Context, comparisonVersionStr string) (*Analysis, error) {
```

**Step 2: Update cmd/root.go to use Analyse**

In `cmd/root.go`, find the call to `Analyze` and change to `Analyse`:

```go
// Run analysis
analysis, err := checker.Analyse(cmd.Context(), comparisonVersion)
if err != nil {
 return fmt.Errorf("analysis failed: %w", err)
}
```

**Step 3: Rename all test functions**

In `internal/version/checker_test.go`, rename all test functions:

- `TestAnalyze_LatestVersion` → `TestAnalyse_LatestVersion`
- `TestAnalyze_CurrentVersion` → `TestAnalyse_CurrentVersion`
- `TestAnalyze_ExpiredVersion` → `TestAnalyse_ExpiredVersion`
- `TestAnalyze_CriticalVersion` → `TestAnalyse_CriticalVersion`

Update all `checker.Analyze()` calls to `checker.Analyse()`.

**Step 4: Run tests to verify**

Run: `go test ./... -v`

Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/version/checker.go internal/version/checker_test.go cmd/root.go
git commit -m "refactor: rename Analyze to Analyse (British English)

- Rename Analyze method to Analyse
- Update all test function names
- Update cmd/root.go to use Analyse"
```

---

## Task 4: Add calculateRecentReleases Method

**Files:**

- Modify: `internal/version/checker.go`
- Modify: `internal/version/checker_test.go`

**Step 1: Write failing test for 90-day window**

Add to `internal/version/checker_test.go`:

```go
func TestCalculateRecentReleases_Last90Days(t *testing.T) {
 // Create releases spanning 120 days
 releases := []Release{
 newTestRelease("2.329.0", 5), // 5 days ago
 newTestRelease("2.328.0", 25), // 25 days ago
 newTestRelease("2.327.1", 50), // 50 days ago
 newTestRelease("2.327.0", 80), // 80 days ago
 newTestRelease("2.326.0", 100), // 100 days ago - outside window
 }

 comparisonVersion := semver.MustParse("2.327.0")
 latestVersion := semver.MustParse("2.329.0")

 checker := &Checker{}
 recent := checker.calculateRecentReleases(releases, comparisonVersion, latestVersion)

 // Should include releases from last 90 days: 2.329.0, 2.328.0, 2.327.1, 2.327.0
 if len(recent) != 4 {
 t.Errorf("expected 4 releases, got %d", len(recent))
 }
}

func TestCalculateRecentReleases_Minimum4(t *testing.T) {
 // Only 2 releases in last 90 days, but should return minimum 4
 releases := []Release{
 newTestRelease("2.329.0", 5), // 5 days ago
 newTestRelease("2.328.0", 25), // 25 days ago
 newTestRelease("2.327.0", 100), // 100 days ago
 newTestRelease("2.326.0", 120), // 120 days ago
 }

 comparisonVersion := semver.MustParse("2.327.0")
 latestVersion := semver.MustParse("2.329.0")

 checker := &Checker{}
 recent := checker.calculateRecentReleases(releases, comparisonVersion, latestVersion)

 if len(recent) != 4 {
 t.Errorf("expected minimum 4 releases, got %d", len(recent))
 }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/version -v -run TestCalculateRecentReleases`

Expected: FAIL with "undefined: Checker.calculateRecentReleases"

**Step 3: Implement calculateRecentReleases method**

Add to `internal/version/checker.go`:

```go
// calculateRecentReleases returns releases for the expiry timeline table
// Shows all releases from last 90 days, or minimum 4 releases
func (c *Checker) calculateRecentReleases(allReleases []Release, comparisonVersion *semver.Version, latestVersion *semver.Version) []ReleaseExpiry {
 now := time.Now()
 ninetyDaysAgo := now.AddDate(0, 0, -90)

 var recentReleases []Release

 // Collect releases from last 90 days
 for _, release := range allReleases {
 if release.PublishedAt.After(ninetyDaysAgo) {
 recentReleases = append(recentReleases, release)
 }
 }

 // Ensure minimum 4 releases
 if len(recentReleases) < 4 && len(allReleases) >= 4 {
 // Sort all releases by date (newest first)
 sorted := make([]Release, len(allReleases))
 copy(sorted, allReleases)
 for i := 0; i < len(sorted)-1; i++ {
 for j := i + 1; j < len(sorted); j++ {
 if sorted[i].PublishedAt.Before(sorted[j].PublishedAt) {
 sorted[i], sorted[j] = sorted[j], sorted[i]
 }
 }
 }
 // Take first 4
 recentReleases = sorted[:4]
 }

 // Sort oldest first (for display)
 for i := 0; i < len(recentReleases)-1; i++ {
 for j := i + 1; j < len(recentReleases); j++ {
 if recentReleases[i].PublishedAt.After(recentReleases[j].PublishedAt) {
 recentReleases[i], recentReleases[j] = recentReleases[j], recentReleases[i]
 }
 }
 }

 // Convert to ReleaseExpiry with expiry calculations
 var result []ReleaseExpiry
 for i, release := range recentReleases {
 expiry := ReleaseExpiry{
 Version: release.Version,
 ReleasedAt: release.PublishedAt,
 IsLatest: release.Version.Equal(latestVersion),
 }

 // Calculate expiry (30 days after next release)
 if i < len(recentReleases)-1 {
 nextRelease := recentReleases[i+1]
 expiryDate := nextRelease.PublishedAt.AddDate(0, 0, 30)
 expiry.ExpiresAt = &expiryDate
 expiry.DaysUntilExpiry = daysBetween(now, expiryDate)
 expiry.IsExpired = now.After(expiryDate)
 } else {
 // Latest version - no expiry
 expiry.ExpiresAt = nil
 expiry.DaysUntilExpiry = 0
 expiry.IsExpired = false
 }

 result = append(result, expiry)
 }

 return result
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/version -v -run TestCalculateRecentReleases`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/version/checker.go internal/version/checker_test.go
git commit -m "feat: add calculateRecentReleases method

- Shows all releases from last 90 days
- Minimum 4 releases even if older than 90 days
- Calculates expiry dates (30 days after next release)
- Latest version has no expiry
- Test 90-day window and minimum 4 logic"
```

---

## Task 5: Populate RecentReleases in Analyse Method

**Files:**

- Modify: `internal/version/checker.go`
- Modify: `internal/version/checker_test.go`

**Step 1: Write test for RecentReleases population**

Add to `internal/version/checker_test.go`:

```go
func TestAnalyse_PopulatesRecentReleases(t *testing.T) {
 releases := []Release{
 newTestRelease("2.329.0", 5),
 newTestRelease("2.328.0", 25),
 newTestRelease("2.327.1", 50),
 newTestRelease("2.327.0", 80),
 }

 client := &MockGitHubClient{
 LatestRelease: &releases[0],
 AllReleases: releases,
 }

 checker := NewChecker(client, CheckerConfig{
 CriticalAgeDays: 12,
 MaxAgeDays: 30,
 })

 ctx := context.Background()
 analysis, err := checker.Analyse(ctx, "2.327.1")

 if err != nil {
 t.Fatalf("Analyse failed: %v", err)
 }

 if len(analysis.RecentReleases) == 0 {
 t.Error("expected RecentReleases to be populated")
 }

 // Latest should have no expiry
 latest := analysis.RecentReleases[len(analysis.RecentReleases)-1]
 if latest.ExpiresAt != nil {
 t.Error("latest version should have no expiry date")
 }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/version -v -run TestAnalyse_PopulatesRecentReleases`

Expected: FAIL with "expected RecentReleases to be populated"

**Step 3: Call calculateRecentReleases in Analyse**

In `internal/version/checker.go`, add after building the analysis struct:

```go
// Build analysis
analysis := &Analysis{
 LatestVersion: latestRelease.Version,
 ComparisonVersion: comparisonVersion,
 IsLatest: false,
 ReleasesBehind: len(newerReleases),
 NewerReleases: newerReleases,
 CriticalAgeDays: c.config.CriticalAgeDays,
 MaxAgeDays: c.config.MaxAgeDays,
}

// NEW: Calculate recent releases for timeline table
analysis.RecentReleases = c.calculateRecentReleases(allReleases, comparisonVersion, latestRelease.Version)

// Find comparison version release date
for _, release := range allReleases {
 if release.Version.Equal(comparisonVersion) {
 analysis.ComparisonReleasedAt = &release.PublishedAt
 break
 }
}

// Calculate age from first newer release
if len(newerReleases) > 0 {
 // ... existing code
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/version -v -run TestAnalyse_PopulatesRecentReleases`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/version/checker.go internal/version/checker_test.go
git commit -m "feat: populate RecentReleases in Analyse

- Call calculateRecentReleases to build expiry timeline
- Find and store comparison version release date
- Test RecentReleases population"
```

---

## Task 6: Add Quiet Flag and Colour Alias

**Files:**

- Modify: `cmd/root.go`

**Step 1: Add quiet flag**

In `cmd/root.go`, add to the var block:

```go
var (
 comparisonVersion string
 criticalAgeDays int
 maxAgeDays int
 verbose bool
 jsonOutput bool
 ciOutput bool
 quiet bool // NEW
 githubToken string
```

**Step 2: Add flag to init function**

In `cmd/root.go`, add to the init() function:

```go
func init() {
 rootCmd.Flags().StringVarP(&comparisonVersion, "compare", "c", "", "version to compare against (e.g., 2.327.1)")
 rootCmd.Flags().IntVarP(&criticalAgeDays, "critical-days", "d", 12, "days before critical warning")
 rootCmd.Flags().IntVarP(&maxAgeDays, "max-days", "m", 30, "days before version expires")
 rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
 rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
 rootCmd.Flags().BoolVar(&ciOutput, "ci", false, "format output for CI/GitHub Actions")
 rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "quiet output (suppress expiry table)")
 rootCmd.Flags().StringVarP(&githubToken, "token", "t", os.Getenv("GITHUB_TOKEN"), "GitHub token (or GITHUB_TOKEN env var)")
}
```

**Step 3: Import color as colour**

In `cmd/root.go`, change the import:

```go
import (
 "fmt"
 "os"
 "time"

 colour "github.com/fatih/color"
 "github.com/spf13/cobra"
 "github.com/nickromney-org/github-release-version-checker/internal/github"
 "github.com/nickromney-org/github-release-version-checker/internal/version"
)
```

**Step 4: Rename color variables to use colour**

In `cmd/root.go`, rename all color variables:

```go
var (
 comparisonVersion string
 criticalAgeDays int
 maxAgeDays int
 verbose bool
 jsonOutput bool
 ciOutput bool
 quiet bool
 githubToken string

 // Colours for output
 green = colour.New(colour.FgGreen, colour.Bold)
 yellow = colour.New(colour.FgYellow, colour.Bold)
 red = colour.New(colour.FgRed, colour.Bold)
 cyan = colour.New(colour.FgCyan)
 gray = colour.New(colour.Faint)
)
```

**Step 5: Rename getStatusColor to getStatusColour**

In `cmd/root.go`, rename the function and update all references:

```go
func getStatusColour(status version.Status) *colour.Color {
 switch status {
 case version.StatusCurrent:
 return green
 case version.StatusWarning:
 return yellow
 case version.StatusCritical:
 return yellow
 case version.StatusExpired:
 return red
 default:
 return cyan
 }
}
```

Update the call in `printStatus`:

```go
func printStatus(analysis *version.Analysis) {
 status := analysis.Status()
 icon := getStatusIcon(status)
 colourFunc := getStatusColour(status)

 // Main status line
 colourFunc.Printf("%s %s\n", icon, analysis.Message)
 // ...
}
```

**Step 6: Run build to verify**

Run: `make build`

Expected: Build succeeds

**Step 7: Commit**

```bash
git add cmd/root.go
git commit -m "feat: add quiet flag and use British English spelling

- Add -q, --quiet flag to suppress expiry table
- Import color as colour
- Rename getStatusColor to getStatusColour
- Rename all color variables to colour"
```

---

## Task 7: Add UK Date Formatting Helper

**Files:**

- Create: `cmd/format.go`
- Create: `cmd/format_test.go`

**Step 1: Write failing test for UK date format**

Create `cmd/format_test.go`:

```go
package cmd

import (
 "testing"
 "time"
)

func TestFormatUKDate(t *testing.T) {
 tests := []struct {
 name string
 date time.Time
 expected string
 }{
 {
 name: "standard date",
 date: time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC),
 expected: "25 Jul 2024",
 },
 {
 name: "single digit day",
 date: time.Date(2024, 10, 5, 0, 0, 0, 0, time.UTC),
 expected: "5 Oct 2024",
 },
 {
 name: "december",
 date: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
 expected: "31 Dec 2024",
 },
 }

 for _, tt := range tests {
 t.Run(tt.name, func(t *testing.T) {
 result := formatUKDate(tt.date)
 if result != tt.expected {
 t.Errorf("expected %q, got %q", tt.expected, result)
 }
 })
 }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd -v -run TestFormatUKDate`

Expected: FAIL with "undefined: formatUKDate"

**Step 3: Create format.go with helper**

Create `cmd/format.go`:

```go
package cmd

import (
 "time"
)

// formatUKDate formats a date in UK format: "25 Jul 2024"
func formatUKDate(t time.Time) string {
 return t.Format("2 Jan 2006")
}

// formatDaysAgo returns a human-readable string for days
func formatDaysAgo(days int) string {
 if days < 0 {
 return formatDaysInFuture(-days)
 }
 if days == 0 {
 return "today"
 }
 if days == 1 {
 return "1 day ago"
 }
 return formatInt(days) + " days ago"
}

// formatDaysInFuture returns a human-readable string for future days
func formatDaysInFuture(days int) string {
 if days == 0 {
 return "today"
 }
 if days == 1 {
 return "1 day"
 }
 return formatInt(days) + " days"
}

// formatInt converts int to string
func formatInt(n int) string {
 return string(rune(n + '0'))
}
```

Wait - formatInt is wrong. Let me fix:

```go
package cmd

import (
 "fmt"
 "time"
)

// formatUKDate formats a date in UK format: "25 Jul 2024"
func formatUKDate(t time.Time) string {
 return t.Format("2 Jan 2006")
}

// formatDaysAgo returns a human-readable string for days
func formatDaysAgo(days int) string {
 if days < 0 {
 return formatDaysInFuture(-days)
 }
 if days == 0 {
 return "today"
 }
 if days == 1 {
 return "1 day ago"
 }
 return fmt.Sprintf("%d days ago", days)
}

// formatDaysInFuture returns a human-readable string for future days
func formatDaysInFuture(days int) string {
 if days == 0 {
 return "today"
 }
 if days == 1 {
 return "1 day"
 }
 return fmt.Sprintf("%d days", days)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd -v -run TestFormatUKDate`

Expected: PASS

**Step 5: Commit**

```bash
git add cmd/format.go cmd/format_test.go
git commit -m "feat: add UK date formatting helpers

- Add formatUKDate for '25 Jul 2024' format
- Add formatDaysAgo for human-readable day counts
- Test date formatting"
```

---

## Task 8: Update printStatus with New Format

**Files:**

- Modify: `cmd/root.go`

**Step 1: Replace printStatus function**

In `cmd/root.go`, replace the entire `printStatus` function:

```go
func printStatus(analysis *version.Analysis) {
 status := analysis.Status()
 icon := getStatusIcon(status)
 colourFunc := getStatusColour(status)

 var statusLine string

 if analysis.ComparisonVersion == nil {
 // No comparison - just show latest
 statusLine = fmt.Sprintf("%s Latest version: v%s", icon, analysis.LatestVersion)
 } else if analysis.IsLatest {
 // On latest version
 if analysis.ComparisonReleasedAt != nil {
 statusLine = fmt.Sprintf("%s Version %s (%s) is the latest version",
 icon,
 analysis.ComparisonVersion,
 formatUKDate(*analysis.ComparisonReleasedAt))
 } else {
 statusLine = fmt.Sprintf("%s Version %s is the latest version",
 icon,
 analysis.ComparisonVersion)
 }
 } else {
 // Behind - construct full status line
 comparisonDate := ""
 if analysis.ComparisonReleasedAt != nil {
 comparisonDate = fmt.Sprintf(" (%s)", formatUKDate(*analysis.ComparisonReleasedAt))
 }

 expiryInfo := ""
 if analysis.FirstNewerReleaseDate != nil {
 expiryDate := analysis.FirstNewerReleaseDate.AddDate(0, 0, 30)

 if analysis.IsExpired {
 expiryInfo = fmt.Sprintf(" EXPIRED %s", formatUKDate(expiryDate))
 } else if analysis.IsCritical {
 daysLeft := 30 - analysis.DaysSinceUpdate
 expiryInfo = fmt.Sprintf(" EXPIRES %s (%d days)", formatUKDate(expiryDate), daysLeft)
 } else {
 expiryInfo = fmt.Sprintf(" expires %s", formatUKDate(expiryDate))
 }
 }

 latestDate := ""
 for _, r := range analysis.RecentReleases {
 if r.IsLatest {
 latestDate = fmt.Sprintf(" (Released %s)", formatUKDate(r.ReleasedAt))
 break
 }
 }

 statusLine = fmt.Sprintf("%s Version %s%s%s: Update to v%s%s",
 icon,
 analysis.ComparisonVersion,
 comparisonDate,
 expiryInfo,
 analysis.LatestVersion,
 latestDate)
 }

 colourFunc.Println(statusLine)
}
```

**Step 2: Run build to verify**

Run: `make build`

Expected: Build succeeds

**Step 3: Test manually**

Run: `./bin/github-actions-runner-version -c 2.327.1`

Expected: New format with all dates visible

**Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat: update status message with UK dates and all information

- Show all key dates in single line
- Format: Version X (date) [STATUS] expiry: Update to Y (date)
- Use UK date format throughout
- Handle current, warning, critical, expired states"
```

---

## Task 9: Add Expiry Timeline Table

**Files:**

- Modify: `cmd/root.go`

**Step 1: Add printExpiryTable function**

Add to `cmd/root.go`:

```go
func printExpiryTable(analysis *version.Analysis) {
 if len(analysis.RecentReleases) == 0 {
 return
 }

 fmt.Println()
 cyan.Println(" Release Expiry Timeline")
 cyan.Println("─────────────────────────────────────────────────────")
 fmt.Printf("%-10s %-14s %-14s %s\n", "Version", "Released", "Expires", "Status")

 for _, release := range analysis.RecentReleases {
 versionStr := release.Version.String()
 releasedStr := formatUKDate(release.ReleasedAt)

 var expiresStr string
 var statusStr string

 if release.IsLatest {
 expiresStr = "-"
 daysAgo := int(time.Since(release.ReleasedAt).Hours() / 24)
 statusStr = fmt.Sprintf(" Latest (%s)", formatDaysAgo(daysAgo))
 } else if release.ExpiresAt != nil {
 expiresStr = formatUKDate(*release.ExpiresAt)

 if release.IsExpired {
 daysExpired := -release.DaysUntilExpiry
 statusStr = fmt.Sprintf(" Expired %s", formatDaysAgo(daysExpired))
 } else {
 statusStr = fmt.Sprintf(" Valid (%s left)", formatDaysInFuture(release.DaysUntilExpiry))
 }
 }

 // Mark user's version
 arrow := ""
 if analysis.ComparisonVersion != nil && release.Version.Equal(analysis.ComparisonVersion) {
 arrow = " ← Your version"
 }

 fmt.Printf("%-10s %-14s %-14s %s%s\n", versionStr, releasedStr, expiresStr, statusStr, arrow)
 }
}
```

**Step 2: Call printExpiryTable in outputTerminal**

In `cmd/root.go`, update `outputTerminal`:

```go
func outputTerminal(analysis *version.Analysis) error {
 // Always print latest version first (for script compatibility)
 fmt.Println(analysis.LatestVersion)

 // If no comparison, we're done
 if analysis.ComparisonVersion == nil {
 return nil
 }

 // Print status
 fmt.Println()
 printStatus(analysis)

 // Print expiry table unless quiet mode
 if !quiet {
 printExpiryTable(analysis)
 }

 // Print verbose details if requested
 if verbose {
 fmt.Println()
 printDetails(analysis)
 }

 return nil
}
```

**Step 3: Run build and test**

Run: `make build && ./bin/github-actions-runner-version -c 2.327.1`

Expected: Table displays with UK dates

**Step 4: Test quiet flag**

Run: `./bin/github-actions-runner-version -c 2.327.1 -q`

Expected: No table displayed

**Step 5: Commit**

```bash
git add cmd/root.go
git commit -m "feat: add release expiry timeline table

- Display recent releases with expiry dates
- Show version, release date, expiry date, status
- Mark user's version with arrow
- Use UK date format
- Suppress with -q/--quiet flag"
```

---

## Task 10: Update CI Output Format

**Files:**

- Modify: `cmd/root.go`

**Step 1: Update outputCI function**

In `cmd/root.go`, replace the status printing in `outputCI`:

```go
func outputCI(analysis *version.Analysis) error {
 // Always print latest version first (for script compatibility)
 fmt.Println(analysis.LatestVersion)

 // If no comparison, we're done
 if analysis.ComparisonVersion == nil {
 return nil
 }

 status := analysis.Status()

 // Build status line (same as terminal output but without colours)
 var statusLine string
 if analysis.IsLatest {
 comparisonDate := ""
 if analysis.ComparisonReleasedAt != nil {
 comparisonDate = fmt.Sprintf(" (%s)", formatUKDate(*analysis.ComparisonReleasedAt))
 }
 statusLine = fmt.Sprintf("Version %s%s is the latest version",
 analysis.ComparisonVersion,
 comparisonDate)
 } else {
 comparisonDate := ""
 if analysis.ComparisonReleasedAt != nil {
 comparisonDate = fmt.Sprintf(" (%s)", formatUKDate(*analysis.ComparisonReleasedAt))
 }

 expiryInfo := ""
 if analysis.FirstNewerReleaseDate != nil {
 expiryDate := analysis.FirstNewerReleaseDate.AddDate(0, 0, 30)

 if analysis.IsExpired {
 expiryInfo = fmt.Sprintf(" EXPIRED %s", formatUKDate(expiryDate))
 } else if analysis.IsCritical {
 daysLeft := 30 - analysis.DaysSinceUpdate
 expiryInfo = fmt.Sprintf(" EXPIRES %s (%d days)", formatUKDate(expiryDate), daysLeft)
 } else {
 expiryInfo = fmt.Sprintf(" expires %s", formatUKDate(expiryDate))
 }
 }

 latestDate := ""
 for _, r := range analysis.RecentReleases {
 if r.IsLatest {
 latestDate = fmt.Sprintf(" (Released %s)", formatUKDate(r.ReleasedAt))
 break
 }
 }

 statusLine = fmt.Sprintf("Version %s%s%s: Update to v%s%s",
 analysis.ComparisonVersion,
 comparisonDate,
 expiryInfo,
 analysis.LatestVersion,
 latestDate)
 }

 // Print GitHub Actions workflow commands
 fmt.Println()
 fmt.Println("::group:: Runner Version Check")
 fmt.Printf("Latest version: v%s\n", analysis.LatestVersion)
 fmt.Printf("Your version: v%s\n", analysis.ComparisonVersion)
 fmt.Printf("Status: %s\n", getStatusText(status))
 fmt.Println("::endgroup::")
 fmt.Println()

 // Use appropriate workflow command based on status
 icon := getStatusIcon(status)
 switch status {
 case version.StatusExpired:
 fmt.Printf("::error title=Runner Version Expired::%s %s\n", icon, statusLine)
 case version.StatusCritical:
 fmt.Printf("::warning title=Runner Version Critical::%s %s\n", icon, statusLine)
 case version.StatusWarning:
 fmt.Printf("::notice title=Runner Version Behind::%s %s\n", icon, statusLine)
 case version.StatusCurrent:
 fmt.Printf("::notice title=Runner Version Current::%s %s\n", icon, statusLine)
 }

 // Print expiry table
 if len(analysis.RecentReleases) > 0 {
 fmt.Println()
 fmt.Println("::group:: Release Expiry Timeline")
 fmt.Printf("%-10s %-14s %-14s %s\n", "Version", "Released", "Expires", "Status")

 for _, release := range analysis.RecentReleases {
 versionStr := release.Version.String()
 releasedStr := formatUKDate(release.ReleasedAt)

 var expiresStr string
 var statusStr string

 if release.IsLatest {
 expiresStr = "-"
 daysAgo := int(time.Since(release.ReleasedAt).Hours() / 24)
 statusStr = fmt.Sprintf("Latest (%s)", formatDaysAgo(daysAgo))
 } else if release.ExpiresAt != nil {
 expiresStr = formatUKDate(*release.ExpiresAt)

 if release.IsExpired {
 daysExpired := -release.DaysUntilExpiry
 statusStr = fmt.Sprintf("Expired %s", formatDaysAgo(daysExpired))
 } else {
 statusStr = fmt.Sprintf("Valid (%s left)", formatDaysInFuture(release.DaysUntilExpiry))
 }
 }

 arrow := ""
 if analysis.ComparisonVersion != nil && release.Version.Equal(analysis.ComparisonVersion) {
 arrow = " [Your version]"
 }

 fmt.Printf(" %-10s %-14s %-14s %s%s\n", versionStr, releasedStr, expiresStr, statusStr, arrow)
 }

 fmt.Println("::endgroup::")
 }

 // Write markdown summary to $GITHUB_STEP_SUMMARY
 if summaryFile := os.Getenv("GITHUB_STEP_SUMMARY"); summaryFile != "" {
 if err := writeGitHubSummary(summaryFile, analysis); err != nil {
 fmt.Printf("::warning::Failed to write job summary: %v\n", err)
 }
 }

 return nil
}
```

**Step 2: Update writeGitHubSummary to use UK dates**

In `cmd/root.go`, update date formatting in `writeGitHubSummary`:

Find lines with `.Format("Jan 2, 2006")` and replace with calls to `formatUKDate()`.

**Step 3: Run build to verify**

Run: `make build`

Expected: Build succeeds

**Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat: update CI output with new format and expiry table

- Use new status line format in workflow commands
- Include expiry timeline table in ::group::
- Use UK date format in GitHub Actions summary
- Update writeGitHubSummary to use formatUKDate"
```

---

## Task 11: Add golangci-lint Configuration

**Files:**

- Create: `.golangci.yml`

**Step 1: Create .golangci.yml**

Create `.golangci.yml`:

```yaml
linters:
 enable:
 - misspell
 - gofmt
 - govet

linters-settings:
 misspell:
 locale: UK
 ignore-words:
 - color # Keep for imported package names (fatih/color)
```

**Step 2: Run linter**

Run: `make lint`

Expected: Linter runs and checks for British spelling

**Step 3: Commit**

```bash
git add .golangci.yml
git commit -m "feat: add golangci-lint config with UK locale

- Configure misspell linter for British English
- Ignore 'color' in package names
- Enable gofmt and govet"
```

---

## Task 12: Update Documentation

**Files:**

- Modify: `README.md`
- Modify: `CLAUDE.md`

**Step 1: Update README.md with new output examples**

In `README.md`, update the example outputs to show new format:

Find the "Example 2: Check Expired Version" section and update:

```markdown
### Example 2: Check Expired Version

```bash
$ github-actions-runner-version -c 2.327.1
1.329.0

 Version 2.327.1 (25 Jul 2024) EXPIRED 24 Aug 2024: Update to v2.329.0 (Released 14 Oct 2024)

 Release Expiry Timeline
─────────────────────────────────────────────────────
Version Released Expires Status
1.327.1 25 Jul 2024 24 Aug 2024 Expired 77 days ago ← Your version
1.328.0 13 Aug 2024 12 Sep 2024 Expired 48 days ago
1.328.1 10 Sep 2024 13 Nov 2024 Valid (28 days left)
1.329.0 14 Oct 2024 - Latest (2 days ago)
```

```

Update other examples similarly. Add quiet flag documentation:

```markdown
# Quiet output (suppress expiry table)
github-actions-runner-version -c 2.327.1 -q
```

**Step 2: Update CLAUDE.md with new method names**

In `CLAUDE.md`, update references:

- `Analyze()` → `Analyse()`
- `color` → `colour`
- Update output examples to show new format

**Step 3: Add note about British English**

Add to `README.md` in a "Development" section:

```markdown
## Development

This project uses British English spelling throughout:
- `Analyse` (not Analyze)
- `colour` (not color)

Linting enforces British spelling via `golangci-lint` with UK locale.
```

**Step 4: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: update documentation with new output format

- Update example outputs with new status line format
- Show expiry timeline table
- Document -q/--quiet flag
- Update method names (Analyse, colour)
- Add note about British English"
```

---

## Task 13: Run Full Test Suite and Verify

**Files:**

- All

**Step 1: Run all tests**

Run: `make test`

Expected: All tests pass

**Step 2: Run linter**

Run: `make lint`

Expected: No errors

**Step 3: Build the project**

Run: `make build`

Expected: Build succeeds

**Step 4: Test manually with real API**

Run: `./bin/github-actions-runner-version -c 2.327.1`

Expected: Shows new format with expiry table

Run: `./bin/github-actions-runner-version -c 2.327.99`

Expected: Error message about non-existent version

Run: `./bin/github-actions-runner-version -c 2.328.1 -q`

Expected: No table displayed

**Step 5: Commit if any fixes needed**

If tests revealed issues and you fixed them:

```bash
git add .
git commit -m "fix: address test failures and linting issues"
```

---

## Task 14: Final Integration Test

**Files:**

- None (testing only)

**Step 1: Test all output modes**

Terminal:

```bash
./bin/github-actions-runner-version -c 2.327.1
./bin/github-actions-runner-version -c 2.327.1 -v
./bin/github-actions-runner-version -c 2.327.1 -q
```

JSON:

```bash
./bin/github-actions-runner-version -c 2.327.1 --json
```

CI:

```bash
./bin/github-actions-runner-version -c 2.327.1 --ci
```

**Step 2: Verify British English**

Check that:

- Status messages use "colour" not "color"
- Method names are "Analyse" not "Analyze"
- Comments use British spelling

**Step 3: Verify expiry calculations**

Check that:

- Latest version has no expiry date ("-")
- Expiry dates are 30 days after next release
- Table shows 90-day window or minimum 4 releases
- Dates are UK format ("25 Jul 2024")

**Step 4: Document any issues**

If found, create new tasks to address them.

---

## Completion Checklist

- [ ] All tests passing (`make test`)
- [ ] Linter passing (`make lint`)
- [ ] Build succeeds (`make build`)
- [ ] Version validation rejects non-existent versions
- [ ] Expiry table displays with UK dates
- [ ] Quiet flag suppresses table
- [ ] JSON output includes RecentReleases
- [ ] CI output includes expiry table
- [ ] British English throughout (Analyse, colour)
- [ ] Documentation updated

---

## Next Steps

After completing implementation:

1. Review all changes
1. Run full test suite one final time
1. Ready for code review (use `superpowers:requesting-code-review`)
