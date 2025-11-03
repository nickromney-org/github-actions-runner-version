package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nickromney-org/github-actions-runner-version/pkg/checker"
	"github.com/nickromney-org/github-actions-runner-version/pkg/client"
	"github.com/nickromney-org/github-actions-runner-version/pkg/policy"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <owner> <repo> <version>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s hashicorp terraform 1.5.0\n", os.Args[0])
		os.Exit(1)
	}

	owner := os.Args[1]
	repo := os.Args[2]
	version := os.Args[3]

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")

	// Create GitHub client for custom repository
	ghClient := client.NewClient(token, owner, repo)

	// Create a days-based policy
	pol := policy.NewDaysPolicy(12, 30)

	// Create checker
	versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
		NoCache:         true, // Disable cache for custom repos
	}, pol)

	// Analyse the version
	analysis, err := versionChecker.Analyse(context.Background(), version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analysing %s/%s version %s: %v\n", owner, repo, version, err)
		os.Exit(1)
	}

	// Print detailed results
	fmt.Printf("Repository: %s/%s\n", owner, repo)
	fmt.Printf("Latest version: %s\n", analysis.LatestVersion)
	fmt.Printf("Your version: %s\n", analysis.ComparisonVersion)
	fmt.Printf("Status: %s\n", analysis.Status())
	fmt.Printf("Releases behind: %d\n", analysis.ReleasesBehind)

	if analysis.IsLatest {
		fmt.Printf("✅ You are on the latest version!\n")
	} else {
		fmt.Printf("\nNewer releases available:\n")
		for i, rel := range analysis.NewerReleases {
			if i >= 5 { // Show only first 5
				fmt.Printf("  ... and %d more\n", len(analysis.NewerReleases)-5)
				break
			}
			fmt.Printf("  - %s (released %s)\n", rel.Version, rel.PublishedAt.Format("2006-01-02"))
		}
	}

	// Exit with status code
	if analysis.IsExpired {
		os.Exit(2)
	} else if analysis.IsCritical {
		os.Exit(1)
	}
	os.Exit(0)
}
