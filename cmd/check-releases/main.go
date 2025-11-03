package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/nickromney-org/github-actions-runner-version/internal/data"
	"github.com/nickromney-org/github-actions-runner-version/internal/github"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
)

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	flag.Parse()

	// TODO: owner/repo will be configurable in Phase 3.2
	client := github.NewClient(*token, "actions", "runner")
	ctx := context.Background()

	// Load embedded releases
	embeddedData, err := data.LoadEmbeddedReleases()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading embedded releases: %v\n", err)
		os.Exit(1)
	}

	// Convert data.Release to version.Release
	embedded := make([]version.Release, len(embeddedData))
	for i, r := range embeddedData {
		embedded[i] = version.Release{
			Version:     r.Version,
			PublishedAt: r.PublishedAt,
			URL:         r.URL,
		}
	}

	// Find latest embedded version using helper
	if len(embedded) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No embedded releases found\n")
		os.Exit(1)
	}

	latestEmbeddedRelease := version.FindLatestRelease(embedded)
	if latestEmbeddedRelease == nil {
		fmt.Fprintf(os.Stderr, "Error: Could not find latest embedded release\n")
		os.Exit(1)
	}
	latestEmbedded := latestEmbeddedRelease.Version.String()

	// Fetch 5 most recent from API
	recent, err := client.GetRecentReleases(ctx, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching recent releases: %v\n", err)
		os.Exit(1)
	}

	// Check if latest embedded is in recent 5
	found := false
	for _, r := range recent {
		if r.Version.String() == latestEmbedded {
			found = true
			break
		}
	}

	if found {
		fmt.Printf("✅ Cache is current (latest: %s)\n", latestEmbedded)
		os.Exit(0)
	}

	// Get latest available version from API using helper
	latestAvailableRelease := version.FindLatestRelease(recent)
	latestAvailable := ""
	if latestAvailableRelease != nil {
		latestAvailable = latestAvailableRelease.Version.String()
	}

	fmt.Printf("⚠️  Cache needs update (latest embedded: %s, latest available: %s)\n",
		latestEmbedded, latestAvailable)
	os.Exit(1)
}
