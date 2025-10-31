package version

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
)

// GitHubClient defines the interface for fetching releases
type GitHubClient interface {
	GetLatestRelease(ctx context.Context) (*Release, error)
	GetAllReleases(ctx context.Context) ([]Release, error)
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

// Analyze performs the version analysis
func (c *Checker) Analyze(ctx context.Context, comparisonVersionStr string) (*Analysis, error) {
	// Validate config
	if err := c.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Fetch latest release
	latestRelease, err := c.client.GetLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
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

	// Fetch all releases to find newer versions
	allReleases, err := c.client.GetAllReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all releases: %w", err)
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
