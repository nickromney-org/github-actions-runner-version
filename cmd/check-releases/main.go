package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/nickromney-org/github-actions-runner-version/internal/data"
	"github.com/nickromney-org/github-actions-runner-version/internal/github"
)

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	flag.Parse()

	client := github.NewClient(*token)
	ctx := context.Background()

	// Load embedded releases
	embedded, err := data.LoadEmbeddedReleases()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading embedded releases: %v\n", err)
		os.Exit(1)
	}

	// Find latest embedded version
	if len(embedded) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No embedded releases found\n")
		os.Exit(1)
	}

	latestEmbeddedVersion := embedded[0].Version
	for _, r := range embedded {
		if r.Version.GreaterThan(latestEmbeddedVersion) {
			latestEmbeddedVersion = r.Version
		}
	}
	latestEmbedded := latestEmbeddedVersion.String()

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

	// Get latest available version from API
	latestAvailable := ""
	if len(recent) > 0 {
		latestAvailableVersion := recent[0].Version
		// Find the actual latest by comparing all
		for _, r := range recent {
			if r.Version.GreaterThan(latestAvailableVersion) {
				latestAvailableVersion = r.Version
			}
		}
		latestAvailable = latestAvailableVersion.String()
	}

	fmt.Printf("⚠️  Cache needs update (latest embedded: %s, latest available: %s)\n",
		latestEmbedded, latestAvailable)
	os.Exit(1)
}
