package version

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
)

// benchTestRelease creates a test release for benchmarking
func benchTestRelease(versionStr string, daysAgo int) Release {
	v := semver.MustParse(versionStr)
	return Release{
		Version:     v,
		PublishedAt: time.Now().AddDate(0, 0, -daysAgo),
		URL:         fmt.Sprintf("https://github.com/actions/runner/releases/tag/v%s", versionStr),
	}
}

// mockGitHubClient is a minimal mock for benchmarking
type mockGitHubClient struct {
	allReleases []Release
}

func (m *mockGitHubClient) GetLatestRelease(ctx context.Context) (*Release, error) {
	if len(m.allReleases) > 0 {
		return &m.allReleases[0], nil
	}
	return nil, fmt.Errorf("no releases")
}

func (m *mockGitHubClient) GetAllReleases(ctx context.Context) ([]Release, error) {
	return m.allReleases, nil
}

func (m *mockGitHubClient) GetRecentReleases(ctx context.Context, count int) ([]Release, error) {
	if len(m.allReleases) <= count {
		return m.allReleases, nil
	}
	return m.allReleases[:count], nil
}

// BenchmarkAnalyse benchmarks the main Analyse function
func BenchmarkAnalyse(b *testing.B) {
	releases := make([]Release, 0, 50)
	for i := 0; i < 50; i++ {
		releases = append(releases, benchTestRelease("2.300.0", i*7))
	}

	mockClient := &mockGitHubClient{
		allReleases: releases,
	}

	checker := NewChecker(mockClient, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
	})

	ctx := context.Background()
	comparisonVersion := "2.290.0"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = checker.Analyse(ctx, comparisonVersion)
	}
}

// BenchmarkAnalyseLatestVersion benchmarks checking latest version
func BenchmarkAnalyseLatestVersion(b *testing.B) {
	releases := make([]Release, 0, 50)
	for i := 0; i < 50; i++ {
		releases = append(releases, benchTestRelease("2.300.0", i*7))
	}

	mockClient := &mockGitHubClient{
		allReleases: releases,
	}

	checker := NewChecker(mockClient, CheckerConfig{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = checker.Analyse(ctx, "")
	}
}

// BenchmarkAnalysisMarshalJSON benchmarks JSON marshalling
func BenchmarkAnalysisMarshalJSON(b *testing.B) {
	now := time.Now()
	firstNewer := now.AddDate(0, 0, -5)

	analysis := &Analysis{
		LatestVersion:         semver.MustParse("2.329.0"),
		ComparisonVersion:     semver.MustParse("2.328.0"),
		ComparisonReleasedAt:  &now,
		FirstNewerVersion:     semver.MustParse("2.329.0"),
		FirstNewerReleaseDate: &firstNewer,
		IsLatest:              false,
		IsExpired:             false,
		IsCritical:            false,
		ReleasesBehind:        1,
		DaysSinceUpdate:       5,
		Message:               "Update available",
		CriticalAgeDays:       12,
		MaxAgeDays:            30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analysis.MarshalJSON()
	}
}
