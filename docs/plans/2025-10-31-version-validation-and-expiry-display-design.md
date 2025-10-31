# Version Validation and Expiry Display Improvements

**Date:** 31 October 2025
**Status:** Approved
**Author:** Design Session with User

## Problem Statement

The current implementation has two critical UX issues:

1. **No version validation** - Tool accepts non-existent versions (e.g., `2.327.99`) and processes them, leading to misleading output
2. **Confusing recommendations** - Shows "Update available: v2.328.0" when v2.329.0 (latest) exists, causing users to question why they should update to an older version

Additionally, users need visibility into the release cadence and expiry timeline to plan maintenance windows effectively.

## Current Behaviour

```bash
$ ./github-actions-runner-version -c 2.327.99
2.329.0

🚨 Version 2.327.99 EXPIRED: 2 releases behind AND 48 days overdue

   📦 Update available: v2.328.0
      Released: Aug 13, 2025 (78 days ago)
   🎯 Latest version: v2.329.0
   ⚠️  2 releases behind
```

**Issues:**
- Accepts fictional version 2.327.99 without error
- Recommends updating to 2.328.0 instead of 2.329.0 (latest)
- Lacks context on release patterns and expiry timeline

## Design Goals

1. **Validate versions** - Reject non-existent versions with helpful error messages
2. **Clear recommendations** - Always recommend latest version as primary action
3. **Transparency** - Show how policy status is calculated (from first newer release)
4. **Planning context** - Display recent release history and expiry timeline
5. **British English** - Consistent spelling throughout (analyse, colour)

## GitHub Actions Runner Policy Context

From GitHub's documentation:

> Any updates released for the software, including major, minor, or patch releases, are considered as an available update. If you do not perform a software update within 30 days, the GitHub Actions service will not queue jobs to your runner.

**Key policy mechanics:**
- Monthly release cadence (with ad-hoc security patches)
- 30-day expiry window starts from NEXT release, not current release
- Latest version has no expiry date
- Version 2.328.1 with 2.329.0 released on 14 Oct expires on 13 Nov (30 days after 14 Oct)

## Proposed Solution

### 1. Version Validation

**Implementation:**
```go
// After fetching all releases, validate comparison version exists
func (c *Checker) versionExists(releases []Release, version *semver.Version) bool {
    for _, release := range releases {
        if release.Version.Equal(version) {
            return true
        }
    }
    return false
}

// In Analyse(), after parsing comparisonVersion
if !c.versionExists(allReleases, comparisonVersion) {
    return nil, fmt.Errorf("version %s does not exist in GitHub releases (latest: %s)",
        comparisonVersion, latestRelease.Version)
}
```

**Error output:**
```bash
$ ./github-actions-runner-version -c 2.327.99
Error: version 2.327.99 does not exist in GitHub releases (latest: 2.329.0)
```

### 2. Clear Status Messages

**New single-line format with all key dates:**

```bash
# Expired
🚨 Version 2.327.1 (25 Jul 2024) EXPIRED 24 Aug 2024: Update to v2.329.0 (Released 14 Oct 2024)

# Critical (approaching expiry)
🔶 Version 2.328.1 (10 Sep 2024) EXPIRES 13 Nov 2024 (28 days): Update to v2.329.0 (Released 14 Oct 2024)

# Warning (behind but not critical)
⚠️  Version 2.328.0 (13 Aug 2024) expires 9 Sep 2024: Update to v2.329.0 (Released 14 Oct 2024)

# Current (on latest)
✅ Version 2.329.0 (14 Oct 2024) is the latest version
```

**Format pattern:**
- `[Icon] Version X.Y.Z ([release date]) [STATUS] [expiry date]: Update to vA.B.C (Released [date])`
- All dates in UK format: "25 Jul 2024"
- Expiry shows actual calendar date for planning
- Critical status includes countdown for urgency

### 3. Release Expiry Timeline Table

**Display by default** (suppress with `-q, --quiet` flag):

```
📅 Release Expiry Timeline
─────────────────────────────────────────────────────
Version    Released       Expires        Status
2.327.1    25 Jul 2024    24 Aug 2024    ❌ Expired 77 days ago
2.328.0    13 Aug 2024    9 Sep 2024     ❌ Expired 47 days ago
2.328.1    10 Sep 2024    13 Nov 2024    ✅ Valid (28 days left)  ← Your version
2.329.0    14 Oct 2024    -              ✅ Latest (2 days ago)
```

**Release selection logic:**
- Show all releases from last 90 days
- Minimum 4 releases (even if older than 90 days)
- Always include user's version (marked with arrow)
- Always include latest version
- Sort: oldest first (latest at bottom)

**Rationale:**
- 90 days captures ~3 monthly release cycles
- Shows actual release patterns (not just stated "monthly" cadence)
- Helps users plan maintenance windows
- Minimum 4 ensures context even with sparse releases

### 4. Data Structure Changes

**Add to `internal/version/types.go`:**

```go
// ReleaseExpiry represents expiry information for a single release
type ReleaseExpiry struct {
    Version         *semver.Version
    ReleasedAt      time.Time
    ExpiresAt       *time.Time  // nil for latest version
    DaysUntilExpiry int         // negative if expired
    IsExpired       bool
    IsLatest        bool
}

// Analysis - add new field
type Analysis struct {
    // ... existing fields ...
    ComparisonVersionReleasedAt *time.Time      // When comparison version was released
    RecentReleases              []ReleaseExpiry // Last 90 days or min 4 releases
}
```

### 5. Quiet Flag

Add `-q, --quiet` flag following industry standards (git, npm, docker):

**Behaviour:**
- Suppresses expiry table
- Shows only: latest version + status message
- Useful for scripting/CI where full output is noise
- JSON/CI modes ignore this flag (they have their own formats)

**Example:**
```bash
$ ./github-actions-runner-version -c 2.328.1 -q
2.329.0
⚠️  Version 2.328.1 (10 Sep 2024) expires 13 Nov 2024 (28 days): Update to v2.329.0 (Released 14 Oct 2024)
```

### 6. JSON Output Updates

**Add expiry timeline to JSON:**

```json
{
  "latest_version": "2.329.0",
  "comparison_version": "2.327.1",
  "comparison_version_released": "2024-07-25T00:00:00Z",
  "expired_on": "2024-08-24T00:00:00Z",
  "is_expired": true,
  "status": "expired",
  "recent_releases": [
    {
      "version": "2.327.1",
      "released": "2024-07-25T00:00:00Z",
      "expires": "2024-08-24T00:00:00Z",
      "is_expired": true,
      "is_latest": false,
      "days_until_expiry": -77
    },
    {
      "version": "2.329.0",
      "released": "2024-10-14T00:00:00Z",
      "expires": null,
      "is_expired": false,
      "is_latest": true,
      "days_until_expiry": null
    }
  ]
}
```

Allows automation to process full expiry timeline for monitoring/alerting.

### 7. CI Output Updates

**GitHub Actions workflow commands:**

```bash
::error title=Runner Version Expired::🚨 Version 2.327.1 (25 Jul 2024) EXPIRED 24 Aug 2024: Update to v2.329.0 (Released 14 Oct 2024)

::group::📅 Release Expiry Timeline
[Same table format as terminal output]
::endgroup::
```

**Job summary:** Include table in `$GITHUB_STEP_SUMMARY` for visibility in Actions UI.

### 8. British English Throughout

Apply British English spelling consistently:

**Code changes:**
- `Analyze()` → `Analyse()`
- `import color` → `import colour` (aliased from `github.com/fatih/color`)
- `getStatusColor()` → `getStatusColour()`
- All test names: `TestAnalyze_*` → `TestAnalyse_*`

**Documentation/messages:**
- "colorized" → "colourised"
- "analyze" → "analyse"
- "summarize" → "summarise"
- "initialize" → "initialise"

**Linting configuration** (`.golangci.yml`):

```yaml
linters:
  enable:
    - misspell

linters-settings:
  misspell:
    locale: UK
    ignore-words:
      - color  # Keep for imported package names
```

## Implementation Plan

### Files to Modify

1. **`internal/version/types.go`**
   - Add `ReleaseExpiry` struct
   - Add fields to `Analysis`: `RecentReleases`, `ComparisonVersionReleasedAt`
   - Update JSON marshaling for new fields

2. **`internal/version/checker.go`**
   - Rename: `Analyze()` → `Analyse()`
   - Add `versionExists()` method
   - Add `calculateRecentReleases()` method (90-day/min 4 logic)
   - Add `calculateExpiry()` method (expiry = next_release + 30 days)
   - Update error message for non-existent versions
   - British English in comments/messages

3. **`cmd/root.go`**
   - Add `-q, --quiet` flag
   - Import alias: `colour` from `github.com/fatih/color`
   - Rewrite `printStatus()` with new single-line format
   - Add `printExpiryTable()` function (UK date format: "25 Jul 2024")
   - Rename: `getStatusColor()` → `getStatusColour()`
   - Update JSON marshaling
   - Update CI output with new format
   - British English in help text, messages

4. **`internal/github/client.go`**
   - British English in comments

5. **`README.md`**
   - Update examples with new output format
   - British English spelling ("colourised", etc.)
   - Document `-q, --quiet` flag

6. **`CLAUDE.md`**
   - Update method names (`Analyse`)
   - British English spelling

7. **`.golangci.yml`** (new file)
   - Configure misspell linter for UK locale

8. **`Makefile`**
   - Already has lint target, no changes needed

### Test Coverage

**`internal/version/checker_test.go`:**

1. **Version Validation:**
   - `TestVersionExists_ValidVersion`
   - `TestVersionExists_InvalidVersion`
   - `TestAnalyse_NonExistentVersion` - Error with helpful message

2. **Recent Releases Calculation:**
   - `TestCalculateRecentReleases_Last90Days` - 10 releases in 90 days
   - `TestCalculateRecentReleases_Minimum4` - 2 releases in 90 days, returns 4
   - `TestCalculateRecentReleases_IncludesUserVersion` - User on 100-day-old version
   - `TestCalculateRecentReleases_AlwaysIncludesLatest`

3. **Expiry Date Calculation:**
   - `TestCalculateExpiry_MiddleRelease` - expiry = next_release + 30 days
   - `TestCalculateExpiry_LatestRelease` - No expiry (nil)
   - `TestCalculateExpiry_DaysUntilExpiry` - Positive/negative/zero

4. **Integration:**
   - `TestAnalyse_ExpiredWithExpiryTable` - Full analysis includes RecentReleases
   - `TestAnalyse_ValidatesBeforeProcessing`

5. **Edge Cases:**
   - `TestAnalyse_OnlyOneRelease`
   - `TestAnalyse_VeryOldVersion` - 200 days old

**`cmd/root_test.go`:** (new file)

6. **Date Formatting:**
   - `TestFormatUKDate` - "25 Jul 2024" format
   - `TestPrintStatus_WithAllDates` - Status line format
   - `TestPrintExpiryTable_UKDates`

**Coverage target:** 80%+ on new code

## Breaking Changes

**Public API changes** (acceptable for new project):
- `Analyze()` → `Analyse()` - Method name change
- `Analysis` struct has new fields (additive, backwards compatible for JSON consumers)

**Output format changes:**
- Status message format completely rewritten
- New table added to default output
- `-q` flag required to suppress table

Since project is new with only README.md in public repo, breaking changes are acceptable now.

## Future Enhancements (Out of Scope)

1. **Security release detection** - Parse release notes for CVE mentions, highlight in table
2. **Custom expiry windows** - Allow users to set their own warning thresholds
3. **Comparison against multiple versions** - Check entire fleet at once
4. **Historical trend analysis** - Track release cadence over time

## Success Criteria

- ✅ Non-existent versions rejected with clear error
- ✅ Status message shows all relevant dates in one line
- ✅ Expiry table displays by default, shows 90-day window
- ✅ `-q` flag suppresses table for scripting
- ✅ JSON output includes full expiry timeline
- ✅ All tests pass with 80%+ coverage
- ✅ British English throughout project
- ✅ `make lint` passes with UK locale
