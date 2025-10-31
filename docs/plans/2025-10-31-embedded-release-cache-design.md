# Embedded Release Cache Design

**Date:** 31 October 2025
**Status:** Approved
**Author:** Design Session with User

## Problem Statement

Currently, the tool makes 2 API calls per invocation:
1. `GetLatestRelease()` - fetch latest version
2. `GetAllReleases()` - fetch up to 1000 releases (10 pages × 100 per page)

**Issues:**
- Unnecessary API consumption for historical data that rarely changes
- Rate limit concerns (60/hour unauth, 5000/hour auth)
- Slow for users without tokens
- Historical releases from v2.159.0 (mid-2019) are constant

## Design Goals

1. **Minimize API calls** - Reduce from 2 per invocation to 1
2. **Maintain accuracy** - Stay current with recent releases
3. **Preserve atomicity** - CLI remains single portable binary
4. **Automatic updates** - Tool rebuilds when new runner releases appear
5. **Rate limit friendly** - Predictable, minimal API usage

## Solution Architecture

### 1. Embedded Release Cache

**Storage:** `data/releases.json` embedded in binary via `//go:embed`

**Structure:**
```json
{
  "generated_at": "2025-10-31T12:00:00Z",
  "releases": [
    {
      "version": "2.329.0",
      "published_at": "2024-10-14T00:00:00Z",
      "url": "https://github.com/actions/runner/releases/tag/v2.329.0"
    },
    {
      "version": "2.328.0",
      "published_at": "2024-08-13T00:00:00Z",
      "url": "https://github.com/actions/runner/releases/tag/v2.328.0"
    }
  ]
}
```

**Size estimate:**
- ~300-400 releases from v2.159.0 to present
- ~50-100KB JSON uncompressed
- Negligible impact on binary size

**Embedding:**
```go
package data

import _ "embed"

//go:embed releases.json
var ReleasesJSON []byte
```

### 2. Runtime Validation Strategy

**On every CLI invocation:**

```
1. Load embedded releases from data.ReleasesJSON
2. API call: Fetch 5 most recent releases (single request)
3. Validate: Is embedded latest release in those 5?
   ├─ YES → Embedded data is current (0-4 releases behind)
   │        Use: embedded + API top 5 (merged dataset)
   └─ NO  → Embedded data is stale (>5 releases behind)
            Fallback: Full API query for all releases
```

**Benefits:**
- **Predictable:** Exactly 1 API call per invocation
- **Simple validation:** Check if latest embedded version exists in top 5
- **Covers ~5 weeks:** With weekly release cadence, 5 releases = ~35 days buffer
- **Rate limit friendly:** 1 request/check × 5000/hour limit = 5000 checks/hour
- **Always accurate:** Recent releases always fresh from API

**No binary age checks needed** - the data validates itself by comparing versions.

### 3. Initial Bootstrap

**One-time script:** `scripts/update-releases.sh`

```bash
#!/bin/bash
set -euo pipefail

GITHUB_TOKEN="${GITHUB_TOKEN:-}"
OUTPUT_FILE="data/releases.json"

echo "Fetching all releases from GitHub..."

# Call GitHub API to fetch all releases
# Use our binary's GitHub client logic
go run ./cmd/bootstrap-releases \
  ${GITHUB_TOKEN:+--token "$GITHUB_TOKEN"} \
  --output "$OUTPUT_FILE"

RELEASE_COUNT=$(jq '.releases | length' "$OUTPUT_FILE")
echo "✅ Updated $OUTPUT_FILE with $RELEASE_COUNT releases"
```

**Bootstrap program:** `cmd/bootstrap-releases/main.go`

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
)

func main() {
    token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
    output := flag.String("output", "data/releases.json", "Output file")
    flag.Parse()

    client := github.NewClient(*token)
    ctx := context.Background()

    // Fetch all releases
    releases, err := client.GetAllReleases(ctx)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Build JSON structure
    data := struct {
        GeneratedAt time.Time           `json:"generated_at"`
        Releases    []github.ReleaseJSON `json:"releases"`
    }{
        GeneratedAt: time.Now().UTC(),
        Releases:    convertToJSON(releases),
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
    if err := encoder.Encode(data); err != nil {
        fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("✅ Wrote %d releases to %s\n", len(releases), *output)
}
```

### 4. Automated Updates via Cron

**New workflow:** `.github/workflows/update-releases.yml`

**Triggers:**
- Daily cron: 6:00 AM UTC (3 hours before runner check at 9:00 AM)
- Manual dispatch

**Logic:**

```yaml
name: Update Release Cache

on:
  schedule:
    - cron: "0 6 * * *"  # Daily at 6 AM UTC
  workflow_dispatch:

jobs:
  update-releases:
    runs-on: ubuntu-latest
    permissions:
      contents: write

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.21"

      - name: Fetch latest releases
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          # Fetch 5 most recent releases
          go run ./cmd/check-releases \
            --token "$GITHUB_TOKEN" \
            --cache data/releases.json \
            --check

          # Exit code 0 = no updates needed
          # Exit code 1 = updates needed

      - name: Update release cache
        if: failure()  # Previous step exits 1 when updates needed
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          ./scripts/update-releases.sh

      - name: Commit and push
        if: failure()
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add data/releases.json
          git commit -m "chore: update release cache with latest runner versions [skip ci]"
          git push
```

**Check program:** `cmd/check-releases/main.go` (returns exit 1 if updates needed)

**Result:**
- Commit to main triggers semantic-release
- Semantic-release creates new tag (patch bump for cache updates)
- Tag triggers ci.yml → builds new binaries with fresh embedded data
- New binaries distributed via GitHub Releases

### 5. Code Changes

**New package:** `internal/data/`

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
            continue // Skip invalid versions
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

**Updated:** `internal/github/client.go`

```go
// GetRecentReleases fetches only the N most recent releases
func (c *Client) GetRecentReleases(ctx context.Context, count int) ([]version.Release, error) {
    opts := &gh.ListOptions{PerPage: count}

    releases, _, err := c.gh.Repositories.ListReleases(ctx, owner, repo, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to list releases: %w", err)
    }

    var result []version.Release
    for _, ghRelease := range releases {
        if ghRelease.GetDraft() || ghRelease.GetPrerelease() {
            continue
        }

        release, err := c.parseRelease(ghRelease)
        if err != nil {
            continue
        }

        result = append(result, *release)
    }

    return result, nil
}
```

**Updated:** `internal/version/checker.go`

```go
func (c *Checker) Analyse(ctx context.Context, comparisonVersionStr string) (*Analysis, error) {
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

    // Validate embedded data is current
    allReleases := embeddedReleases
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

    // ... rest of existing logic
}

func (c *Checker) isEmbeddedCurrent(embedded, recent []version.Release) bool {
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

func (c *Checker) mergeReleases(embedded, recent []version.Release) []version.Release {
    // Use map to deduplicate by version
    seen := make(map[string]bool)
    var merged []version.Release

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

## Implementation Plan Summary

### Phase 1: Bootstrap Infrastructure
1. Create `data/` directory
2. Create `scripts/update-releases.sh`
3. Create `cmd/bootstrap-releases/main.go`
4. Run script to generate initial `data/releases.json`
5. Commit to repo

### Phase 2: Embedded Data Loading
1. Create `internal/data/loader.go` with `//go:embed`
2. Add `LoadEmbeddedReleases()` function
3. Test loading and parsing

### Phase 3: Update Runtime Logic
1. Add `GetRecentReleases(count int)` to GitHub client
2. Update `Checker.Analyse()` to use embedded + API strategy
3. Add `isEmbeddedCurrent()` and `mergeReleases()` helpers
4. Test with mock data

### Phase 4: Automated Updates
1. Create `cmd/check-releases/main.go`
2. Create `.github/workflows/update-releases.yml`
3. Test workflow (manual dispatch)
4. Enable daily cron

### Phase 5: Testing & Validation
1. Unit tests for merge logic
2. Integration tests for cache validation
3. Manual testing with real API
4. Verify binary size impact

## Success Criteria

- ✅ API calls reduced from 2 to 1 per invocation
- ✅ Binary remains single portable executable
- ✅ Cache updates automatically when new releases appear
- ✅ Tool stays accurate (max 5 releases behind embedded data)
- ✅ Fallback to full API query when embedded data too stale
- ✅ Rate limit friendly (1 request per check)

## Future Enhancements (Out of Scope)

1. Compression of embedded JSON (gzip)
2. Differential updates (only fetch releases after certain date)
3. Multiple cache tiers (embedded + user-local cache file)
4. Metrics on cache hit/miss rates
