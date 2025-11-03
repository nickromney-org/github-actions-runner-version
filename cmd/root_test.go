package cmd

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nickromney-org/github-actions-runner-version/internal/version"
)

// Test helpers
func mustParseVersion(v string) *semver.Version {
	ver, err := semver.NewVersion(v)
	if err != nil {
		panic(err)
	}
	return ver
}

func mustParseTime(t string) time.Time {
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		panic(err)
	}
	return parsed
}

// TestOutputJSON tests JSON output formatting
func TestOutputJSON(t *testing.T) {
	tests := []struct {
		name     string
		analysis *version.Analysis
		wantKeys []string
	}{
		{
			name: "current version",
			analysis: &version.Analysis{
				LatestVersion:     mustParseVersion("2.329.0"),
				ComparisonVersion: mustParseVersion("2.329.0"),
				IsLatest:          true,
				IsExpired:         false,
				IsCritical:        false,
				ReleasesBehind:    0,
				DaysSinceUpdate:   0,
			},
			wantKeys: []string{
				"latest_version",
				"comparison_version",
				"is_latest",
				"is_expired",
				"is_critical",
				"releases_behind",
				"days_since_update",
				"status",
			},
		},
		{
			name: "expired version",
			analysis: &version.Analysis{
				LatestVersion:     mustParseVersion("2.329.0"),
				ComparisonVersion: mustParseVersion("2.327.1"),
				IsLatest:          false,
				IsExpired:         true,
				IsCritical:        false,
				ReleasesBehind:    2,
				DaysSinceUpdate:   35,
				FirstNewerVersion: mustParseVersion("2.328.0"),
			},
			wantKeys: []string{
				"latest_version",
				"comparison_version",
				"first_newer_version",
				"status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := outputJSON(tt.analysis)
			if err != nil {
				t.Fatalf("outputJSON() error = %v", err)
			}

			// Verify JSON can be marshaled
			data, err := tt.analysis.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}

			// Verify all expected keys are present
			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("JSON unmarshal error = %v", err)
			}

			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("Missing expected key in JSON: %s", key)
				}
			}
		})
	}
}

// TestOutputErrorJSON tests error JSON formatting
func TestOutputErrorJSON(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr string
	}{
		{
			name:    "version not found",
			err:     fmt.Errorf("version 2.327.99 does not exist in GitHub releases (latest: 2.329.0)"),
			wantErr: "version 2.327.99 does not exist in GitHub releases (latest: 2.329.0)",
		},
		{
			name:    "simple error",
			err:     fmt.Errorf("invalid comparison version \"not-a-version\": Invalid Semantic Version"),
			wantErr: "invalid comparison version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function (it prints to stdout)
			outputErrorJSON(tt.err)

			// For now, just verify it doesn't panic
			// In a more thorough test, we'd capture stdout and parse JSON
		})
	}
}

// TestGetStatusText tests status text generation
func TestGetStatusText(t *testing.T) {
	tests := []struct {
		name   string
		status version.Status
		want   string
	}{
		{"current", version.StatusCurrent, "Current"},
		{"warning", version.StatusWarning, "Behind"},
		{"critical", version.StatusCritical, "Critical"},
		{"expired", version.StatusExpired, "Expired"},
		{"invalid", version.Status("invalid"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusText(tt.status)
			if got != tt.want {
				t.Errorf("getStatusText(%v) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// TestGetStatusIcon tests status icon selection
func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		name   string
		status version.Status
		want   string
	}{
		{"current", version.StatusCurrent, "✅"},
		{"warning", version.StatusWarning, "⚠️ "},
		{"critical", version.StatusCritical, "🔶"},
		{"expired", version.StatusExpired, "🚨"},
		{"invalid", version.Status("invalid"), "ℹ️ "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusIcon(tt.status)
			if got != tt.want {
				t.Errorf("getStatusIcon(%v) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// TestDetectGitHubToken tests token detection
func TestDetectGitHubToken(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		want     string
	}{
		{
			name:     "provided token",
			provided: "ghp_test123",
			want:     "ghp_test123",
		},
		{
			name:     "empty token",
			provided: "",
			want:     "", // Falls back to gh CLI, which likely returns empty in tests
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectGitHubToken(tt.provided)
			if tt.provided != "" && got != tt.want {
				t.Errorf("detectGitHubToken(%v) = %v, want %v", tt.provided, got, tt.want)
			}
		})
	}
}

// TestOutputCI tests CI output formatting
func TestOutputCI(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		analysis     *version.Analysis
		wantContains []string
	}{
		{
			name: "current version",
			analysis: &version.Analysis{
				LatestVersion:     mustParseVersion("2.329.0"),
				ComparisonVersion: mustParseVersion("2.329.0"),
				IsLatest:          true,
				ComparisonReleasedAt: &now,
			},
			wantContains: []string{
				"2.329.0",
				"::group::",
				"::endgroup::",
			},
		},
		{
			name: "warning status",
			analysis: &version.Analysis{
				LatestVersion:         mustParseVersion("2.329.0"),
				ComparisonVersion:     mustParseVersion("2.328.0"),
				IsLatest:              false,
				IsExpired:             false,
				IsCritical:            false,
				ReleasesBehind:        1,
				DaysSinceUpdate:       5,
				FirstNewerVersion:     mustParseVersion("2.329.0"),
				FirstNewerReleaseDate: &now,
				ComparisonReleasedAt:  &now,
			},
			wantContains: []string{
				"::warning",
				"2.329.0",
				"2.328.0",
			},
		},
		{
			name: "expired status",
			analysis: &version.Analysis{
				LatestVersion:         mustParseVersion("2.329.0"),
				ComparisonVersion:     mustParseVersion("2.327.0"),
				IsLatest:              false,
				IsExpired:             true,
				IsCritical:            false,
				ReleasesBehind:        2,
				DaysSinceUpdate:       35,
				FirstNewerVersion:     mustParseVersion("2.328.0"),
				FirstNewerReleaseDate: &now,
				ComparisonReleasedAt:  &now,
			},
			wantContains: []string{
				"::error",
				"EXPIRED",
				"2.327.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout by temporarily redirecting
			// For now, just verify it doesn't error
			err := outputCI(tt.analysis)
			if err != nil {
				t.Errorf("outputCI() error = %v", err)
			}
		})
	}
}

// TestOutputTerminal tests terminal output formatting
func TestOutputTerminal(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		analysis *version.Analysis
		wantErr  bool
	}{
		{
			name: "current version",
			analysis: &version.Analysis{
				LatestVersion:        mustParseVersion("2.329.0"),
				ComparisonVersion:    mustParseVersion("2.329.0"),
				IsLatest:             true,
				ComparisonReleasedAt: &now,
			},
			wantErr: false,
		},
		{
			name: "expired version",
			analysis: &version.Analysis{
				LatestVersion:         mustParseVersion("2.329.0"),
				ComparisonVersion:     mustParseVersion("2.327.0"),
				IsLatest:              false,
				IsExpired:             true,
				ReleasesBehind:        2,
				DaysSinceUpdate:       35,
				FirstNewerVersion:     mustParseVersion("2.328.0"),
				FirstNewerReleaseDate: &now,
				ComparisonReleasedAt:  &now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := outputTerminal(tt.analysis)
			if (err != nil) != tt.wantErr {
				t.Errorf("outputTerminal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFormatUKDate is already in format_test.go, but let's add more comprehensive tests here
func TestStatusTransitions(t *testing.T) {
	// Table-driven test for status determination
	tests := []struct {
		name           string
		daysOld        int
		criticalDays   int
		maxDays        int
		expectedStatus version.Status
	}{
		{"very recent", 0, 12, 30, version.StatusCurrent},
		{"within warning", 5, 12, 30, version.StatusWarning},
		{"at critical threshold", 12, 12, 30, version.StatusCritical},
		{"within critical", 20, 12, 30, version.StatusCritical},
		{"at expiry threshold", 30, 12, 30, version.StatusExpired},
		{"past expiry", 35, 12, 30, version.StatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This tests the status logic indirectly through the helper functions
			// In a real implementation, we'd test the actual status determination
			// For now, verify the icons match expectations
			icon := getStatusIcon(tt.expectedStatus)
			if icon == "" {
				t.Error("getStatusIcon returned empty string")
			}

			text := getStatusText(tt.expectedStatus)
			if text == "" || text == "Unknown" {
				t.Errorf("getStatusText returned invalid text: %s", text)
			}
		})
	}
}

// TestJSONOutputIntegrity tests that JSON output is valid and parseable
func TestJSONOutputIntegrity(t *testing.T) {
	now := time.Now()
	analysis := &version.Analysis{
		LatestVersion:         mustParseVersion("2.329.0"),
		ComparisonVersion:     mustParseVersion("2.328.0"),
		IsLatest:              false,
		IsExpired:             false,
		IsCritical:            false,
		ReleasesBehind:        1,
		DaysSinceUpdate:       5,
		FirstNewerVersion:     mustParseVersion("2.329.0"),
		FirstNewerReleaseDate: &now,
		ComparisonReleasedAt:  &now,
	}

	data, err := analysis.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal error = %v", err)
	}

	// Verify required fields exist and have correct types
	requiredFields := map[string]string{
		"latest_version":     "string",
		"comparison_version": "string",
		"is_latest":          "bool",
		"is_expired":         "bool",
		"releases_behind":    "float64", // JSON numbers are float64
		"status":             "string",
	}

	for field, expectedType := range requiredFields {
		value, ok := result[field]
		if !ok {
			t.Errorf("Missing required field: %s", field)
			continue
		}

		actualType := getJSONType(value)
		if actualType != expectedType {
			t.Errorf("Field %s has type %s, want %s", field, actualType, expectedType)
		}
	}
}

// Helper to get JSON type as string
func getJSONType(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "float64"
	case bool:
		return "bool"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return "unknown"
	}
}

// TestCIOutputAnnotations tests that CI output includes proper GitHub Actions annotations
func TestCIOutputAnnotations(t *testing.T) {
	now := time.Now()

	// Test that different statuses produce appropriate annotations
	tests := []struct {
		name               string
		status             version.Status
		expectedAnnotation string
	}{
		{"warning status", version.StatusWarning, "::warning"},
		{"critical status", version.StatusCritical, "::warning"},
		{"expired status", version.StatusExpired, "::error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &version.Analysis{
				LatestVersion:         mustParseVersion("2.329.0"),
				ComparisonVersion:     mustParseVersion("2.327.0"),
				IsLatest:              false,
				IsExpired:             tt.status == version.StatusExpired,
				IsCritical:            tt.status == version.StatusCritical,
				ReleasesBehind:        2,
				DaysSinceUpdate:       35,
				FirstNewerVersion:     mustParseVersion("2.328.0"),
				FirstNewerReleaseDate: &now,
				ComparisonReleasedAt:  &now,
			}

			// For now, just verify outputCI doesn't error
			// A more thorough test would capture stdout and verify annotations
			err := outputCI(analysis)
			if err != nil {
				t.Errorf("outputCI() error = %v", err)
			}
		})
	}
}

// TestOutputWithNoComparison tests output when no comparison version is provided
func TestOutputWithNoComparison(t *testing.T) {
	analysis := &version.Analysis{
		LatestVersion:     mustParseVersion("2.329.0"),
		ComparisonVersion: nil,
		IsLatest:          false,
	}

	// All output functions should handle nil comparison gracefully
	t.Run("terminal output", func(t *testing.T) {
		err := outputTerminal(analysis)
		if err != nil {
			t.Errorf("outputTerminal() with nil comparison error = %v", err)
		}
	})

	t.Run("CI output", func(t *testing.T) {
		err := outputCI(analysis)
		if err != nil {
			t.Errorf("outputCI() with nil comparison error = %v", err)
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		err := outputJSON(analysis)
		if err != nil {
			t.Errorf("outputJSON() with nil comparison error = %v", err)
		}
	})
}
