package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nickromney-org/github-actions-runner-version/internal/github"
)

type ReleaseJSON struct {
	Version     string    `json:"version"`
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url"`
}

type CacheFile struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Releases    []ReleaseJSON `json:"releases"`
}

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	output := flag.String("output", "internal/data/releases.json", "Output file")
	flag.Parse()

	// TODO: owner/repo will be configurable in Phase 3.2
	client := github.NewClient(*token, "actions", "runner")
	ctx := context.Background()

	fmt.Println("Fetching all releases from GitHub API...")

	// Fetch all releases
	releases, err := client.GetAllReleases(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Convert to JSON-friendly format
	jsonReleases := make([]ReleaseJSON, len(releases))
	for i, r := range releases {
		jsonReleases[i] = ReleaseJSON{
			Version:     r.Version.String(),
			PublishedAt: r.PublishedAt,
			URL:         r.URL,
		}
	}

	// Build cache file
	cache := CacheFile{
		GeneratedAt: time.Now().UTC(),
		Releases:    jsonReleases,
	}

	// Write to file
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cache); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Wrote %d releases to %s\n", len(releases), *output)
}
