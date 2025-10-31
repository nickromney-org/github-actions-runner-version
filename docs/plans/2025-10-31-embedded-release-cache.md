# Embedded Release Cache Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce API calls from 2 to 1 per invocation by embedding historical release data in the binary.

**Architecture:** Embed releases.json via go:embed, always fetch 5 most recent from API, validate embedded data is current, merge datasets.

**Tech Stack:** Go 1.21+, go:embed, go-github/v57

---

## Task 1: Create Bootstrap Script

**Files:**
- Create: `scripts/update-releases.sh`
- Create: `data/.gitkeep`

**Step 1: Create data directory**

```bash
mkdir -p data
touch data/.gitkeep
```

**Step 2: Create bootstrap script**

Create `scripts/update-releases.sh`:

```bash
#!/bin/bash
set -euo pipefail

GITHUB_TOKEN="${GITHUB_TOKEN:-}"
OUTPUT_FILE="data/releases.json"

echo "Fetching all releases from GitHub..."

# Use go run to bootstrap
go run ./cmd/bootstrap-releases \
  ${GITHUB_TOKEN:+--token "$GITHUB_TOKEN"} \
  --output "$OUTPUT_FILE"

RELEASE_COUNT=$(jq '.releases | length' "$OUTPUT_FILE")
echo "✅ Updated $OUTPUT_FILE with $RELEASE_COUNT releases"
```

**Step 3: Make executable**

Run: `chmod +x scripts/update-releases.sh`

Expected: Script is executable

**Step 4: Commit**

```bash
git add data/.gitkeep scripts/update-releases.sh
git commit -m "feat: add bootstrap script for release cache"
```

---

## Task 2: Create Bootstrap Command

**Files:**
- Create: `cmd/bootstrap-releases/main.go`

**Step 1: Create bootstrap command**

Create `cmd/bootstrap-releases/main.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nickromney-org/github-actions-runner-version/internal/github"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
)

type ReleaseJSON struct {
	Version     string    `json:"version"`
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url"`
}

type CacheFile struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Releases    []ReleaseJSON `json:"releases"`
}

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	output := flag.String("output", "data/releases.json", "Output file")
	flag.Parse()

	client := github.NewClient(*token)
	ctx := context.Background()

	fmt.Println("Fetching all releases from GitHub API...")

	// Fetch all releases
	releases, err := client.GetAllReleases(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Convert to JSON-friendly format
	jsonReleases := make([]ReleaseJSON, len(releases))
	for i, r := range releases {
		jsonReleases[i] = ReleaseJSON{
			Version:     r.Version.String(),
			PublishedAt: r.PublishedAt,
			URL:         r.URL,
		}
	}

	// Build cache file
	cache := CacheFile{
		GeneratedAt: time.Now().UTC(),
		Releases:    jsonReleases,
	}

	// Write to file
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cache); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Wrote %d releases to %s\n", len(releases), *output)
}
```

**Step 2: Run bootstrap to generate initial data**

Run: `./scripts/update-releases.sh`

Expected: Creates `data/releases.json` with all releases

**Step 3: Verify output**

Run: `jq '.releases | length' data/releases.json`

Expected: Shows release count (~300-400)

Run: `jq '.releases[0]' data/releases.json`

Expected: Shows most recent release

**Step 4: Commit**

```bash
git add cmd/bootstrap-releases/main.go data/releases.json
git commit -m "feat: add bootstrap command and initial release cache

Generated cache contains all releases from v2.159.0 to present"
```

---

## Task 3: Create Data Loading Package

**Files:**
- Create: `internal/data/loader.go`
- Create: `internal/data/loader_test.go`

**Step 1: Write failing test**

Create `internal/data/loader_test.go`:

```go
package data

import (
	"testing"
)

func TestLoadEmbeddedReleases(t *testing.T) {
	releases, err := LoadEmbeddedReleases()
	if err != nil {
		t.Fatalf("LoadEmbeddedReleases failed: %v", err)
	}

	if len(releases) == 0 {
		t.Error("expected releases, got empty slice")
	}

	// Check first release has valid data
	first := releases[0]
	if first.Version == nil {
		t.Error("first release has nil version")
	}
	if first.PublishedAt.IsZero() {
		t.Error("first release has zero published date")
	}
	if first.URL == "" {
		t.Error("first release has empty URL")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/data -v`

Expected: FAIL with "undefined: LoadEmbeddedReleases"

**Step 3: Create loader with go:embed**

Create `internal/data/loader.go`:

```go
package data

import (
	_ "embed"
	"encoding/json"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
)

//go:embed releases.json
var releasesJSON []byte

type CachedReleases struct {
	GeneratedAt time.Time `json:"generated_at"`
	Releases    []struct {
		Version     string    `json:"version"`
		PublishedAt time.Time `json:"published_at"`
		URL         string    `json:"url"`
	} `json:"releases"`
}

// LoadEmbeddedReleases loads releases from embedded JSON
func LoadEmbeddedReleases() ([]version.Release, error) {
	var cached CachedReleases
	if err := json.Unmarshal(releasesJSON, &cached); err != nil {
		return nil, err
	}

	releases := make([]version.Release, 0, len(cached.Releases))
	for _, r := range cached.Releases {
		ver, err := semver.NewVersion(r.Version)
		if err != nil {
			// Skip invalid versions
			continue
		}
		releases = append(releases, version.Release{
			Version:     ver,
			PublishedAt: r.PublishedAt,
			URL:         r.URL,
		})
	}

	return releases, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/data -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/data/
git commit -m "feat: add embedded release data loader

- Use go:embed to embed releases.json
- LoadEmbeddedReleases() parses JSON and returns Release slice
- Test validates loading and parsing"
```

---

## Task 4: Add GetRecentReleases to GitHub Client

**Files:**
- Modify: `internal/github/client.go`
- Modify: `internal/github/client_test.go` (if exists, otherwise skip)

**Step 1: Add GetRecentReleases method**

Add to `internal/github/client.go` after `GetAllReleases`:

```go
// GetRecentReleases fetches only the N most recent releases
func (c *Client) GetRecentReleases(ctx context.Context, count int) ([]version.Release, error) {
	opts := &gh.ListOptions{PerPage: count}

	releases, _, err := c.gh.Repositories.ListReleases(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent releases: %w", err)
	}

	var result []version.Release
	for _, ghRelease := range releases {
		// Skip drafts and prereleases
		if ghRelease.GetDraft() || ghRelease.GetPrerelease() {
			continue
		}

		release, err := c.parseRelease(ghRelease)
		if err != nil {
			// Log but don't fail - just skip invalid releases
			continue
		}

		result = append(result, *release)
	}

	return result, nil
}
```

**Step 2: Update MockClient**

Add to `internal/github/client.go` MockClient:

```go
// GetRecentReleases returns the first N mocked releases
func (m *MockClient) GetRecentReleases(ctx context.Context, count int) ([]version.Release, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if len(m.AllReleases) <= count {
		return m.AllReleases, nil
	}
	return m.AllReleases[:count], nil
}
```

**Step 3: Update interface**

Update the `GitHubClient` interface in `internal/version/checker.go`:

```go
// GitHubClient defines the interface for fetching releases
type GitHubClient interface {
	GetLatestRelease(ctx context.Context) (*Release, error)
	GetAllReleases(ctx context.Context) ([]Release, error)
	GetRecentReleases(ctx context.Context, count int) ([]Release, error)
}
```

**Step 4: Build to verify**

Run: `go build ./...`

Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/github/client.go internal/version/checker.go
git commit -m "feat: add GetRecentReleases method to GitHub client

- Fetch only N most recent releases
- Update GitHubClient interface
- Add MockClient implementation"
```

---

## Task 5: Add Cache Validation Logic

**Files:**
- Modify: `internal/version/checker.go`
- Modify: `internal/version/checker_test.go`

**Step 1: Write test for isEmbeddedCurrent**

Add to `internal/version/checker_test.go`:

```go
func TestChecker_IsEmbeddedCurrent(t *testing.T) {
	tests := []struct {
		name     string
		embedded []Release
		recent   []Release
		want     bool
	}{
		{
			name: "embedded is current (latest in top 5)",
			embedded: []Release{
				newTestRelease("2.329.0", 5),
				newTestRelease("2.328.0", 20),
			},
			recent: []Release{
				newTestRelease("2.329.0", 5),
				newTestRelease("2.328.0", 20),
			},
			want: true,
		},
		{
			name: "embedded is stale (latest not in top 5)",
			embedded: []Release{
				newTestRelease("2.320.0", 100),
				newTestRelease("2.319.0", 110),
			},
			recent: []Release{
				newTestRelease("2.329.0", 5),
				newTestRelease("2.328.0", 20),
			},
			want: false,
		},
		{
			name:     "empty embedded",
			embedded: []Release{},
			recent: []Release{
				newTestRelease("2.329.0", 5),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &Checker{}
			got := checker.isEmbeddedCurrent(tt.embedded, tt.recent)
			if got != tt.want {
				t.Errorf("isEmbeddedCurrent() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/version -v -run TestChecker_IsEmbeddedCurrent`

Expected: FAIL with "undefined: Checker.isEmbeddedCurrent"

**Step 3: Implement isEmbeddedCurrent**

Add to `internal/version/checker.go`:

```go
// isEmbeddedCurrent checks if embedded data contains the latest release
// by verifying the latest embedded version is in the recent 5 releases
func (c *Checker) isEmbeddedCurrent(embedded, recent []Release) bool {
	if len(embedded) == 0 || len(recent) == 0 {
		return false
	}

	// Find latest embedded release
	latestEmbedded := embedded[0]
	for _, r := range embedded {
		if r.Version.GreaterThan(latestEmbedded.Version) {
			latestEmbedded = r
		}
	}

	// Check if it exists in recent 5
	for _, r := range recent {
		if r.Version.Equal(latestEmbedded.Version) {
			return true
		}
	}

	return false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/version -v -run TestChecker_IsEmbeddedCurrent`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/version/checker.go internal/version/checker_test.go
git commit -m "feat: add cache validation logic

- isEmbeddedCurrent checks if latest embedded release is in top 5
- Test coverage for current, stale, and edge cases"
```

---

## Task 6: Add Merge Logic

**Files:**
- Modify: `internal/version/checker.go`
- Modify: `internal/version/checker_test.go`

**Step 1: Write test for mergeReleases**

Add to `internal/version/checker_test.go`:

```go
func TestChecker_MergeReleases(t *testing.T) {
	embedded := []Release{
		newTestRelease("2.327.0", 50),
		newTestRelease("2.326.0", 60),
	}

	recent := []Release{
		newTestRelease("2.329.0", 5),
		newTestRelease("2.328.0", 20),
		newTestRelease("2.327.0", 50), // Duplicate
	}

	checker := &Checker{}
	merged := checker.mergeReleases(embedded, recent)

	// Should have 4 unique releases (deduplicated 2.327.0)
	if len(merged) != 4 {
		t.Errorf("expected 4 releases, got %d", len(merged))
	}

	// Check all versions present
	versions := make(map[string]bool)
	for _, r := range merged {
		versions[r.Version.String()] = true
	}

	expected := []string{"2.329.0", "2.328.0", "2.327.0", "2.326.0"}
	for _, v := range expected {
		if !versions[v] {
			t.Errorf("missing version %s in merged releases", v)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/version -v -run TestChecker_MergeReleases`

Expected: FAIL with "undefined: Checker.mergeReleases"

**Step 3: Implement mergeReleases**

Add to `internal/version/checker.go`:

```go
// mergeReleases combines embedded and recent releases, deduplicating by version
func (c *Checker) mergeReleases(embedded, recent []Release) []Release {
	// Use map to deduplicate by version
	seen := make(map[string]bool)
	var merged []Release

	// Add recent first (they're authoritative)
	for _, r := range recent {
		key := r.Version.String()
		if !seen[key] {
			seen[key] = true
			merged = append(merged, r)
		}
	}

	// Add embedded (skip duplicates)
	for _, r := range embedded {
		key := r.Version.String()
		if !seen[key] {
			seen[key] = true
			merged = append(merged, r)
		}
	}

	return merged
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/version -v -run TestChecker_MergeReleases`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/version/checker.go internal/version/checker_test.go
git commit -m "feat: add release merge logic

- mergeReleases combines embedded and API data
- Deduplicates by version string
- Recent releases take precedence
- Test validates deduplication"
```

---

## Task 7: Update Analyse Method to Use Cache

**Files:**
- Modify: `internal/version/checker.go`

**Step 1: Import data package**

Add to imports in `internal/version/checker.go`:

```go
import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nickromney-org/github-actions-runner-version/internal/data"
)
```

**Step 2: Update Analyse method**

Replace the beginning of `Analyse()` method (before version validation):

```go
func (c *Checker) Analyse(ctx context.Context, comparisonVersionStr string) (*Analysis, error) {
	// Validate config
	if err := c.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Load embedded releases
	embeddedReleases, err := data.LoadEmbeddedReleases()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded releases: %w", err)
	}

	// Fetch 5 most recent releases from API
	recentReleases, err := c.client.GetRecentReleases(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent releases: %w", err)
	}

	// Determine which dataset to use
	var allReleases []Release
	if !c.isEmbeddedCurrent(embeddedReleases, recentReleases) {
		// Embedded data is stale (>5 releases behind)
		// Fall back to full API query
		allReleases, err = c.client.GetAllReleases(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch all releases: %w", err)
		}
	} else {
		// Merge embedded + recent (deduplicating)
		allReleases = c.mergeReleases(embeddedReleases, recentReleases)
	}

	// Get latest release from dataset
	latestRelease := allReleases[0]
	for _, r := range allReleases {
		if r.Version.GreaterThan(latestRelease.Version) {
			latestRelease = r
		}
	}

	// If no comparison version, just return latest
	if comparisonVersionStr == "" {
		return &Analysis{
			LatestVersion:   latestRelease.Version,
			IsLatest:        false,
			CriticalAgeDays: c.config.CriticalAgeDays,
			MaxAgeDays:      c.config.MaxAgeDays,
			Message:         fmt.Sprintf("Latest version: %s", latestRelease.Version),
		}, nil
	}

	// ... rest of existing logic (validation, comparison, etc.)
```

**Step 3: Run tests to verify**

Run: `go test ./internal/version -v`

Expected: All tests PASS

**Step 4: Build to verify**

Run: `go build ./...`

Expected: Build succeeds

**Step 5: Test manually**

Run: `go run . -c 2.327.1`

Expected: Output shows version check (using embedded + API data)

**Step 6: Commit**

```bash
git add internal/version/checker.go
git commit -m "feat: integrate embedded cache into Analyse method

- Load embedded releases at start
- Fetch 5 most recent from API
- Validate embedded data currency
- Merge or fallback to full API query
- Reduces API calls from 2 to 1 per invocation"
```

---

## Task 8: Create Check Releases Command

**Files:**
- Create: `cmd/check-releases/main.go`

**Step 1: Create check command**

Create `cmd/check-releases/main.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nickromney-org/github-actions-runner-version/internal/data"
	"github.com/nickromney-org/github-actions-runner-version/internal/github"
)

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	cacheFile := flag.String("cache", "data/releases.json", "Cache file path")
	flag.Parse()

	client := github.NewClient(*token)
	ctx := context.Background()

	// Load embedded releases
	embedded, err := data.LoadEmbeddedReleases()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading embedded releases: %v\n", err)
		os.Exit(1)
	}

	// Find latest embedded version
	var latestEmbedded string
	for _, r := range embedded {
		if latestEmbedded == "" || r.Version.String() > latestEmbedded {
			latestEmbedded = r.Version.String()
		}
	}

	// Fetch 5 most recent from API
	recent, err := client.GetRecentReleases(ctx, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching recent releases: %v\n", err)
		os.Exit(1)
	}

	// Check if latest embedded is in recent 5
	found := false
	for _, r := range recent {
		if r.Version.String() == latestEmbedded {
			found = true
			break
		}
	}

	if found {
		fmt.Printf("✅ Cache is current (latest: %s)\n", latestEmbedded)
		os.Exit(0)
	}

	fmt.Printf("⚠️  Cache needs update (latest embedded: %s, latest available: %s)\n",
		latestEmbedded, recent[0].Version.String())
	os.Exit(1)
}
```

**Step 2: Test check command**

Run: `go run ./cmd/check-releases`

Expected: Either "Cache is current" (exit 0) or "Cache needs update" (exit 1)

**Step 3: Commit**

```bash
git add cmd/check-releases/main.go
git commit -m "feat: add check-releases command

- Validates embedded cache against API
- Exits 0 if current, 1 if updates needed
- Used by automation to trigger cache updates"
```

---

## Task 9: Create Update Releases Workflow

**Files:**
- Create: `.github/workflows/update-releases.yml`

**Step 1: Create workflow**

Create `.github/workflows/update-releases.yml`:

```yaml
name: Update Release Cache

on:
  schedule:
    - cron: "0 6 * * *"  # Daily at 6 AM UTC (3 hours before runner check)
  workflow_dispatch:

permissions:
  contents: write

jobs:
  update-cache:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.21"

      - name: Check if cache needs update
        id: check
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          go run ./cmd/check-releases \
            --token "$GITHUB_TOKEN" \
            --cache data/releases.json
        continue-on-error: true

      - name: Update release cache
        if: steps.check.outcome == 'failure'
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          chmod +x scripts/update-releases.sh
          ./scripts/update-releases.sh

      - name: Commit and push changes
        if: steps.check.outcome == 'failure'
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"

          git add data/releases.json

          # Check if there are changes
          if git diff --staged --quiet; then
            echo "No changes to commit"
            exit 0
          fi

          git commit -m "chore: update release cache with latest runner versions [skip ci]"
          git push
```

**Step 2: Commit**

```bash
git add .github/workflows/update-releases.yml
git commit -m "feat: add automated release cache update workflow

- Runs daily at 6 AM UTC
- Checks if cache needs update
- Updates and commits if new releases found
- Triggers semantic-release for new tool build"
```

---

## Task 10: Update Check Runner Workflow

**Files:**
- Modify: `.github/workflows/check-runner.yml`

**Step 1: Update binary download URL**

In `.github/workflows/check-runner.yml`, find the "Download version checker" step and update:

```yaml
      - name: Download version checker
        run: |
          # Use correct repository URL and binary name
          curl -LO https://github.com/nickromney-org/github-actions-runner-version/releases/latest/download/github-actions-runner-version-linux-amd64
          chmod +x github-actions-runner-version-linux-amd64
          sudo mv github-actions-runner-version-linux-amd64 /usr/local/bin/github-actions-runner-version
```

**Step 2: Update check command**

In the "Check version compliance" step:

```yaml
      - name: Check version compliance
        id: check
        run: |
          # GITHUB_TOKEN is automatically detected - no need to pass -t flag
          github-actions-runner-version -c ${{ steps.runner-version.outputs.version }} --ci
        continue-on-error: true
```

**Step 3: Commit**

```bash
git add .github/workflows/check-runner.yml
git commit -m "fix: update check-runner workflow for new binary name

- Use github-actions-runner-version binary name
- Update repository URL
- Update command invocation"
```

---

## Task 11: Test End-to-End

**Files:**
- None (testing only)

**Step 1: Rebuild binary**

Run: `make clean && make build`

Expected: Build succeeds with embedded data

**Step 2: Test with embedded cache (no token)**

Run: `./bin/github-actions-runner-version -c 2.327.1`

Expected: Output shows version check, makes 1 API call for recent releases

**Step 3: Test cache validation**

Run: `go run ./cmd/check-releases`

Expected: Shows cache status

**Step 4: Test with non-existent version**

Run: `./bin/github-actions-runner-version -c 2.327.99`

Expected: Error about non-existent version

**Step 5: Verify binary size**

Run: `ls -lh bin/github-actions-runner-version`

Expected: Binary size reasonable (~8-10MB with embedded JSON)

**Step 6: Run all tests**

Run: `make test`

Expected: All tests PASS

**Step 7: Run linter**

Run: `make lint`

Expected: No linting errors

---

## Task 12: Update Documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Step 1: Update README.md with cache information**

Add a new section to README.md:

```markdown
## Release Data Caching

This tool embeds historical release data to minimize GitHub API calls:

- **Embedded cache:** All releases from v2.159.0 to build time
- **API calls:** Always fetches 5 most recent releases (1 API call)
- **Validation:** Checks if embedded cache is current (within top 5)
- **Fallback:** Full API query if cache >5 releases behind
- **Updates:** Automated daily checks trigger new tool builds

**Result:** Reduces API calls from 2 per invocation to 1, improving speed and rate limit usage.
```

**Step 2: Update CLAUDE.md**

Update the architecture section:

```markdown
### Key Design Patterns

- **Interface-based design**: `GitHubClient` interface allows easy mocking in tests
- **Semantic versioning**: Uses `Masterminds/semver/v3` for proper version comparison
- **Configuration validation**: `CheckerConfig.Validate()` ensures critical days < max days
- **Embedded cache**: Historical releases embedded via `go:embed` for minimal API usage
- **No database**: Stateless tool with embedded data
```

**Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: document embedded release cache feature

- Explain caching strategy in README
- Update CLAUDE.md architecture notes
- Highlight API call reduction"
```

---

## Completion Checklist

- [ ] Bootstrap script created and working
- [ ] Initial releases.json generated with ~300-400 releases
- [ ] Embedded data loader implemented and tested
- [ ] GetRecentReleases method added to GitHub client
- [ ] Cache validation logic (isEmbeddedCurrent) tested
- [ ] Merge logic (mergeReleases) tested
- [ ] Analyse method updated to use cache
- [ ] Check-releases command working
- [ ] Update-releases workflow created
- [ ] Check-runner workflow updated
- [ ] End-to-end testing complete
- [ ] Documentation updated
- [ ] All tests passing
- [ ] Linter passing

---

## Success Metrics

- ✅ API calls reduced from 2 to 1 per invocation
- ✅ Embedded cache contains 300-400 releases
- ✅ Binary size increase <2MB
- ✅ Tool remains atomic (single binary)
- ✅ Automated updates working (daily cron)
- ✅ Fallback to full API query when needed
