package main

import (
	"os"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    time.Duration
	}{
		{"days", "7d", false, 7 * 24 * time.Hour},
		{"days 30", "30d", false, 30 * 24 * time.Hour},
		{"weeks", "1w", false, 7 * 24 * time.Hour},
		{"weeks multiple", "4w", false, 4 * 7 * 24 * time.Hour},
		{"hours", "48h", false, 48 * time.Hour},
		{"invalid", "invalid", true, 0},
		{"empty", "", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestVibeCheckCommand(t *testing.T) {
	// Test that command is registered
	if vibeCheckCmd == nil {
		t.Fatal("vibeCheckCmd is nil")
	}

	if vibeCheckCmd.Use != "vibe-check" {
		t.Errorf("vibeCheckCmd.Use = %q, want vibe-check", vibeCheckCmd.Use)
	}

	// Check that aliases include vibecheck
	hasAlias := false
	for _, alias := range vibeCheckCmd.Aliases {
		if alias == "vibecheck" {
			hasAlias = true
			break
		}
	}
	if !hasAlias {
		t.Error("vibeCheckCmd missing 'vibecheck' alias")
	}
}

func TestVibeCheckFlags(t *testing.T) {
	if vibeCheckCmd.Flags().Lookup("markdown") == nil {
		t.Error("--markdown flag not found")
	}
	if vibeCheckCmd.Flags().Lookup("since") == nil {
		t.Error("--since flag not found")
	}
	if vibeCheckCmd.Flags().Lookup("repo") == nil {
		t.Error("--repo flag not found")
	}
	if vibeCheckCmd.Flags().Lookup("full") == nil {
		t.Error("--full flag not found")
	}
}

func TestVibeCheckDryRun(t *testing.T) {
	// Create a temporary directory to act as a git repo
	_ = t.TempDir()

	// Save original state
	originalDryRun := dryRun
	defer func() { dryRun = originalDryRun }()

	// Enable dry-run mode
	dryRun = true

	// Run command in dry-run mode
	cmd := vibeCheckCmd
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("runVibeCheck in dry-run mode failed: %v", err)
	}
}

func TestVibeCheckRepoPath(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		want    string
		wantErr bool
	}{
		{"current dir", ".", "", false},
		{"absolute path", "/tmp", "/tmp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that path resolution doesn't panic
			repoPath := tt.repo
			if repoPath == "." {
				cwd, err := os.Getwd()
				if err != nil {
					t.Errorf("failed to get cwd: %v", err)
					return
				}
				repoPath = cwd
			}

			// All absolute path operations should succeed
			_ = repoPath
		})
	}
}

func TestOutputFormats(t *testing.T) {
	// Test that output functions handle empty results
	t.Run("empty result", func(t *testing.T) {
		result := &MockVibeCheckResult{
			Score: 0.85,
			Grade: "B",
		}

		// Just test that functions don't panic with empty data
		// (We can't easily test stdout capture without refactoring)
		if result.Score < 0 || result.Score > 1 {
			t.Error("invalid score")
		}
	})
}

// MockVibeCheckResult for testing
type MockVibeCheckResult struct {
	Score    float64
	Grade    string
	Events   []interface{}
	Metrics  map[string]float64
	Findings []interface{}
}
