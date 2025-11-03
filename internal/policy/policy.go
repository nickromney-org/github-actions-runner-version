package policy

import (
	"github.com/nickromney-org/github-actions-runner-version/internal/config"
	"github.com/nickromney-org/github-actions-runner-version/pkg/policy"
)

// NewPolicy creates a policy from config
// This is an internal adapter that uses the public pkg/policy package
func NewPolicy(repoConfig *config.RepositoryConfig) policy.VersionPolicy {
	switch repoConfig.PolicyType {
	case config.PolicyTypeDays:
		return policy.NewDaysPolicy(repoConfig.CriticalDays, repoConfig.MaxDays)
	case config.PolicyTypeVersions:
		return policy.NewVersionsPolicy(repoConfig.MaxVersionsBehind)
	default:
		// Default to days-based
		return policy.NewDaysPolicy(12, 30)
	}
}
