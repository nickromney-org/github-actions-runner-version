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
