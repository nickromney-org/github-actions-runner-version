package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	colour "github.com/fatih/color"
	"github.com/nickromney-org/github-actions-runner-version/internal/github"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
	"github.com/spf13/cobra"
)

var (
	comparisonVersion string
	criticalAgeDays   int
	maxAgeDays        int
	verbose           bool
	jsonOutput        bool
	ciOutput          bool
	quiet             bool
	githubToken       string

	// Colours for output
	green  = colour.New(colour.FgGreen, colour.Bold)
	yellow = colour.New(colour.FgYellow, colour.Bold)
	red    = colour.New(colour.FgRed, colour.Bold)
	cyan   = colour.New(colour.FgCyan)
	grey   = colour.New(colour.FgHiBlack) // Faint grey for timestamps
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
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "quiet output (suppress expiry table)")
	rootCmd.Flags().StringVarP(&githubToken, "token", "t", os.Getenv("GITHUB_TOKEN"), "GitHub token (or GITHUB_TOKEN env var)")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	// Disable automatic usage printing on error
	cmd.SilenceUsage = true

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
	analysis, err := checker.Analyse(cmd.Context(), comparisonVersion)
	if err != nil {
		// If version doesn't exist, show helpful context instead of just erroring
		if strings.Contains(err.Error(), "does not exist in GitHub releases") {
			red.Printf("\n❌ Error: %v\n\n", err)

			// Fetch latest release to show helpful info
			latestRelease, fetchErr := client.GetLatestRelease(cmd.Context())
			if fetchErr == nil {
				yellow.Printf("💡 Use v%s (Released %s)\n", latestRelease.Version, formatUKDate(latestRelease.PublishedAt))

				// Show recent releases table if we can fetch them
				allReleases, fetchErr := client.GetAllReleases(cmd.Context())
				if fetchErr == nil && len(allReleases) > 0 {
					// Create a minimal analysis just for displaying the table
					tempAnalysis := &version.Analysis{
						LatestVersion: latestRelease.Version,
					}
					// Calculate recent releases for display
					tempChecker := version.NewChecker(client, version.CheckerConfig{})
					tempAnalysis.RecentReleases = tempChecker.CalculateRecentReleases(allReleases, latestRelease.Version, latestRelease.Version)

					printExpiryTable(tempAnalysis)
				}
			}

			os.Exit(1) // Exit with error code after showing helpful context
		}
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
	icon := getStatusIcon(status)

	// Build status line (same as terminal output but without colours)
	var statusLine string
	if analysis.IsLatest {
		comparisonDate := ""
		if analysis.ComparisonReleasedAt != nil {
			comparisonDate = fmt.Sprintf(" (%s)", formatUKDate(*analysis.ComparisonReleasedAt))
		}
		statusLine = fmt.Sprintf("Version %s%s is the latest version",
			analysis.ComparisonVersion,
			comparisonDate)
	} else {
		comparisonDate := ""
		if analysis.ComparisonReleasedAt != nil {
			comparisonDate = fmt.Sprintf(" (%s)", formatUKDate(*analysis.ComparisonReleasedAt))
		}

		expiryInfo := ""
		if analysis.FirstNewerReleaseDate != nil {
			expiryDate := analysis.FirstNewerReleaseDate.AddDate(0, 0, 30)

			if analysis.IsExpired {
				expiryInfo = fmt.Sprintf(" EXPIRED %s", formatUKDate(expiryDate))
			} else if analysis.IsCritical {
				daysLeft := 30 - analysis.DaysSinceUpdate
				expiryInfo = fmt.Sprintf(" EXPIRES %s (%d days)", formatUKDate(expiryDate), daysLeft)
			} else {
				expiryInfo = fmt.Sprintf(" expires %s", formatUKDate(expiryDate))
			}
		}

		latestDate := ""
		for _, r := range analysis.RecentReleases {
			if r.IsLatest {
				latestDate = fmt.Sprintf(" (Released %s)", formatUKDate(r.ReleasedAt))
				break
			}
		}

		statusLine = fmt.Sprintf("Version %s%s%s: Update to v%s%s",
			analysis.ComparisonVersion,
			comparisonDate,
			expiryInfo,
			analysis.LatestVersion,
			latestDate)
	}

	// Print GitHub Actions workflow commands
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
		fmt.Printf("::error title=Runner Version Expired::%s %s\n", icon, statusLine)
	case version.StatusCritical:
		fmt.Printf("::warning title=Runner Version Critical::%s %s\n", icon, statusLine)
	case version.StatusWarning:
		fmt.Printf("::notice title=Runner Version Behind::%s %s\n", icon, statusLine)
	case version.StatusCurrent:
		fmt.Printf("::notice title=Runner Version Current::%s %s\n", icon, statusLine)
	}

	// Print expiry table
	if len(analysis.RecentReleases) > 0 {
		fmt.Println()
		fmt.Println("::group::📅 Release Expiry Timeline")
		fmt.Printf("%-10s %-14s %-14s %s\n", "Version", "Release Date", "Expiry Date", "Status")

		for _, release := range analysis.RecentReleases {
			versionStr := release.Version.String()
			releasedStr := formatUKDate(release.ReleasedAt)

			var expiresStr string
			var statusStr string

			if release.IsLatest {
				expiresStr = "-"
				daysAgo := int(time.Since(release.ReleasedAt).Hours() / 24)
				statusStr = fmt.Sprintf("Latest (%s)", formatDaysAgo(daysAgo))
			} else if release.ExpiresAt != nil {
				expiresStr = formatUKDate(*release.ExpiresAt)

				if release.IsExpired {
					daysExpired := -release.DaysUntilExpiry
					statusStr = fmt.Sprintf("Expired %s", formatDaysAgo(daysExpired))
				} else {
					statusStr = fmt.Sprintf("Valid (%s left)", formatDaysInFuture(release.DaysUntilExpiry))
				}
			}

			arrow := ""
			if analysis.ComparisonVersion != nil && release.Version.Equal(analysis.ComparisonVersion) {
				arrow = "  [Your version]"
			}

			fmt.Printf("  %-10s %-14s %-14s %s%s\n", versionStr, releasedStr, expiresStr, statusStr, arrow)
		}

		// Add timestamp
		now := time.Now().UTC()
		timestamp := now.Format("2 Jan 2006 15:04:05 MST")
		fmt.Printf("\n  Checked at: %s\n", timestamp)

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
				formatUKDate(release.PublishedAt),
				releasedDaysAgo)
		}
	}

	// Add timestamp
	now := time.Now().UTC()
	timestamp := now.Format("2 Jan 2006 15:04:05 MST")
	fmt.Fprintf(f, "\n*Checked at: %s*\n", timestamp)

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

	// Print status
	fmt.Println()
	printStatus(analysis)

	// Print expiry table unless quiet mode
	if !quiet {
		printExpiryTable(analysis)
	}

	// Print verbose details if requested
	if verbose {
		fmt.Println()
		printDetails(analysis)
	}

	return nil
}

func printStatus(analysis *version.Analysis) {
	status := analysis.Status()
	icon := getStatusIcon(status)
	colourFunc := getStatusColour(status)

	var statusLine string

	if analysis.ComparisonVersion == nil {
		// No comparison - just show latest
		statusLine = fmt.Sprintf("%s Latest version: v%s", icon, analysis.LatestVersion)
	} else if analysis.IsLatest {
		// On latest version
		if analysis.ComparisonReleasedAt != nil {
			statusLine = fmt.Sprintf("%s Version %s (%s) is the latest version",
				icon,
				analysis.ComparisonVersion,
				formatUKDate(*analysis.ComparisonReleasedAt))
		} else {
			statusLine = fmt.Sprintf("%s Version %s is the latest version",
				icon,
				analysis.ComparisonVersion)
		}
	} else {
		// Behind - construct full status line
		comparisonDate := ""
		if analysis.ComparisonReleasedAt != nil {
			comparisonDate = fmt.Sprintf(" (%s)", formatUKDate(*analysis.ComparisonReleasedAt))
		}

		expiryInfo := ""
		if analysis.FirstNewerReleaseDate != nil {
			expiryDate := analysis.FirstNewerReleaseDate.AddDate(0, 0, 30)

			if analysis.IsExpired {
				expiryInfo = fmt.Sprintf(" EXPIRED %s", formatUKDate(expiryDate))
			} else if analysis.IsCritical {
				daysLeft := 30 - analysis.DaysSinceUpdate
				expiryInfo = fmt.Sprintf(" EXPIRES %s (%d days)", formatUKDate(expiryDate), daysLeft)
			} else {
				expiryInfo = fmt.Sprintf(" expires %s", formatUKDate(expiryDate))
			}
		}

		latestDate := ""
		for _, r := range analysis.RecentReleases {
			if r.IsLatest {
				latestDate = fmt.Sprintf(" (Released %s)", formatUKDate(r.ReleasedAt))
				break
			}
		}

		statusLine = fmt.Sprintf("%s Version %s%s%s: Update to v%s%s",
			icon,
			analysis.ComparisonVersion,
			comparisonDate,
			expiryInfo,
			analysis.LatestVersion,
			latestDate)
	}

	colourFunc.Println(statusLine)
}

func printExpiryTable(analysis *version.Analysis) {
	if len(analysis.RecentReleases) == 0 {
		return
	}

	fmt.Println()
	cyan.Println("📅 Release Expiry Timeline")
	cyan.Println("─────────────────────────────────────────────────────")
	fmt.Printf("%-10s %-14s %-14s %s\n", "Version", "Release Date", "Expiry Date", "Status")

	for _, release := range analysis.RecentReleases {
		versionStr := release.Version.String()
		releasedStr := formatUKDate(release.ReleasedAt)

		var expiresStr string
		var statusStr string

		if release.IsLatest {
			expiresStr = "-"
			daysAgo := int(time.Since(release.ReleasedAt).Hours() / 24)
			statusStr = fmt.Sprintf("✅ Latest (%s)", formatDaysAgo(daysAgo))
		} else if release.ExpiresAt != nil {
			expiresStr = formatUKDate(*release.ExpiresAt)

			if release.IsExpired {
				daysExpired := -release.DaysUntilExpiry
				statusStr = fmt.Sprintf("❌ Expired %s", formatDaysAgo(daysExpired))
			} else {
				statusStr = fmt.Sprintf("✅ Valid (%s left)", formatDaysInFuture(release.DaysUntilExpiry))
			}
		}

		// Mark user's version with bold and arrow
		arrow := ""
		isUserVersion := analysis.ComparisonVersion != nil && release.Version.Equal(analysis.ComparisonVersion)
		if isUserVersion {
			arrow = "  ← Your version"
			// Format the whole line in bold
			bold := colour.New(colour.Bold)
			bold.Printf("%-10s %-14s %-14s %s%s\n", versionStr, releasedStr, expiresStr, statusStr, arrow)
		} else {
			fmt.Printf("%-10s %-14s %-14s %s%s\n", versionStr, releasedStr, expiresStr, statusStr, arrow)
		}
	}

	// Add timestamp footer
	now := time.Now().UTC()
	timestamp := now.Format("2 Jan 2006 15:04:05 MST")
	grey.Printf("\nChecked at: %s\n", timestamp)
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

func getStatusColour(status version.Status) *colour.Color {
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
