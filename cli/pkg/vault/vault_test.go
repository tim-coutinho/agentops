package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectVault(t *testing.T) {
	// Create temp dir structure
	tmpDir := t.TempDir()

	// No vault case
	if got := DetectVault(tmpDir); got != "" {
		t.Errorf("DetectVault() = %q, want empty string", got)
	}

	// Create .obsidian directory to simulate vault
	vaultDir := filepath.Join(tmpDir, "my-vault")
	obsidianDir := filepath.Join(vaultDir, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Detect from vault root
	if got := DetectVault(vaultDir); got != vaultDir {
		t.Errorf("DetectVault(%q) = %q, want %q", vaultDir, got, vaultDir)
	}

	// Detect from subdirectory
	subDir := filepath.Join(vaultDir, "notes", "daily")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if got := DetectVault(subDir); got != vaultDir {
		t.Errorf("DetectVault(%q) = %q, want %q", subDir, got, vaultDir)
	}
}

func TestHasSmartConnections(t *testing.T) {
	tmpDir := t.TempDir()

	// No vault
	if HasSmartConnections(tmpDir) {
		t.Error("HasSmartConnections() = true, want false for non-vault")
	}

	// Empty string
	if HasSmartConnections("") {
		t.Error("HasSmartConnections(\"\") = true, want false")
	}

	// Vault without SC
	vaultDir := filepath.Join(tmpDir, "vault")
	obsidianDir := filepath.Join(vaultDir, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0755); err != nil {
		t.Fatal(err)
	}
	if HasSmartConnections(vaultDir) {
		t.Error("HasSmartConnections() = true, want false without SC plugin")
	}

	// Vault with SC
	scDir := filepath.Join(obsidianDir, "plugins", "smart-connections")
	if err := os.MkdirAll(scDir, 0755); err != nil {
		t.Fatal(err)
	}
	if !HasSmartConnections(vaultDir) {
		t.Error("HasSmartConnections() = false, want true with SC plugin")
	}
}

func TestDetectVault_EmptyString(t *testing.T) {
	// Empty string should use current working directory (os.Getwd)
	// and walk upward. We don't control cwd, but it should not panic.
	result := DetectVault("")
	// Result depends on whether we're inside an Obsidian vault.
	// Just verify no panic and it returns a string.
	_ = result
}

func TestIsInVault(t *testing.T) {
	tmpDir := t.TempDir()

	if IsInVault(tmpDir) {
		t.Error("IsInVault() = true, want false")
	}

	vaultDir := filepath.Join(tmpDir, "vault")
	if err := os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755); err != nil {
		t.Fatal(err)
	}

	if !IsInVault(vaultDir) {
		t.Error("IsInVault() = false, want true")
	}
}
