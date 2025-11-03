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
	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")

	// Create GitHub client for Kubernetes
	ghClient := client.NewClient(token, "kubernetes", "kubernetes")

	// Create a version-based policy: support up to 3 minor versions behind
	pol := policy.NewVersionsPolicy(3)

	// Create checker with version-based policy
	versionChecker := checker.NewCheckerWithPolicy(ghClient, checker.Config{
		NoCache: false,
	}, pol)

	// Check if version 1.28.0 is still supported
	analysis, err := versionChecker.Analyse(context.Background(), "1.28.0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("Latest Kubernetes version: %s\n", analysis.LatestVersion)
	fmt.Printf("Your version: %s\n", analysis.ComparisonVersion)
	fmt.Printf("Status: %s\n", analysis.Status())
	fmt.Printf("Minor versions behind: %d\n", analysis.MinorVersionsBehind)
	fmt.Printf("Policy type: %s\n", analysis.PolicyType)

	if analysis.IsExpired {
		fmt.Printf("⚠️  Version %s is UNSUPPORTED (more than 3 minor versions behind)\n", analysis.ComparisonVersion)
		os.Exit(2)
	} else if analysis.IsCritical {
		fmt.Printf("🔶 Version %s is CRITICAL (at support boundary: 3 minor versions behind)\n", analysis.ComparisonVersion)
		os.Exit(1)
	} else if analysis.ReleasesBehind > 0 {
		fmt.Printf("✅ Version %s is supported (%d minor versions behind)\n", analysis.ComparisonVersion, analysis.MinorVersionsBehind)
		os.Exit(0)
	} else {
		fmt.Printf("✅ Version %s is up to date\n", analysis.ComparisonVersion)
		os.Exit(0)
	}
}
