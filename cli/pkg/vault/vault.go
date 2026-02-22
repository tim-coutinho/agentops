// Package vault provides Obsidian vault detection utilities.
package vault

import (
	"os"
	"path/filepath"
)

// DetectVault walks up from the given directory to find an Obsidian vault.
// Returns the vault path if found, empty string otherwise.
func DetectVault(startDir string) string {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}

	dir := startDir
	for {
		obsidianPath := filepath.Join(dir, ".obsidian")
		if info, err := os.Stat(obsidianPath); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	return ""
}

// HasSmartConnections checks if Smart Connections plugin is installed in the vault.
func HasSmartConnections(vaultPath string) bool {
	if vaultPath == "" {
		return false
	}
	scPath := filepath.Join(vaultPath, ".obsidian", "plugins", "smart-connections")
	if _, err := os.Stat(scPath); err == nil {
		return true
	}
	return false
}

// IsInVault returns true if the given directory is within an Obsidian vault.
func IsInVault(dir string) bool {
	return DetectVault(dir) != ""
}
