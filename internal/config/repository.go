package config

import (
	"fmt"
	"strings"
)

// PolicyType defines the type of expiry policy
type PolicyType string

const (
	PolicyTypeDays     PolicyType = "days"     // Time-based: expires after N days
	PolicyTypeVersions PolicyType = "versions" // Version-based: expires after N minor versions
)

// RepositoryConfig defines a GitHub repository and its version policy
type RepositoryConfig struct {
	Owner string // GitHub owner (e.g., "actions", "kubernetes")
	Repo  string // GitHub repo (e.g., "runner", "kubernetes")

	// Policy configuration
	PolicyType        PolicyType
	CriticalDays      int // For PolicyTypeDays
	MaxDays           int // For PolicyTypeDays
	MaxVersionsBehind int // For PolicyTypeVersions

	// Cache configuration
	CachePath    string // Path to embedded cache file
	CacheEnabled bool   // Whether to use embedded cache
}

// Predefined repository configurations
var (
	ConfigActionsRunner = RepositoryConfig{
		Owner:        "actions",
		Repo:         "runner",
		PolicyType:   PolicyTypeDays,
		CriticalDays: 12,
		MaxDays:      30,
		CachePath:    "data/actions-runner.json",
		CacheEnabled: true,
	}

	ConfigKubernetes = RepositoryConfig{
		Owner:             "kubernetes",
		Repo:              "kubernetes",
		PolicyType:        PolicyTypeVersions,
		MaxVersionsBehind: 3, // Support last 3 minor versions
		CachePath:         "data/kubernetes.json",
		CacheEnabled:      false, // Will be enabled when cache is created
	}

	ConfigPulumi = RepositoryConfig{
		Owner:             "pulumi",
		Repo:              "pulumi",
		PolicyType:        PolicyTypeVersions,
		MaxVersionsBehind: 3,
		CachePath:         "data/pulumi.json",
		CacheEnabled:      false, // Will be enabled when cache is created
	}

	ConfigUbuntu = RepositoryConfig{
		Owner:        "canonical",
		Repo:         "ubuntu",
		PolicyType:   PolicyTypeDays,
		CriticalDays: 180, // 6 months
		MaxDays:      365, // 1 year
		CachePath:    "data/ubuntu.json",
		CacheEnabled: false, // Will be enabled when cache is created
	}
)

// GetPredefinedConfig returns a predefined config by name
func GetPredefinedConfig(name string) (*RepositoryConfig, error) {
	configs := map[string]RepositoryConfig{
		"actions-runner": ConfigActionsRunner,
		"github-runner":  ConfigActionsRunner, // Alias
		"runner":         ConfigActionsRunner, // Alias
		"kubernetes":     ConfigKubernetes,
		"k8s":            ConfigKubernetes, // Alias
		"pulumi":         ConfigPulumi,
		"ubuntu":         ConfigUbuntu,
	}

	config, ok := configs[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown repository: %s", name)
	}
	return &config, nil
}

// ParseRepositoryString parses "owner/repo" format or URL
func ParseRepositoryString(repoStr string) (*RepositoryConfig, error) {
	// Try to parse as predefined config first
	if config, err := GetPredefinedConfig(repoStr); err == nil {
		return config, nil
	}

	// Check if it's a GitHub URL
	if strings.Contains(repoStr, "github.com") {
		// Extract owner/repo from URL
		// https://github.com/owner/repo -> owner/repo
		parts := strings.Split(repoStr, "github.com/")
		if len(parts) == 2 {
			repoStr = strings.TrimSuffix(parts[1], "/")
			repoStr = strings.Split(repoStr, "/releases")[0]
			repoStr = strings.Split(repoStr, "/tags")[0]
		}
	}

	// Parse as owner/repo
	parts := strings.Split(repoStr, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s (expected: owner/repo or predefined name)", repoStr)
	}

	// Default to version-based policy with conservative defaults
	return &RepositoryConfig{
		Owner:             parts[0],
		Repo:              parts[1],
		PolicyType:        PolicyTypeVersions,
		MaxVersionsBehind: 3,
		CacheEnabled:      false, // No embedded cache for custom repos
	}, nil
}

// FullName returns the full repository name (owner/repo)
func (c *RepositoryConfig) FullName() string {
	return fmt.Sprintf("%s/%s", c.Owner, c.Repo)
}
