package cmd

import (
	"testing"
	"time"
)

func TestFormatUKDate(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{
			name:     "standard date",
			date:     time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC),
			expected: "25 Jul 2024",
		},
		{
			name:     "single digit day",
			date:     time.Date(2024, 10, 5, 0, 0, 0, 0, time.UTC),
			expected: "5 Oct 2024",
		},
		{
			name:     "december",
			date:     time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			expected: "31 Dec 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUKDate(tt.date)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
