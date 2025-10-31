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
