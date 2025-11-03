package version

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nickromney-org/github-actions-runner-version/internal/types"
)

// Status represents the current state of a version
type Status string

const (
	StatusCurrent  Status = "current"
	StatusWarning  Status = "warning"
	StatusCritical Status = "critical"
	StatusExpired  Status = "expired"
)

// Release is a type alias for types.Release for backward compatibility
type Release = types.Release

// ReleaseExpiry represents expiry information for a single release
type ReleaseExpiry struct {
	Version         *semver.Version `json:"version"`
	ReleasedAt      time.Time       `json:"released"`
	ExpiresAt       *time.Time      `json:"expires"`
	DaysUntilExpiry int             `json:"days_until_expiry"`
	IsExpired       bool            `json:"is_expired"`
	IsLatest        bool            `json:"is_latest"`
}

// MarshalJSON implements custom JSON marshalling for ReleaseExpiry
func (r *ReleaseExpiry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Version         string  `json:"version"`
		ReleasedAt      string  `json:"released"`
		ExpiresAt       *string `json:"expires"`
		DaysUntilExpiry int     `json:"days_until_expiry"`
		IsExpired       bool    `json:"is_expired"`
		IsLatest        bool    `json:"is_latest"`
	}{
		Version:         r.Version.String(),
		ReleasedAt:      r.ReleasedAt.Format(time.RFC3339),
		ExpiresAt:       timeString(r.ExpiresAt),
		DaysUntilExpiry: r.DaysUntilExpiry,
		IsExpired:       r.IsExpired,
		IsLatest:        r.IsLatest,
	})
}

// Analysis contains the full version analysis results
type Analysis struct {
	LatestVersion         *semver.Version `json:"latest_version"`
	ComparisonVersion     *semver.Version `json:"comparison_version,omitempty"`
	ComparisonReleasedAt  *time.Time      `json:"comparison_released_at,omitempty"`
	IsLatest              bool            `json:"is_latest"`
	IsExpired             bool            `json:"is_expired"`
	IsCritical            bool            `json:"is_critical"`
	ReleasesBehind        int             `json:"releases_behind"`
	DaysSinceUpdate       int             `json:"days_since_update"`
	FirstNewerVersion     *semver.Version `json:"first_newer_version,omitempty"`
	FirstNewerReleaseDate *time.Time      `json:"first_newer_release_date,omitempty"`
	NewerReleases         []Release       `json:"newer_releases,omitempty"`
	RecentReleases        []ReleaseExpiry `json:"recent_releases,omitempty"`
	Message               string          `json:"message"`

	// Configuration used
	CriticalAgeDays int `json:"critical_age_days"`
	MaxAgeDays      int `json:"max_age_days"`

	// Policy information
	PolicyType          string `json:"policy_type,omitempty"`           // "days" or "versions"
	MinorVersionsBehind int    `json:"minor_versions_behind,omitempty"` // For version-based policies
}

// Status returns the current status level
func (a *Analysis) Status() Status {
	if a.ComparisonVersion == nil {
		return StatusCurrent
	}

	if a.IsExpired {
		return StatusExpired
	}

	if a.IsCritical {
		return StatusCritical
	}

	if a.ReleasesBehind > 0 {
		return StatusWarning
	}

	return StatusCurrent
}

// MarshalJSON implements custom JSON marshalling
func (a *Analysis) MarshalJSON() ([]byte, error) {
	type Alias Analysis
	return json.MarshalIndent(&struct {
		LatestVersion         string  `json:"latest_version"`
		ComparisonVersion     string  `json:"comparison_version,omitempty"`
		ComparisonReleasedAt  *string `json:"comparison_released_at,omitempty"`
		FirstNewerVersion     string  `json:"first_newer_version,omitempty"`
		FirstNewerReleaseDate *string `json:"first_newer_release_date,omitempty"`
		Status                Status  `json:"status"`
		*Alias
	}{
		LatestVersion:         a.LatestVersion.String(),
		ComparisonVersion:     versionString(a.ComparisonVersion),
		ComparisonReleasedAt:  timeString(a.ComparisonReleasedAt),
		FirstNewerVersion:     versionString(a.FirstNewerVersion),
		FirstNewerReleaseDate: timeString(a.FirstNewerReleaseDate),
		Status:                a.Status(),
		Alias:                 (*Alias)(a),
	}, "", "  ")
}

// CheckerConfig holds configuration for the version checker
type CheckerConfig struct {
	CriticalAgeDays int
	MaxAgeDays      int
	NoCache         bool // If true, bypass embedded cache and always fetch from API
}

// Validate checks if the configuration is valid
func (c CheckerConfig) Validate() error {
	if c.CriticalAgeDays < 0 {
		return fmt.Errorf("critical_age_days must be non-negative")
	}
	if c.MaxAgeDays < 0 {
		return fmt.Errorf("max_age_days must be non-negative")
	}
	// Skip validation if both are 0 (indicates version-based policy)
	if c.MaxAgeDays > 0 && c.CriticalAgeDays >= c.MaxAgeDays {
		return fmt.Errorf("critical_age_days must be less than max_age_days")
	}
	return nil
}

// Helper functions for JSON marshalling
func versionString(v *semver.Version) string {
	if v == nil {
		return ""
	}
	return v.String()
}

func timeString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
