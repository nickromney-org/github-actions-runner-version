package github

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	gh "github.com/google/go-github/v57/github"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client
type Client struct {
	gh    *gh.Client
	Owner string
	Repo  string
}

// NewClient creates a new GitHub API client
func NewClient(token, owner, repo string) *Client {
	var client *gh.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(context.Background(), ts)
		client = gh.NewClient(tc)
	} else {
		client = gh.NewClient(nil)
	}

	return &Client{
		gh:    client,
		Owner: owner,
		Repo:  repo,
	}
}

// GetLatestRelease fetches the latest release from GitHub
func (c *Client) GetLatestRelease(ctx context.Context) (*version.Release, error) {
	release, _, err := c.gh.Repositories.GetLatestRelease(ctx, c.Owner, c.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	return c.parseRelease(release)
}

// GetAllReleases fetches all releases from GitHub
func (c *Client) GetAllReleases(ctx context.Context) ([]version.Release, error) {
	var allReleases []version.Release

	opts := &gh.ListOptions{PerPage: 100}

	for page := 1; page <= 10; page++ { // Safety limit of 10 pages
		opts.Page = page

		releases, resp, err := c.gh.Repositories.ListReleases(ctx, c.Owner, c.Repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases (page %d): %w", page, err)
		}

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

			allReleases = append(allReleases, *release)
		}

		// Check if we've reached the last page
		if resp.NextPage == 0 {
			break
		}
	}

	return allReleases, nil
}

// GetRecentReleases fetches only the N most recent releases
func (c *Client) GetRecentReleases(ctx context.Context, count int) ([]version.Release, error) {
	opts := &gh.ListOptions{PerPage: count}

	releases, _, err := c.gh.Repositories.ListReleases(ctx, c.Owner, c.Repo, opts)
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

// parseRelease converts a GitHub release to our Release type
func (c *Client) parseRelease(ghRelease *gh.RepositoryRelease) (*version.Release, error) {
	tagName := ghRelease.GetTagName()
	if tagName == "" {
		return nil, fmt.Errorf("release has no tag name")
	}

	// Parse version (removing 'v' prefix if present)
	ver, err := semver.NewVersion(tagName)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q: %w", tagName, err)
	}

	// Parse published date
	publishedAt := ghRelease.GetPublishedAt()
	if publishedAt.IsZero() {
		return nil, fmt.Errorf("release has no published date")
	}

	return &version.Release{
		Version:     ver,
		PublishedAt: publishedAt.Time,
		URL:         ghRelease.GetHTMLURL(),
	}, nil
}

// MockClient is a mock implementation for testing
type MockClient struct {
	LatestRelease *version.Release
	AllReleases   []version.Release
	Error         error
}

// GetLatestRelease returns the mocked latest release
func (m *MockClient) GetLatestRelease(ctx context.Context) (*version.Release, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.LatestRelease, nil
}

// GetAllReleases returns the mocked releases
func (m *MockClient) GetAllReleases(ctx context.Context) ([]version.Release, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.AllReleases, nil
}

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

// Helper for creating test releases
func NewTestRelease(versionStr, owner, repo string, daysAgo int) version.Release {
	v := semver.MustParse(versionStr)
	return version.Release{
		Version:     v,
		PublishedAt: time.Now().AddDate(0, 0, -daysAgo),
		URL:         fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s", owner, repo, versionStr),
	}
}
