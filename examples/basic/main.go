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
	// Get GitHub token from environment (optional, but recommended to avoid rate limits)
	token := os.Getenv("GITHUB_TOKEN")

	// Create GitHub client for actions/runner repository
	ghClient := client.NewClient(token, "actions", "runner")

	// Create a days-based policy: warn after 12 days, expire after 30 days
	pol := policy.NewDaysPolicy(12, 30)

	// Create checker with policy
	versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
		NoCache:         false, // Use embedded cache for faster lookups
	}, pol)

	// Analyse a specific version
	analysis, err := versionChecker.Analyse(context.Background(), "2.328.0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("Latest version: %s\n", analysis.LatestVersion)
	fmt.Printf("Your version: %s\n", analysis.ComparisonVersion)
	fmt.Printf("Status: %s\n", analysis.Status())
	fmt.Printf("Releases behind: %d\n", analysis.ReleasesBehind)
	fmt.Printf("Days since update available: %d\n", analysis.DaysSinceUpdate)
	fmt.Printf("Is expired: %v\n", analysis.IsExpired)
	fmt.Printf("Is critical: %v\n", analysis.IsCritical)
	fmt.Printf("Message: %s\n", analysis.Message)

	// Exit with appropriate code
	switch analysis.Status() {
	case checker.StatusExpired:
		os.Exit(2)
	case checker.StatusCritical:
		os.Exit(1)
	default:
		os.Exit(0)
	}
}
