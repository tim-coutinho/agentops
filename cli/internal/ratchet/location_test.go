package ratchet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchOrder(t *testing.T) {
	// Verify search order is crew → rig → town → plugins
	expected := []LocationType{
		LocationCrew,
		LocationRig,
		LocationTown,
		LocationPlugins,
	}

	if len(SearchOrder) != len(expected) {
		t.Errorf("SearchOrder has %d elements, want %d", len(SearchOrder), len(expected))
	}

	for i, loc := range expected {
		if SearchOrder[i] != loc {
			t.Errorf("SearchOrder[%d] = %s, want %s", i, SearchOrder[i], loc)
		}
	}
}

func TestNewLocator(t *testing.T) {
	tmpDir := t.TempDir()

	loc, err := NewLocator(tmpDir)
	if err != nil {
		t.Fatalf("NewLocator failed: %v", err)
	}

	if loc.startDir != tmpDir {
		t.Errorf("startDir = %s, want %s", loc.startDir, tmpDir)
	}

	if loc.home == "" {
		t.Error("home directory should not be empty")
	}

	if loc.townDir == "" {
		t.Error("townDir should not be empty")
	}
}

func TestLocator_FindFirst(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".agents", "research")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents dir: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(agentsDir, "test.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loc, err := NewLocator(tmpDir)
	if err != nil {
		t.Fatalf("NewLocator failed: %v", err)
	}

	// Should find the file
	path, locType, err := loc.FindFirst("research/test.md")
	if err != nil {
		t.Errorf("FindFirst failed: %v", err)
	}

	if path != testFile {
		t.Errorf("FindFirst path = %s, want %s", path, testFile)
	}

	if locType != LocationCrew {
		t.Errorf("FindFirst location = %s, want %s", locType, LocationCrew)
	}
}

func TestLocator_FindWithDuplicateWarning(t *testing.T) {
	// Create a temp directory structure simulating crew + rig
	tmpDir := t.TempDir()

	// Create "crew" level .agents
	crewAgents := filepath.Join(tmpDir, ".agents", "research")
	if err := os.MkdirAll(crewAgents, 0755); err != nil {
		t.Fatalf("Failed to create crew agents: %v", err)
	}
	crewFile := filepath.Join(crewAgents, "shared.md")
	if err := os.WriteFile(crewFile, []byte("# Crew"), 0644); err != nil {
		t.Fatalf("Failed to create crew file: %v", err)
	}

	// Create a rig marker so we can have rig-level .agents
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	loc, err := NewLocator(tmpDir)
	if err != nil {
		t.Fatalf("NewLocator failed: %v", err)
	}

	result, err := loc.Find("research/*.md")
	if err != nil {
		t.Errorf("Find failed: %v", err)
	}

	if len(result.Matches) == 0 {
		t.Error("Find should return at least one match")
	}

	// Verify pattern is set
	if result.Pattern != "research/*.md" {
		t.Errorf("Pattern = %s, want %s", result.Pattern, "research/*.md")
	}
}

func TestLocator_ArtifactExists(t *testing.T) {
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents dir: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(agentsDir, "exists.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loc, err := NewLocator(tmpDir)
	if err != nil {
		t.Fatalf("NewLocator failed: %v", err)
	}

	// File exists
	if !loc.ArtifactExists("exists.md") {
		t.Error("ArtifactExists should return true for existing file")
	}

	// File doesn't exist
	if loc.ArtifactExists("nonexistent.md") {
		t.Error("ArtifactExists should return false for non-existing file")
	}
}

func TestLocator_ResolveArtifactPath(t *testing.T) {
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".agents", "research")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents dir: %v", err)
	}

	testFile := filepath.Join(agentsDir, "topic.md")
	if err := os.WriteFile(testFile, []byte("# Topic"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loc, err := NewLocator(tmpDir)
	if err != nil {
		t.Fatalf("NewLocator failed: %v", err)
	}

	tests := []struct {
		name    string
		ref     string
		wantErr bool
	}{
		{"relative path", "research/topic.md", false},
		{"with .agents/ prefix", ".agents/research/topic.md", false},
		{"absolute path", testFile, false},
		{"glob pattern", "research/*.md", false},
		{"nonexistent", "nonexistent.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, locType, err := loc.ResolveArtifactPath(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveArtifactPath(%q) error = %v, wantErr %v", tt.ref, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if path == "" {
					t.Errorf("ResolveArtifactPath(%q) returned empty path", tt.ref)
				}
				if locType == "" {
					t.Errorf("ResolveArtifactPath(%q) returned empty location", tt.ref)
				}
			}
		})
	}
}

func TestLocator_GetLocationPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a rig marker
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	// Create plugins dir
	if err := os.MkdirAll(filepath.Join(tmpDir, "plugins"), 0755); err != nil {
		t.Fatalf("Failed to create plugins dir: %v", err)
	}

	loc, err := NewLocator(tmpDir)
	if err != nil {
		t.Fatalf("NewLocator failed: %v", err)
	}

	paths := loc.GetLocationPaths()

	// Should have crew path
	if _, ok := paths[LocationCrew]; !ok {
		t.Error("GetLocationPaths should include crew")
	}

	// Should have plugins path (since we created it)
	if _, ok := paths[LocationPlugins]; !ok {
		t.Error("GetLocationPaths should include plugins")
	}

	// Should have town path
	if _, ok := paths[LocationTown]; !ok {
		t.Error("GetLocationPaths should include town")
	}
}

func TestLocationTypeValues(t *testing.T) {
	// Verify location type string values
	if LocationCrew != "crew" {
		t.Errorf("LocationCrew = %s, want crew", LocationCrew)
	}
	if LocationRig != "rig" {
		t.Errorf("LocationRig = %s, want rig", LocationRig)
	}
	if LocationTown != "town" {
		t.Errorf("LocationTown = %s, want town", LocationTown)
	}
	if LocationPlugins != "plugins" {
		t.Errorf("LocationPlugins = %s, want plugins", LocationPlugins)
	}
}
