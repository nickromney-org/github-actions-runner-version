package version

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
)

func TestReleaseExpiry_JSONMarshaling(t *testing.T) {
	releaseDate := time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(2024, 8, 24, 0, 0, 0, 0, time.UTC)

	expiry := ReleaseExpiry{
		Version:         semver.MustParse("2.327.1"),
		ReleasedAt:      releaseDate,
		ExpiresAt:       &expiryDate,
		DaysUntilExpiry: -77,
		IsExpired:       true,
		IsLatest:        false,
	}

	data, err := json.Marshal(expiry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["version"] != "2.327.1" {
		t.Errorf("expected version 2.327.1, got %v", result["version"])
	}
}

// TestAnalysis_Status tests status determination logic
func TestAnalysis_Status(t *testing.T) {
	tests := []struct {
		name     string
		analysis *Analysis
		want     Status
	}{
		{
			name: "nil comparison version",
			analysis: &Analysis{
				ComparisonVersion: nil,
			},
			want: StatusCurrent,
		},
		{
			name: "expired",
			analysis: &Analysis{
				ComparisonVersion: semver.MustParse("2.327.0"),
				IsExpired:         true,
				IsCritical:        false,
				ReleasesBehind:    2,
			},
			want: StatusExpired,
		},
		{
			name: "critical",
			analysis: &Analysis{
				ComparisonVersion: semver.MustParse("2.327.0"),
				IsExpired:         false,
				IsCritical:        true,
				ReleasesBehind:    2,
			},
			want: StatusCritical,
		},
		{
			name: "warning",
			analysis: &Analysis{
				ComparisonVersion: semver.MustParse("2.328.0"),
				IsExpired:         false,
				IsCritical:        false,
				ReleasesBehind:    1,
			},
			want: StatusWarning,
		},
		{
			name: "current",
			analysis: &Analysis{
				ComparisonVersion: semver.MustParse("2.329.0"),
				IsExpired:         false,
				IsCritical:        false,
				ReleasesBehind:    0,
			},
			want: StatusCurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.analysis.Status()
			if got != tt.want {
				t.Errorf("Status() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAnalysis_MarshalJSON tests JSON marshalling
func TestAnalysis_MarshalJSON(t *testing.T) {
	now := time.Now()
	firstNewerTime := now.AddDate(0, 0, -5)

	tests := []struct {
		name         string
		analysis     *Analysis
		wantFields   map[string]interface{}
		wantMissing  []string
		checkStatus  bool
		expectStatus Status
	}{
		{
			name: "complete analysis",
			analysis: &Analysis{
				LatestVersion:         semver.MustParse("2.329.0"),
				ComparisonVersion:     semver.MustParse("2.328.0"),
				ComparisonReleasedAt:  &now,
				FirstNewerVersion:     semver.MustParse("2.329.0"),
				FirstNewerReleaseDate: &firstNewerTime,
				IsLatest:              false,
				IsExpired:             false,
				IsCritical:            false,
				ReleasesBehind:        1,
				DaysSinceUpdate:       5,
				Message:               "Update available",
				CriticalAgeDays:       12,
				MaxAgeDays:            30,
			},
			wantFields: map[string]interface{}{
				"latest_version":     "2.329.0",
				"comparison_version": "2.328.0",
				"is_latest":          false,
				"releases_behind":    float64(1),
			},
			checkStatus:  true,
			expectStatus: StatusWarning,
		},
		{
			name: "current version",
			analysis: &Analysis{
				LatestVersion:     semver.MustParse("2.329.0"),
				ComparisonVersion: semver.MustParse("2.329.0"),
				IsLatest:          true,
				ReleasesBehind:    0,
			},
			wantFields: map[string]interface{}{
				"latest_version":     "2.329.0",
				"comparison_version": "2.329.0",
				"is_latest":          true,
				"releases_behind":    float64(0),
			},
			checkStatus:  true,
			expectStatus: StatusCurrent,
		},
		{
			name: "nil comparison version",
			analysis: &Analysis{
				LatestVersion: semver.MustParse("2.329.0"),
				IsLatest:      false,
			},
			wantFields: map[string]interface{}{
				"latest_version": "2.329.0",
			},
			wantMissing: []string{
				"comparison_version",
				"first_newer_version",
			},
			checkStatus:  true,
			expectStatus: StatusCurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.analysis.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("JSON unmarshal error = %v", err)
			}

			// Check expected fields
			for key, expected := range tt.wantFields {
				got, ok := result[key]
				if !ok {
					t.Errorf("missing field %q", key)
					continue
				}
				if got != expected {
					t.Errorf("field %q = %v, want %v", key, got, expected)
				}
			}

			// Check missing fields
			for _, key := range tt.wantMissing {
				if _, ok := result[key]; ok {
					t.Errorf("field %q should not be present", key)
				}
			}

			// Check status
			if tt.checkStatus {
				status, ok := result["status"]
				if !ok {
					t.Error("missing status field")
				} else if status != string(tt.expectStatus) {
					t.Errorf("status = %v, want %v", status, tt.expectStatus)
				}
			}
		})
	}
}

// TestVersionString tests the versionString helper
func TestVersionString(t *testing.T) {
	tests := []struct {
		name string
		ver  *semver.Version
		want string
	}{
		{
			name: "valid version",
			ver:  semver.MustParse("2.329.0"),
			want: "2.329.0",
		},
		{
			name: "nil version",
			ver:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionString(tt.ver)
			if got != tt.want {
				t.Errorf("versionString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTimeString tests the timeString helper
func TestTimeString(t *testing.T) {
	now := time.Date(2024, 10, 31, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		time *time.Time
		want *string
	}{
		{
			name: "valid time",
			time: &now,
			want: stringPtr("2024-10-31T12:00:00Z"),
		},
		{
			name: "nil time",
			time: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeString(tt.time)
			if tt.want == nil {
				if got != nil {
					t.Errorf("timeString() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Error("timeString() = nil, want non-nil")
				} else if *got != *tt.want {
					t.Errorf("timeString() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
