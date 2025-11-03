package cache

import (
	"os"
	"testing"
	"time"

	"github.com/nickromney-org/github-release-version-checker/internal/config"
)

func TestManager_LoadCache_Embedded(t *testing.T) {
	manager := NewManager("")

	repoConfig := &config.ConfigActionsRunner

	releases, err := manager.LoadCache(repoConfig)
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}

	if len(releases) == 0 {
		t.Error("expected releases, got empty list")
	}

	// Verify releases are properly parsed
	for _, rel := range releases {
		if rel.Version == nil {
			t.Error("release has nil version")
		}
		if rel.PublishedAt.IsZero() {
			t.Error("release has zero published date")
		}
		if rel.URL == "" {
			t.Error("release has empty URL")
		}
	}
}

func TestManager_LoadCache_CustomCache(t *testing.T) {
	// Create a temporary custom cache file
	tmpFile, err := os.CreateTemp("", "test-cache-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cacheContent := `{
		"generated_at": "2024-11-03T00:00:00Z",
		"repository": "test/test",
		"releases": [
			{
				"version": "1.0.0",
				"published_at": "2024-10-01T00:00:00Z",
				"url": "https://github.com/test/test/releases/tag/v1.0.0"
			},
			{
				"version": "0.9.0",
				"published_at": "2024-09-01T00:00:00Z",
				"url": "https://github.com/test/test/releases/tag/v0.9.0"
			}
		]
	}`

	if _, err := tmpFile.WriteString(cacheContent); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}
	tmpFile.Close()

	manager := NewManager(tmpFile.Name())

	repoConfig := &config.RepositoryConfig{
		Owner:        "test",
		Repo:         "test",
		CacheEnabled: true,
		CachePath:    "data/test.json",
	}

	releases, err := manager.LoadCache(repoConfig)
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}

	if len(releases) != 2 {
		t.Errorf("expected 2 releases, got %d", len(releases))
	}

	if releases[0].Version.String() != "1.0.0" {
		t.Errorf("first release version = %v, want 1.0.0", releases[0].Version.String())
	}
}

func TestManager_LoadCache_NoCache(t *testing.T) {
	manager := NewManager("")

	repoConfig := &config.RepositoryConfig{
		Owner:        "test",
		Repo:         "test",
		CacheEnabled: false,
	}

	releases, err := manager.LoadCache(repoConfig)
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}

	if releases != nil {
		t.Errorf("expected nil releases, got %d releases", len(releases))
	}
}

func TestManager_LoadCache_CustomCachePriority(t *testing.T) {
	// Create a temporary custom cache file
	tmpFile, err := os.CreateTemp("", "test-cache-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cacheContent := `{
		"generated_at": "2024-11-03T00:00:00Z",
		"repository": "custom/custom",
		"releases": [
			{
				"version": "2.0.0",
				"published_at": "2024-10-01T00:00:00Z",
				"url": "https://github.com/custom/custom/releases/tag/v2.0.0"
			}
		]
	}`

	if _, err := tmpFile.WriteString(cacheContent); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}
	tmpFile.Close()

	manager := NewManager(tmpFile.Name())

	// Even with embedded cache enabled, custom cache should take priority
	repoConfig := &config.ConfigActionsRunner

	releases, err := manager.LoadCache(repoConfig)
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}

	if len(releases) != 1 {
		t.Errorf("expected 1 release from custom cache, got %d", len(releases))
	}

	if len(releases) > 0 && releases[0].Version.String() != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %v", releases[0].Version.String())
	}
}

func TestManager_LoadCache_InvalidEmbeddedPath(t *testing.T) {
	manager := NewManager("")

	repoConfig := &config.RepositoryConfig{
		Owner:        "test",
		Repo:         "test",
		CacheEnabled: true,
		CachePath:    "data/nonexistent.json",
	}

	_, err := manager.LoadCache(repoConfig)
	if err == nil {
		t.Error("expected error for nonexistent cache, got nil")
	}
}

func TestManager_LoadCache_InvalidCustomPath(t *testing.T) {
	manager := NewManager("/nonexistent/cache.json")

	repoConfig := &config.RepositoryConfig{
		Owner:        "test",
		Repo:         "test",
		CacheEnabled: true,
	}

	_, err := manager.LoadCache(repoConfig)
	if err == nil {
		t.Error("expected error for nonexistent custom cache, got nil")
	}
}

func TestManager_LoadCache_InvalidJSON(t *testing.T) {
	// Create a temporary custom cache file with invalid JSON
	tmpFile, err := os.CreateTemp("", "test-cache-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("invalid json"); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}
	tmpFile.Close()

	manager := NewManager(tmpFile.Name())

	repoConfig := &config.RepositoryConfig{
		Owner:        "test",
		Repo:         "test",
		CacheEnabled: true,
	}

	_, err = manager.LoadCache(repoConfig)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestCacheData_Structure(t *testing.T) {
	// Test that CacheData structure matches expected format
	cacheData := CacheData{
		GeneratedAt: time.Now(),
		Repository:  "test/test",
		Releases:    []jsonRelease{},
	}

	if cacheData.Repository != "test/test" {
		t.Errorf("Repository = %v, want test/test", cacheData.Repository)
	}

	if cacheData.GeneratedAt.IsZero() {
		t.Error("GeneratedAt is zero")
	}

	if cacheData.Releases == nil {
		t.Error("Releases is nil")
	}
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name       string
		customPath string
	}{
		{
			name:       "without custom cache",
			customPath: "",
		},
		{
			name:       "with custom cache",
			customPath: "/tmp/cache.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.customPath)
			if manager == nil {
				t.Fatal("NewManager returned nil")
			}
			if manager.customCachePath != tt.customPath {
				t.Errorf("customCachePath = %v, want %v", manager.customCachePath, tt.customPath)
			}
		})
	}
}
