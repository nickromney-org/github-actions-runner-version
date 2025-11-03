package version

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
)

// MockGitHubClient for testing
type MockGitHubClient struct {
	LatestRelease *Release
	AllReleases   []Release
	Error         error
}

func (m *MockGitHubClient) GetLatestRelease(ctx context.Context) (*Release, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.LatestRelease, nil
}

func (m *MockGitHubClient) GetAllReleases(ctx context.Context) ([]Release, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.AllReleases, nil
}

// GetRecentReleases returns the first N mocked releases
func (m *MockGitHubClient) GetRecentReleases(ctx context.Context, count int) ([]Release, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if len(m.AllReleases) <= count {
		return m.AllReleases, nil
	}
	return m.AllReleases[:count], nil
}

func newTestRelease(version string, daysAgo int) Release {
	v := semver.MustParse(version)
	return Release{
		Version:     v,
		PublishedAt: time.Now().AddDate(0, 0, -daysAgo),
		URL:         "https://example.com",
	}
}

func TestAnalyse_LatestVersion(t *testing.T) {
	latest := newTestRelease("2.329.0", 3)

	client := &MockGitHubClient{
		LatestRelease: &latest,
		AllReleases:   []Release{latest},
	}

	checker := NewChecker(client, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
	})

	analysis, err := checker.Analyse(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analysis.LatestVersion.String() != "2.329.0" {
		t.Errorf("expected latest version 2.329.0, got %s", analysis.LatestVersion)
	}

	if analysis.ComparisonVersion != nil {
		t.Errorf("expected no comparison version, got %s", analysis.ComparisonVersion)
	}
}

func TestAnalyse_CurrentVersion(t *testing.T) {
	latest := newTestRelease("2.329.0", 3)

	client := &MockGitHubClient{
		LatestRelease: &latest,
		AllReleases:   []Release{latest},
	}

	checker := NewChecker(client, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
	})

	analysis, err := checker.Analyse(context.Background(), "2.329.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !analysis.IsLatest {
		t.Error("expected IsLatest to be true")
	}

	if analysis.Status() != StatusCurrent {
		t.Errorf("expected status Current, got %s", analysis.Status())
	}
}

func TestAnalyse_ExpiredVersion(t *testing.T) {
	latest := newTestRelease("2.329.0", 3)
	newer := newTestRelease("2.328.0", 65) // Released 65 days ago
	comparison := newTestRelease("2.327.1", 100)

	client := &MockGitHubClient{
		LatestRelease: &latest,
		AllReleases:   []Release{latest, newer, comparison},
	}

	checker := NewChecker(client, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
	})

	analysis, err := checker.Analyse(context.Background(), "2.327.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !analysis.IsExpired {
		t.Error("expected IsExpired to be true")
	}

	if analysis.Status() != StatusExpired {
		t.Errorf("expected status Expired, got %s", analysis.Status())
	}

	if analysis.ReleasesBehind != 2 {
		t.Errorf("expected 2 releases behind, got %d", analysis.ReleasesBehind)
	}

	if analysis.DaysSinceUpdate < 60 {
		t.Errorf("expected at least 60 days since update, got %d", analysis.DaysSinceUpdate)
	}
}

func TestAnalyse_CriticalVersion(t *testing.T) {
	latest := newTestRelease("2.329.0", 3)
	newer := newTestRelease("2.328.0", 20) // Released 20 days ago
	comparison := newTestRelease("2.327.1", 50)

	client := &MockGitHubClient{
		LatestRelease: &latest,
		AllReleases:   []Release{latest, newer, comparison},
	}

	checker := NewChecker(client, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
	})

	analysis, err := checker.Analyse(context.Background(), "2.327.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analysis.IsExpired {
		t.Error("expected IsExpired to be false")
	}

	if !analysis.IsCritical {
		t.Error("expected IsCritical to be true")
	}

	if analysis.Status() != StatusCritical {
		t.Errorf("expected status Critical, got %s", analysis.Status())
	}
}

func TestAnalyse_NonExistentVersion(t *testing.T) {
	latest := newTestRelease("2.329.0", 3)
	older := newTestRelease("2.328.0", 20)

	client := &MockGitHubClient{
		LatestRelease: &latest,
		AllReleases:   []Release{latest, older},
	}

	checker := NewChecker(client, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
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

func TestAnalyse_PopulatesRecentReleases(t *testing.T) {
	releases := []Release{
		newTestRelease("2.329.0", 5),
		newTestRelease("2.328.0", 25),
		newTestRelease("2.327.1", 50),
		newTestRelease("2.327.0", 80),
	}

	client := &MockGitHubClient{
		LatestRelease: &releases[0],
		AllReleases:   releases,
	}

	checker := NewChecker(client, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
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

func TestCalculateRecentReleases_Last90Days(t *testing.T) {
	// Create releases spanning 120 days
	releases := []Release{
		newTestRelease("2.329.0", 5),   // 5 days ago
		newTestRelease("2.328.0", 25),  // 25 days ago
		newTestRelease("2.327.1", 50),  // 50 days ago
		newTestRelease("2.327.0", 80),  // 80 days ago
		newTestRelease("2.326.0", 100), // 100 days ago - outside window
	}

	comparisonVersion := semver.MustParse("2.327.0")
	latestVersion := semver.MustParse("2.329.0")

	checker := &Checker{}
	recent := checker.CalculateRecentReleases(releases, comparisonVersion, latestVersion)

	// Should include releases from last 90 days: 2.329.0, 2.328.0, 2.327.1, 2.327.0
	if len(recent) != 4 {
		t.Errorf("expected 4 releases, got %d", len(recent))
	}
}

func TestCalculateRecentReleases_Minimum4(t *testing.T) {
	// Only 2 releases in last 90 days, but should return minimum 4
	releases := []Release{
		newTestRelease("2.329.0", 5),   // 5 days ago
		newTestRelease("2.328.0", 25),  // 25 days ago
		newTestRelease("2.327.0", 100), // 100 days ago
		newTestRelease("2.326.0", 120), // 120 days ago
	}

	comparisonVersion := semver.MustParse("2.327.0")
	latestVersion := semver.MustParse("2.329.0")

	checker := &Checker{}
	recent := checker.CalculateRecentReleases(releases, comparisonVersion, latestVersion)

	if len(recent) != 4 {
		t.Errorf("expected minimum 4 releases, got %d", len(recent))
	}
}

func TestFindNewerReleases(t *testing.T) {
	releases := []Release{
		newTestRelease("2.329.0", 3),
		newTestRelease("2.328.0", 65),
		newTestRelease("2.327.1", 84),
		newTestRelease("2.327.0", 90),
	}

	client := &MockGitHubClient{}
	checker := NewChecker(client, CheckerConfig{})

	comparisonVersion := semver.MustParse("2.327.0")
	newer := checker.findNewerReleases(releases, comparisonVersion)

	if len(newer) != 3 {
		t.Errorf("expected 3 newer releases, got %d", len(newer))
	}

	// Check they're sorted oldest-first
	if newer[0].Version.String() != "2.327.1" {
		t.Errorf("expected first newer release to be 2.327.1, got %s", newer[0].Version)
	}
}

func TestDaysBetween(t *testing.T) {
	start := time.Date(2024, 8, 13, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 10, 17, 0, 0, 0, 0, time.UTC)

	days := daysBetween(start, end)

	expected := 65
	if days != expected {
		t.Errorf("expected %d days, got %d", expected, days)
	}
}

func TestCheckerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CheckerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  CheckerConfig{CriticalAgeDays: 12, MaxAgeDays: 30},
			wantErr: false,
		},
		{
			name:    "critical >= max",
			config:  CheckerConfig{CriticalAgeDays: 30, MaxAgeDays: 30},
			wantErr: true,
		},
		{
			name:    "negative critical",
			config:  CheckerConfig{CriticalAgeDays: -1, MaxAgeDays: 30},
			wantErr: true,
		},
		{
			name:    "version-based policy (both zero)",
			config:  CheckerConfig{CriticalAgeDays: 0, MaxAgeDays: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
