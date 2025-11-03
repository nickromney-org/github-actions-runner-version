package config

import (
	"testing"
)

func TestGetPredefinedConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "actions-runner",
			input:     "actions-runner",
			wantOwner: "actions",
			wantRepo:  "runner",
			wantErr:   false,
		},
		{
			name:      "github-runner alias",
			input:     "github-runner",
			wantOwner: "actions",
			wantRepo:  "runner",
			wantErr:   false,
		},
		{
			name:      "runner alias",
			input:     "runner",
			wantOwner: "actions",
			wantRepo:  "runner",
			wantErr:   false,
		},
		{
			name:      "kubernetes",
			input:     "kubernetes",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
			wantErr:   false,
		},
		{
			name:      "k8s alias",
			input:     "k8s",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
			wantErr:   false,
		},
		{
			name:      "pulumi",
			input:     "pulumi",
			wantOwner: "pulumi",
			wantRepo:  "pulumi",
			wantErr:   false,
		},
		{
			name:      "ubuntu",
			input:     "ubuntu",
			wantOwner: "canonical",
			wantRepo:  "ubuntu",
			wantErr:   false,
		},
		{
			name:    "unknown",
			input:   "unknown-repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GetPredefinedConfig(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if config.Owner != tt.wantOwner {
				t.Errorf("Owner = %v, want %v", config.Owner, tt.wantOwner)
			}

			if config.Repo != tt.wantRepo {
				t.Errorf("Repo = %v, want %v", config.Repo, tt.wantRepo)
			}
		})
	}
}

func TestParseRepositoryString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "simple owner/repo",
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "predefined config",
			input:     "kubernetes",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
			wantErr:   false,
		},
		{
			name:      "GitHub URL",
			input:     "https://github.com/kubernetes/kubernetes",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
			wantErr:   false,
		},
		{
			name:      "GitHub URL with trailing slash",
			input:     "https://github.com/pulumi/pulumi/",
			wantOwner: "pulumi",
			wantRepo:  "pulumi",
			wantErr:   false,
		},
		{
			name:      "GitHub URL with releases",
			input:     "https://github.com/kubernetes/kubernetes/releases",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
			wantErr:   false,
		},
		{
			name:      "GitHub URL with tag",
			input:     "https://github.com/kubernetes/kubernetes/releases/tag/v1.32.0",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
			wantErr:   false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "too many parts",
			input:   "owner/repo/extra",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseRepositoryString(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if config.Owner != tt.wantOwner {
				t.Errorf("Owner = %v, want %v", config.Owner, tt.wantOwner)
			}

			if config.Repo != tt.wantRepo {
				t.Errorf("Repo = %v, want %v", config.Repo, tt.wantRepo)
			}
		})
	}
}

func TestRepositoryConfig_FullName(t *testing.T) {
	config := &RepositoryConfig{
		Owner: "kubernetes",
		Repo:  "kubernetes",
	}

	got := config.FullName()
	want := "kubernetes/kubernetes"

	if got != want {
		t.Errorf("FullName() = %v, want %v", got, want)
	}
}

func TestPredefinedConfigs(t *testing.T) {
	tests := []struct {
		name         string
		config       RepositoryConfig
		wantPolicy   PolicyType
		wantCache    bool
	}{
		{
			name:       "actions-runner uses days policy",
			config:     ConfigActionsRunner,
			wantPolicy: PolicyTypeDays,
			wantCache:  true,
		},
		{
			name:       "kubernetes uses versions policy",
			config:     ConfigKubernetes,
			wantPolicy: PolicyTypeVersions,
			wantCache:  false, // Will be enabled in Phase 3.1
		},
		{
			name:       "pulumi uses versions policy",
			config:     ConfigPulumi,
			wantPolicy: PolicyTypeVersions,
			wantCache:  false, // Will be enabled in Phase 3.1
		},
		{
			name:       "ubuntu uses days policy",
			config:     ConfigUbuntu,
			wantPolicy: PolicyTypeDays,
			wantCache:  false, // Will be enabled in Phase 3.1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.PolicyType != tt.wantPolicy {
				t.Errorf("PolicyType = %v, want %v", tt.config.PolicyType, tt.wantPolicy)
			}

			if tt.config.CacheEnabled != tt.wantCache {
				t.Errorf("CacheEnabled = %v, want %v", tt.config.CacheEnabled, tt.wantCache)
			}

			if tt.config.CachePath == "" {
				t.Error("CachePath is empty")
			}
		})
	}
}
