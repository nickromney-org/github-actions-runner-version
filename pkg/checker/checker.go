package checker

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nickromney-org/github-release-version-checker/internal/data"
	"github.com/nickromney-org/github-release-version-checker/pkg/policy"
	"github.com/nickromney-org/github-release-version-checker/pkg/types"
)

// GitHubClient defines the interface for fetching releases
type GitHubClient interface {
	GetLatestRelease(ctx context.Context) (*types.Release, error)
	GetAllReleases(ctx context.Context) ([]types.Release, error)
	GetRecentReleases(ctx context.Context, count int) ([]types.Release, error)
}

// Checker performs version analysis
type Checker struct {
	client GitHubClient
	config Config
	policy policy.VersionPolicy // Optional: if set, overrides config-based logic
}

// NewChecker creates a new version checker
func NewChecker(client GitHubClient, config Config) *Checker {
	return &Checker{
		client: client,
		config: config,
		policy: nil,
	}
}

// NewCheckerWithPolicy creates a new version checker with a custom policy
func NewCheckerWithPolicy(client GitHubClient, config Config, pol policy.VersionPolicy) *Checker {
	return &Checker{
		client: client,
		config: config,
		policy: pol,
	}
}

// versionExists checks if a version exists in the releases list
func (c *Checker) versionExists(releases []types.Release, version *semver.Version) bool {
	for _, release := range releases {
		if release.Version.Equal(version) {
			return true
		}
	}
	return false
}

// mergeReleases combines embedded and recent releases, deduplicating by version
func (c *Checker) mergeReleases(embedded, recent []types.Release) []types.Release {
	// Use map to deduplicate by version
	seen := make(map[string]bool)
	var merged []types.Release

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

// Analyse performs the version analysis
func (c *Checker) Analyse(ctx context.Context, comparisonVersionStr string) (*Analysis, error) {
	// Validate config
	if err := c.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Determine which dataset to use
	var allReleases []types.Release
	var err error

	if c.config.NoCache {
		// Bypass embedded cache - fetch all releases from API
		allReleases, err = c.client.GetAllReleases(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch all releases: %w", err)
		}
	} else {
		// Use embedded cache with validation
		// Load embedded releases
		embeddedData, err := data.LoadEmbeddedReleases()
		if err != nil {
			return nil, fmt.Errorf("failed to load embedded releases: %w", err)
		}

		// Convert data.Release to types.Release
		embeddedReleases := make([]types.Release, len(embeddedData))
		for i, r := range embeddedData {
			embeddedReleases[i] = types.Release{
				Version:     r.Version,
				PublishedAt: r.PublishedAt,
				URL:         r.URL,
			}
		}

		// Fetch 5 most recent releases from API
		recentReleases, err := c.client.GetRecentReleases(ctx, 5)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch recent releases: %w", err)
		}

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
	}

	// Ensure we have releases
	if len(allReleases) == 0 {
		return nil, fmt.Errorf("no releases available")
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

	// Parse comparison version
	comparisonVersion, err := semver.NewVersion(comparisonVersionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid comparison version %q: %w", comparisonVersionStr, err)
	}

	// Check if already on latest
	if comparisonVersion.Equal(latestRelease.Version) {
		return &Analysis{
			LatestVersion:     latestRelease.Version,
			ComparisonVersion: comparisonVersion,
			IsLatest:          true,
			CriticalAgeDays:   c.config.CriticalAgeDays,
			MaxAgeDays:        c.config.MaxAgeDays,
			Message:           fmt.Sprintf("✅ Version %s is up to date", comparisonVersion),
		}, nil
	}

	// Validate version exists
	if !c.versionExists(allReleases, comparisonVersion) {
		return nil, fmt.Errorf("version %s does not exist in GitHub releases (latest: %s)",
			comparisonVersion, latestRelease.Version)
	}

	// Find releases newer than comparison version
	newerReleases := c.findNewerReleases(allReleases, comparisonVersion)

	// Build analysis
	analysis := &Analysis{
		LatestVersion:     latestRelease.Version,
		ComparisonVersion: comparisonVersion,
		IsLatest:          false,
		ReleasesBehind:    len(newerReleases),
		NewerReleases:     newerReleases,
		CriticalAgeDays:   c.config.CriticalAgeDays,
		MaxAgeDays:        c.config.MaxAgeDays,
	}

	// Calculate recent releases for timeline table
	analysis.RecentReleases = c.CalculateRecentReleases(allReleases, comparisonVersion, latestRelease.Version)

	// Find comparison version release date
	for _, release := range allReleases {
		if release.Version.Equal(comparisonVersion) {
			analysis.ComparisonReleasedAt = &release.PublishedAt
			break
		}
	}

	// Calculate age from first newer release
	if len(newerReleases) > 0 {
		firstNewer := newerReleases[0]
		analysis.FirstNewerVersion = firstNewer.Version
		analysis.FirstNewerReleaseDate = &firstNewer.PublishedAt
		analysis.DaysSinceUpdate = daysBetween(firstNewer.PublishedAt, time.Now())

		// Determine status using policy if available, otherwise use config
		if c.policy != nil {
			// Use policy system
			comparisonDate := time.Now()
			if analysis.ComparisonReleasedAt != nil {
				comparisonDate = *analysis.ComparisonReleasedAt
			}

			policyResult := c.policy.Evaluate(
				comparisonVersion,
				comparisonDate,
				latestRelease.Version,
				latestRelease.PublishedAt,
				newerReleases,
			)

			analysis.IsExpired = policyResult.IsExpired
			analysis.IsCritical = policyResult.IsCritical
			analysis.PolicyType = c.policy.Type()
			analysis.MinorVersionsBehind = policyResult.VersionsBehind
		} else {
			// Use legacy config-based logic
			analysis.IsExpired = analysis.DaysSinceUpdate >= c.config.MaxAgeDays
			analysis.IsCritical = !analysis.IsExpired && analysis.DaysSinceUpdate >= c.config.CriticalAgeDays
			analysis.PolicyType = "days"
		}
	}

	// Generate message
	analysis.Message = c.generateMessage(analysis)

	return analysis, nil
}

// CalculateRecentReleases returns releases for the expiry timeline table
// Shows all releases from last 90 days, or minimum 4 releases
func (c *Checker) CalculateRecentReleases(allReleases []types.Release, comparisonVersion *semver.Version, latestVersion *semver.Version) []ReleaseExpiry {
	now := time.Now()

	// For version-based policies, show recent minor versions instead of time-based window
	isVersionPolicy := c.policy != nil && c.policy.Type() == "versions"

	var recentReleases []types.Release

	if isVersionPolicy {
		// For version-based policies, show unique minor versions from comparison to latest
		majorVersion := latestVersion.Major()

		// Sort all releases by version (newest first)
		sorted := make([]types.Release, len(allReleases))
		copy(sorted, allReleases)
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i].Version.LessThan(sorted[j].Version) {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		// Get max versions behind from policy
		maxVersionsBehind := 3 // Default
		if c.policy != nil {
			maxVersionsBehind = c.policy.GetMaxVersionsBehind()
			if maxVersionsBehind == 0 {
				maxVersionsBehind = 3
			}
		}

		// Calculate the supported window: latest - maxVersionsBehind to latest
		minSupportedMinor := int(latestVersion.Minor()) - maxVersionsBehind
		if minSupportedMinor < 0 {
			minSupportedMinor = 0
		}

		// Collect releases to show:
		// For each minor version, collect first and latest patch releases
		minorReleases := make(map[string][]types.Release)

		for _, release := range sorted {
			if release.Version.Major() == majorVersion {
				minorKey := fmt.Sprintf("%d.%d", release.Version.Major(), release.Version.Minor())
				minorReleases[minorKey] = append(minorReleases[minorKey], release)
			}
		}

		// Now for each minor version, add first patch, latest patch, and user's version if different
		for _, releases := range minorReleases {
			if len(releases) == 0 {
				continue
			}

			minor := releases[0].Version.Minor()
			isInSupportedWindow := int(minor) >= minSupportedMinor
			isUserMinor := releases[0].Version.Major() == comparisonVersion.Major() &&
				releases[0].Version.Minor() == comparisonVersion.Minor()

			// Skip if not in supported window and not the user's version
			if !isInSupportedWindow && !isUserMinor {
				continue
			}

			// Sort releases by version (highest to lowest)
			for i := 0; i < len(releases)-1; i++ {
				for j := i + 1; j < len(releases); j++ {
					if releases[i].Version.LessThan(releases[j].Version) {
						releases[i], releases[j] = releases[j], releases[i]
					}
				}
			}

			latest := releases[0]
			first := releases[len(releases)-1]

			// Add first patch release
			recentReleases = append(recentReleases, first)

			// Add latest patch if different from first
			if !latest.Version.Equal(first.Version) {
				recentReleases = append(recentReleases, latest)
			}

			// Add user's version if it's different from both first and latest
			if isUserMinor {
				if !comparisonVersion.Equal(first.Version) && !comparisonVersion.Equal(latest.Version) {
					for _, r := range releases {
						if r.Version.Equal(comparisonVersion) {
							recentReleases = append(recentReleases, r)
							break
						}
					}
				}
			}
		}
	} else {
		// For days-based policies, use 90-day window
		ninetyDaysAgo := now.AddDate(0, 0, -90)

		// Collect releases from last 90 days
		for _, release := range allReleases {
			if release.PublishedAt.After(ninetyDaysAgo) {
				recentReleases = append(recentReleases, release)
			}
		}

		// Ensure minimum 4 releases
		if len(recentReleases) < 4 && len(allReleases) >= 4 {
			// Sort all releases by date (newest first)
			sorted := make([]types.Release, len(allReleases))
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
	}

	// Sort for display
	if isVersionPolicy {
		// For version-based policies, sort by version number (oldest first)
		for i := 0; i < len(recentReleases)-1; i++ {
			for j := i + 1; j < len(recentReleases); j++ {
				if recentReleases[i].Version.GreaterThan(recentReleases[j].Version) {
					recentReleases[i], recentReleases[j] = recentReleases[j], recentReleases[i]
				}
			}
		}
	} else {
		// For days-based policies, sort by date (oldest first)
		for i := 0; i < len(recentReleases)-1; i++ {
			for j := i + 1; j < len(recentReleases); j++ {
				if recentReleases[i].PublishedAt.After(recentReleases[j].PublishedAt) {
					recentReleases[i], recentReleases[j] = recentReleases[j], recentReleases[i]
				}
			}
		}
	}

	// Convert to ReleaseExpiry
	var result []ReleaseExpiry
	for i, release := range recentReleases {
		expiry := ReleaseExpiry{
			Version:    release.Version,
			ReleasedAt: release.PublishedAt,
			IsLatest:   release.Version.Equal(latestVersion),
		}

		if isVersionPolicy {
			// For version-based policies, don't calculate time-based expiry
			expiry.ExpiresAt = nil
			expiry.DaysUntilExpiry = 0
			expiry.IsExpired = false
		} else {
			// For days-based policies, calculate expiry (30 days after next release)
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
		}

		result = append(result, expiry)
	}

	return result
}

// findNewerReleases returns releases newer than the comparison version, sorted oldest-first
func (c *Checker) findNewerReleases(releases []types.Release, comparisonVersion *semver.Version) []types.Release {
	var newer []types.Release

	for _, release := range releases {
		if release.Version.GreaterThan(comparisonVersion) {
			newer = append(newer, release)
		}
	}

	// Sort by published date (oldest first) - this gives us the first update
	for i := 0; i < len(newer)-1; i++ {
		for j := i + 1; j < len(newer); j++ {
			if newer[i].PublishedAt.After(newer[j].PublishedAt) {
				newer[i], newer[j] = newer[j], newer[i]
			}
		}
	}

	return newer
}

// generateMessage creates a human-readable status message
func (c *Checker) generateMessage(analysis *Analysis) string {
	if analysis.IsLatest {
		return fmt.Sprintf("Version %s is up to date", analysis.ComparisonVersion)
	}

	// Handle version-based policies
	if analysis.PolicyType == "versions" {
		var prefix string
		if analysis.IsExpired {
			prefix = "UNSUPPORTED"
		} else if analysis.IsCritical {
			prefix = "CRITICAL"
		} else {
			prefix = "Warning"
		}

		// Get max allowed from policy
		maxAllowed := 3 // Default fallback
		if c.policy != nil {
			maxAllowed = c.policy.GetMaxVersionsBehind()
			if maxAllowed == 0 {
				maxAllowed = 3 // Fallback
			}
		}

		msg := fmt.Sprintf("Version %s %s: %d minor version%s behind",
			analysis.ComparisonVersion,
			prefix,
			analysis.MinorVersionsBehind,
			pluralSuffix(analysis.MinorVersionsBehind))

		if analysis.IsExpired {
			msg += fmt.Sprintf(" (maximum %d allowed)", maxAllowed)
		} else if analysis.IsCritical {
			msg += fmt.Sprintf(" (at maximum %d allowed)", maxAllowed)
		}

		return msg
	}

	// Handle days-based policies
	issues := []string{}

	// Count releases
	if analysis.ReleasesBehind > 0 {
		issues = append(issues, fmt.Sprintf("%d release%s behind", analysis.ReleasesBehind, pluralSuffix(analysis.ReleasesBehind)))
	}

	// Age status
	if analysis.IsExpired {
		daysOver := analysis.DaysSinceUpdate - c.config.MaxAgeDays
		issues = append(issues, fmt.Sprintf("%d days overdue", daysOver))
	} else if analysis.IsCritical {
		daysLeft := c.config.MaxAgeDays - analysis.DaysSinceUpdate
		issues = append(issues, fmt.Sprintf("expires in %d days", daysLeft))
	}

	issueStr := ""
	if len(issues) > 0 {
		issueStr = issues[0]
		for i := 1; i < len(issues); i++ {
			issueStr += " AND " + issues[i]
		}
	}

	var prefix string
	if analysis.IsExpired {
		prefix = "EXPIRED"
	} else if analysis.IsCritical {
		prefix = "CRITICAL"
	} else {
		prefix = "Warning"
	}

	return fmt.Sprintf("Version %s %s: %s", analysis.ComparisonVersion, prefix, issueStr)
}

// pluralSuffix returns "s" if count != 1, otherwise ""
func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// daysBetween calculates the number of days between two dates
func daysBetween(start, end time.Time) int {
	duration := end.Sub(start)
	return int(duration.Hours() / 24)
}

// FindLatestRelease finds the release with the highest version number
func FindLatestRelease(releases []types.Release) *types.Release {
	if len(releases) == 0 {
		return nil
	}

	latest := &releases[0]
	for i := range releases {
		if releases[i].Version.GreaterThan(latest.Version) {
			latest = &releases[i]
		}
	}

	return latest
}

// isEmbeddedCurrent checks if embedded data contains the latest release
// by verifying the latest embedded version is in the recent 5 releases
func (c *Checker) isEmbeddedCurrent(embedded, recent []types.Release) bool {
	if len(embedded) == 0 || len(recent) == 0 {
		return false
	}

	// Find latest embedded release
	latestEmbedded := FindLatestRelease(embedded)
	if latestEmbedded == nil {
		return false
	}

	// Check if it exists in recent 5
	for _, r := range recent {
		if r.Version.Equal(latestEmbedded.Version) {
			return true
		}
	}

	return false
}
