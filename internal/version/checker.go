package version

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nickromney-org/github-actions-runner-version/internal/data"
)

// GitHubClient defines the interface for fetching releases
type GitHubClient interface {
	GetLatestRelease(ctx context.Context) (*Release, error)
	GetAllReleases(ctx context.Context) ([]Release, error)
	GetRecentReleases(ctx context.Context, count int) ([]Release, error)
}

// Checker performs version analysis
type Checker struct {
	client GitHubClient
	config CheckerConfig
}

// NewChecker creates a new version checker
func NewChecker(client GitHubClient, config CheckerConfig) *Checker {
	return &Checker{
		client: client,
		config: config,
	}
}

// versionExists checks if a version exists in the releases list
func (c *Checker) versionExists(releases []Release, version *semver.Version) bool {
	for _, release := range releases {
		if release.Version.Equal(version) {
			return true
		}
	}
	return false
}

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

// Analyse performs the version analysis
func (c *Checker) Analyse(ctx context.Context, comparisonVersionStr string) (*Analysis, error) {
	// Validate config
	if err := c.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Load embedded releases
	embeddedData, err := data.LoadEmbeddedReleases()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded releases: %w", err)
	}

	// Convert data.Release to version.Release
	embeddedReleases := make([]Release, len(embeddedData))
	for i, r := range embeddedData {
		embeddedReleases[i] = Release{
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

		// Determine status
		analysis.IsExpired = analysis.DaysSinceUpdate >= c.config.MaxAgeDays
		analysis.IsCritical = !analysis.IsExpired && analysis.DaysSinceUpdate >= c.config.CriticalAgeDays
	}

	// Generate message
	analysis.Message = c.generateMessage(analysis)

	return analysis, nil
}

// CalculateRecentReleases returns releases for the expiry timeline table
// Shows all releases from last 90 days, or minimum 4 releases
func (c *Checker) CalculateRecentReleases(allReleases []Release, comparisonVersion *semver.Version, latestVersion *semver.Version) []ReleaseExpiry {
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
			Version:    release.Version,
			ReleasedAt: release.PublishedAt,
			IsLatest:   release.Version.Equal(latestVersion),
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

// findNewerReleases returns releases newer than the comparison version, sorted oldest-first
func (c *Checker) findNewerReleases(releases []Release, comparisonVersion *semver.Version) []Release {
	var newer []Release

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

	issues := []string{}

	// Count releases
	if analysis.ReleasesBehind > 0 {
		plural := ""
		if analysis.ReleasesBehind > 1 {
			plural = "s"
		}
		issues = append(issues, fmt.Sprintf("%d release%s behind", analysis.ReleasesBehind, plural))
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

// daysBetween calculates the number of days between two dates
func daysBetween(start, end time.Time) int {
	duration := end.Sub(start)
	return int(duration.Hours() / 24)
}

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
