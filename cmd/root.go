package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/nickromney-org/github-actions-runner-version/internal/github"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
)

var (
	comparisonVersion string
	criticalAgeDays   int
	maxAgeDays        int
	verbose           bool
	jsonOutput        bool
	ciOutput          bool
	githubToken       string

	// Colors for output
	green   = color.New(color.FgGreen, color.Bold)
	yellow  = color.New(color.FgYellow, color.Bold)
	red     = color.New(color.FgRed, color.Bold)
	cyan    = color.New(color.FgCyan)
	gray    = color.New(color.Faint)
)

var rootCmd = &cobra.Command{
	Use:   "runner-version-check",
	Short: "Check GitHub Actions runner version status",
	Long: `Check if your GitHub Actions self-hosted runner version is up to date.

According to GitHub's policy, runners must be updated within 30 days of any
new release (major, minor, or patch). This tool helps you stay compliant.`,
	Example: `  # Check latest version
  runner-version-check

  # Check a specific version
  runner-version-check -c 2.327.1

  # Verbose output with custom thresholds
  runner-version-check -c 2.327.1 -v -d 10 -m 30

  # JSON output for automation
  runner-version-check -c 2.327.1 --json`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&comparisonVersion, "compare", "c", "", "version to compare against (e.g., 2.327.1)")
	rootCmd.Flags().IntVarP(&criticalAgeDays, "critical-days", "d", 12, "days before critical warning")
	rootCmd.Flags().IntVarP(&maxAgeDays, "max-days", "m", 30, "days before version expires")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	rootCmd.Flags().BoolVar(&ciOutput, "ci", false, "format output for CI/GitHub Actions")
	rootCmd.Flags().StringVarP(&githubToken, "token", "t", os.Getenv("GITHUB_TOKEN"), "GitHub token (or GITHUB_TOKEN env var)")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	// Validate inputs
	if criticalAgeDays >= maxAgeDays {
		return fmt.Errorf("critical-days (%d) must be less than max-days (%d)", criticalAgeDays, maxAgeDays)
	}

	// Create GitHub client
	client := github.NewClient(githubToken)

	// Create checker with config
	checker := version.NewChecker(client, version.CheckerConfig{
		CriticalAgeDays: criticalAgeDays,
		MaxAgeDays:      maxAgeDays,
	})

	// Run analysis
	analysis, err := checker.Analyze(cmd.Context(), comparisonVersion)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Output results
	if jsonOutput {
		return outputJSON(analysis)
	}

	if ciOutput {
		return outputCI(analysis)
	}

	return outputTerminal(analysis)
}

func outputJSON(analysis *version.Analysis) error {
	data, err := analysis.MarshalJSON()
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputCI(analysis *version.Analysis) error {
	// Always print latest version first (for script compatibility)
	fmt.Println(analysis.LatestVersion)

	// If no comparison, we're done
	if analysis.ComparisonVersion == nil {
		return nil
	}

	status := analysis.Status()

	// Print to stdout using GitHub Actions workflow commands
	fmt.Println()
	fmt.Println("::group::📊 Runner Version Check")
	fmt.Printf("Latest version: v%s\n", analysis.LatestVersion)
	fmt.Printf("Your version: v%s\n", analysis.ComparisonVersion)
	fmt.Printf("Status: %s\n", getStatusText(status))
	fmt.Println("::endgroup::")
	fmt.Println()

	// Use appropriate workflow command based on status
	switch status {
	case version.StatusExpired:
		fmt.Printf("::error title=Runner Version Expired::🚨 Version %s EXPIRED! (%d releases behind AND %d days overdue)\n",
			analysis.ComparisonVersion, analysis.ReleasesBehind, analysis.DaysSinceUpdate-analysis.MaxAgeDays)
		if analysis.FirstNewerVersion != nil {
			fmt.Printf("::error::Update required: v%s was released %d days ago\n",
				analysis.FirstNewerVersion, analysis.DaysSinceUpdate)
		}
		fmt.Printf("::error::Latest version: v%s\n", analysis.LatestVersion)

	case version.StatusCritical:
		daysLeft := analysis.MaxAgeDays - analysis.DaysSinceUpdate
		fmt.Printf("::warning title=Runner Version Critical::⚠️  Version %s expires in %d days! (%d releases behind)\n",
			analysis.ComparisonVersion, daysLeft, analysis.ReleasesBehind)
		if analysis.FirstNewerVersion != nil {
			fmt.Printf("::warning::Update available: v%s (released %d days ago)\n",
				analysis.FirstNewerVersion, analysis.DaysSinceUpdate)
		}
		fmt.Printf("::warning::Latest version: v%s\n", analysis.LatestVersion)

	case version.StatusWarning:
		fmt.Printf("::notice title=Runner Version Behind::ℹ️  Version %s is %d releases behind\n",
			analysis.ComparisonVersion, analysis.ReleasesBehind)
		fmt.Printf("::notice::Latest version: v%s\n", analysis.LatestVersion)

	case version.StatusCurrent:
		fmt.Printf("::notice title=Runner Version Current::✅ Version %s is up to date\n",
			analysis.ComparisonVersion)
	}

	// List available updates
	if len(analysis.NewerReleases) > 0 {
		fmt.Println()
		fmt.Println("::group::📋 Available Updates")
		for i, release := range analysis.NewerReleases {
			releasedDaysAgo := int(time.Since(release.PublishedAt).Hours() / 24)
			label := ""
			if i == 0 {
				label = " [Latest]"
			} else if analysis.FirstNewerVersion != nil && release.Version.Equal(analysis.FirstNewerVersion) {
				label = " [First newer release]"
			}
			fmt.Printf("  • v%s (%s, %d days ago)%s\n",
				release.Version,
				release.PublishedAt.Format("2006-01-02"),
				releasedDaysAgo,
				label)
		}
		fmt.Println("::endgroup::")
	}

	// Write markdown summary to $GITHUB_STEP_SUMMARY
	if summaryFile := os.Getenv("GITHUB_STEP_SUMMARY"); summaryFile != "" {
		if err := writeGitHubSummary(summaryFile, analysis); err != nil {
			fmt.Printf("::warning::Failed to write job summary: %v\n", err)
		}
	}

	return nil
}

func writeGitHubSummary(summaryFile string, analysis *version.Analysis) error {
	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	status := analysis.Status()
	statusEmoji := getStatusIcon(status)
	statusText := getStatusText(status)

	// Write markdown summary
	fmt.Fprintf(f, "## %s Runner Version Status: %s\n\n", statusEmoji, statusText)

	// Summary table
	fmt.Fprintf(f, "| Metric | Value |\n")
	fmt.Fprintf(f, "|--------|-------|\n")
	fmt.Fprintf(f, "| Current Version | v%s |\n", analysis.ComparisonVersion)
	fmt.Fprintf(f, "| Latest Version | v%s |\n", analysis.LatestVersion)
	fmt.Fprintf(f, "| Status | %s %s |\n", statusEmoji, statusText)
	fmt.Fprintf(f, "| Releases Behind | %d |\n", analysis.ReleasesBehind)

	if analysis.DaysSinceUpdate > 0 {
		if analysis.IsExpired {
			daysOver := analysis.DaysSinceUpdate - analysis.MaxAgeDays
			fmt.Fprintf(f, "| Days Overdue | %d |\n", daysOver)
		} else {
			daysLeft := analysis.MaxAgeDays - analysis.DaysSinceUpdate
			fmt.Fprintf(f, "| Days Until Expiry | %d |\n", daysLeft)
		}
	}

	// Action required section
	if status == version.StatusExpired {
		fmt.Fprintf(f, "\n### ⚠️ Action Required\n\n")
		fmt.Fprintf(f, "**Update to v%s or later immediately.** ", analysis.FirstNewerVersion)
		fmt.Fprintf(f, "GitHub will not queue jobs to runners with expired versions.\n")
	} else if status == version.StatusCritical {
		daysLeft := analysis.MaxAgeDays - analysis.DaysSinceUpdate
		fmt.Fprintf(f, "\n### ⚠️ Update Soon\n\n")
		fmt.Fprintf(f, "Version expires in **%d days**. Update to v%s or later.\n", daysLeft, analysis.FirstNewerVersion)
	} else if status == version.StatusWarning {
		fmt.Fprintf(f, "\n### ℹ️ Update Available\n\n")
		fmt.Fprintf(f, "A newer version (v%s) is available.\n", analysis.LatestVersion)
	}

	// Available updates
	if len(analysis.NewerReleases) > 0 {
		fmt.Fprintf(f, "\n### 📦 Available Updates\n\n")
		for _, release := range analysis.NewerReleases {
			releasedDaysAgo := int(time.Since(release.PublishedAt).Hours() / 24)
			fmt.Fprintf(f, "- [v%s](%s) - Released %s (%d days ago)\n",
				release.Version,
				release.URL,
				release.PublishedAt.Format("Jan 2, 2006"),
				releasedDaysAgo)
		}
	}

	fmt.Fprintf(f, "\n---\n\n")

	return nil
}

func getStatusText(status version.Status) string {
	switch status {
	case version.StatusCurrent:
		return "Current"
	case version.StatusWarning:
		return "Behind"
	case version.StatusCritical:
		return "Critical"
	case version.StatusExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

func outputTerminal(analysis *version.Analysis) error {
	// Always print latest version first (for script compatibility)
	fmt.Println(analysis.LatestVersion)

	// If no comparison, we're done
	if analysis.ComparisonVersion == nil {
		return nil
	}

	// Print detailed status
	fmt.Println()
	printStatus(analysis)

	if verbose {
		fmt.Println()
		printDetails(analysis)
	}

	return nil
}

func printStatus(analysis *version.Analysis) {
	status := analysis.Status()
	icon := getStatusIcon(status)
	colorFunc := getStatusColor(status)

	// Main status line
	colorFunc.Printf("%s %s\n", icon, analysis.Message)

	// Additional context
	if status == version.StatusExpired || status == version.StatusCritical {
		fmt.Println()
		if analysis.FirstNewerVersion != nil {
			cyan.Printf("   📦 Update available: v%s\n", analysis.FirstNewerVersion)
			if analysis.FirstNewerReleaseDate != nil {
				gray.Printf("      Released: %s (%d days ago)\n",
					analysis.FirstNewerReleaseDate.Format("Jan 2, 2006"),
					analysis.DaysSinceUpdate)
			}
		}
		cyan.Printf("   🎯 Latest version: v%s\n", analysis.LatestVersion)

		if analysis.ReleasesBehind > 1 {
			yellow.Printf("   ⚠️  %d releases behind\n", analysis.ReleasesBehind)
		}
	}
}

func printDetails(analysis *version.Analysis) {
	cyan.Println("📊 Detailed Analysis")
	cyan.Println("─────────────────────────────────────")

	fmt.Printf("  Current version:      v%s\n", analysis.ComparisonVersion)
	fmt.Printf("  Latest version:       v%s\n", analysis.LatestVersion)
	fmt.Printf("  Status:               %s\n", analysis.Status())
	fmt.Printf("  Releases behind:      %d\n", analysis.ReleasesBehind)

	if analysis.FirstNewerVersion != nil {
		fmt.Printf("  First newer release:  v%s\n", analysis.FirstNewerVersion)
		if analysis.FirstNewerReleaseDate != nil {
			fmt.Printf("  Released on:          %s\n", analysis.FirstNewerReleaseDate.Format("2006-01-02"))
			fmt.Printf("  Days since update:    %d\n", analysis.DaysSinceUpdate)

			if analysis.DaysSinceUpdate < maxAgeDays {
				daysLeft := maxAgeDays - analysis.DaysSinceUpdate
				fmt.Printf("  Days until expired:   %d\n", daysLeft)
			} else {
				daysOver := analysis.DaysSinceUpdate - maxAgeDays
				fmt.Printf("  Days overdue:         %d\n", daysOver)
			}
		}
	}

	// Show available updates
	if len(analysis.NewerReleases) > 0 {
		fmt.Println()
		cyan.Println("📋 Available Updates")
		cyan.Println("─────────────────────────────────────")
		for _, release := range analysis.NewerReleases {
			releaseDate := release.PublishedAt.Format("2006-01-02")
			daysAgo := int(time.Since(release.PublishedAt).Hours() / 24)
			fmt.Printf("  • v%s (%s, %d days ago)\n", release.Version, releaseDate, daysAgo)
		}
	}
}

func getStatusIcon(status version.Status) string {
	switch status {
	case version.StatusCurrent:
		return "✅"
	case version.StatusWarning:
		return "⚠️ "
	case version.StatusCritical:
		return "🔶"
	case version.StatusExpired:
		return "🚨"
	default:
		return "ℹ️ "
	}
}

func getStatusColor(status version.Status) *color.Color {
	switch status {
	case version.StatusCurrent:
		return green
	case version.StatusWarning:
		return yellow
	case version.StatusCritical:
		return yellow
	case version.StatusExpired:
		return red
	default:
		return cyan
	}
}
