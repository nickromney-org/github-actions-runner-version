package types

import (
	"time"

	"github.com/Masterminds/semver/v3"
)

// Release represents a GitHub release
type Release struct {
	Version     *semver.Version
	PublishedAt time.Time
	URL         string
}
