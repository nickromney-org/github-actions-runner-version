package cache

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nickromney-org/github-actions-runner-version/internal/config"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
)

//go:embed data/*.json
var embeddedCaches embed.FS

// CacheData represents the structure of a cache file
type CacheData struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Repository  string         `json:"repository,omitempty"`
	Releases    []jsonRelease `json:"releases"`
}

// jsonRelease is the JSON representation of a release
type jsonRelease struct {
	Version     string    `json:"version"`
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url"`
}

// toVersionRelease converts jsonRelease to version.Release
func (jr *jsonRelease) toVersionRelease() (version.Release, error) {
	ver, err := semver.NewVersion(jr.Version)
	if err != nil {
		return version.Release{}, fmt.Errorf("invalid version %q: %w", jr.Version, err)
	}

	return version.Release{
		Version:     ver,
		PublishedAt: jr.PublishedAt,
		URL:         jr.URL,
	}, nil
}

// Manager handles multiple embedded and custom caches
type Manager struct {
	customCachePath string // Optional custom cache file
}

// NewManager creates a new cache manager
func NewManager(customPath string) *Manager {
	return &Manager{customCachePath: customPath}
}

// LoadCache loads releases for a repository
func (m *Manager) LoadCache(repoConfig *config.RepositoryConfig) ([]version.Release, error) {
	// Priority: custom cache > embedded cache > no cache

	if m.customCachePath != "" {
		return m.loadCustomCache(m.customCachePath)
	}

	if repoConfig.CacheEnabled && repoConfig.CachePath != "" {
		return m.loadEmbeddedCache(repoConfig.CachePath)
	}

	return nil, nil // No cache available
}

func (m *Manager) loadEmbeddedCache(path string) ([]version.Release, error) {
	data, err := embeddedCaches.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded cache %s: %w", path, err)
	}

	return m.parseCache(data)
}

func (m *Manager) loadCustomCache(path string) ([]version.Release, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom cache %s: %w", path, err)
	}

	return m.parseCache(data)
}

func (m *Manager) parseCache(data []byte) ([]version.Release, error) {
	var cacheData CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	// Convert jsonRelease to version.Release
	releases := make([]version.Release, 0, len(cacheData.Releases))
	for _, jr := range cacheData.Releases {
		rel, err := jr.toVersionRelease()
		if err != nil {
			// Skip invalid releases but continue processing
			continue
		}
		releases = append(releases, rel)
	}

	return releases, nil
}
