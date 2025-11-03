package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nickromney-org/github-actions-runner-version/pkg/checker"
	"github.com/nickromney-org/github-actions-runner-version/pkg/client"
	"github.com/nickromney-org/github-actions-runner-version/pkg/policy"
)

func main() {
	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")

	// Create GitHub client
	ghClient := client.NewClient(token, "actions", "runner")

	// Create policy
	pol := policy.NewDaysPolicy(12, 30)

	// Create checker
	versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
		CriticalAgeDays: 12,
		MaxAgeDays:      30,
		NoCache:         false,
	}, pol)

	// Analyse version
	analysis, err := versionChecker.Analyse(context.Background(), "2.328.0")
	if err != nil {
		// Output error as JSON
		errorOutput := map[string]string{
			"error": err.Error(),
		}
		json.NewEncoder(os.Stdout).Encode(errorOutput)
		os.Exit(1)
	}

	// The Analysis type implements custom JSON marshalling
	// It will output all fields in a formatted JSON structure
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(analysis); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	// Exit with appropriate code based on status
	switch analysis.Status() {
	case checker.StatusExpired:
		os.Exit(2)
	case checker.StatusCritical:
		os.Exit(1)
	default:
		os.Exit(0)
	}
}
