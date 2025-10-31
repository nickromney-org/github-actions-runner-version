package data

import (
	"testing"
)

func TestLoadEmbeddedReleases(t *testing.T) {
	releases, err := LoadEmbeddedReleases()
	if err != nil {
		t.Fatalf("LoadEmbeddedReleases failed: %v", err)
	}

	if len(releases) == 0 {
		t.Error("expected releases, got empty slice")
	}

	// Check first release has valid data
	first := releases[0]
	if first.Version == nil {
		t.Error("first release has nil version")
	}
	if first.PublishedAt.IsZero() {
		t.Error("first release has zero published date")
	}
	if first.URL == "" {
		t.Error("first release has empty URL")
	}
}
