package cmd

import (
	"fmt"
	"time"
)

// formatUKDate formats a date in UK format: "25 Jul 2024"
func formatUKDate(t time.Time) string {
	return t.Format("2 Jan 2006")
}

// formatDaysAgo returns a human-readable string for days
func formatDaysAgo(days int) string {
	if days < 0 {
		return formatDaysInFuture(-days)
	}
	if days == 0 {
		return "today"
	}
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// formatDaysInFuture returns a human-readable string for future days
func formatDaysInFuture(days int) string {
	if days == 0 {
		return "today"
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
