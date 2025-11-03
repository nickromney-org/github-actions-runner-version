package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/nickromney-org/github-actions-runner-version/internal/config"
	"github.com/nickromney-org/github-actions-runner-version/internal/data"
	"github.com/nickromney-org/github-actions-runner-version/pkg/checker"
	"github.com/nickromney-org/github-actions-runner-version/pkg/client"
)

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	repo := flag.String("repo", "actions/runner", "Repository to check (e.g., 'actions/runner', 'kubernetes')")
	flag.Parse()

	// Parse repository
	repoConfig, err := config.ParseRepositoryString(*repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid repository %q: %v\n", *repo, err)
		os.Exit(1)
	}

	// Create GitHub client
	ghClient := client.NewClient(*token, repoConfig.Owner, repoConfig.Repo)
	ctx := context.Background()

	// Load embedded releases
	embeddedData, err := data.LoadEmbeddedReleases()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading embedded releases: %v\n", err)
		os.Exit(1)
	}

	// Convert data.Release to checker.Release
	embedded := make([]checker.Release, len(embeddedData))
	for i, r := range embeddedData {
		embedded[i] = checker.Release{
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

	latestEmbeddedRelease := checker.FindLatestRelease(embedded)
	if latestEmbeddedRelease == nil {
		fmt.Fprintf(os.Stderr, "Error: Could not find latest embedded release\n")
		os.Exit(1)
	}
	latestEmbedded := latestEmbeddedRelease.Version.String()

	// Fetch 5 most recent from API
	recent, err := ghClient.GetRecentReleases(ctx, 5)
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
	latestAvailableRelease := checker.FindLatestRelease(recent)
	latestAvailable := ""
	if latestAvailableRelease != nil {
		latestAvailable = latestAvailableRelease.Version.String()
	}

	fmt.Printf("⚠️  Cache needs update (latest embedded: %s, latest available: %s)\n",
		latestEmbedded, latestAvailable)
	os.Exit(1)
}
