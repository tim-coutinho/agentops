package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCouncilVerdictHeadingContract verifies that the wrapper skills used by
// extractCouncilVerdict (pre-mortem, vibe, post-mortem) each contain the
// exact heading "## Council Verdict:" that the CLI regex depends on.
//
// The regex in rpi_phased_processing.go is:
//
//	regexp.MustCompile(`(?m)^## Council Verdict:\s*(PASS|WARN|FAIL)`)
//
// If any wrapper skill is missing this heading, the CLI will fail to extract
// the verdict from council reports produced by those skills.
func TestCouncilVerdictHeadingContract(t *testing.T) {
	// Walk up from cli/cmd/ao to find the repo root (the directory containing skills/).
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	// Ascend until we find a skills/ directory or exhaust the path.
	repoRoot := ""
	dir := cwd
	for {
		candidate := filepath.Join(dir, "skills")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			repoRoot = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding skills/.
			break
		}
		dir = parent
	}

	if repoRoot == "" {
		t.Skip("skills/ directory not found in any ancestor of cwd; skipping contract test")
	}

	skillsDir := filepath.Join(repoRoot, "skills")

	wrapperSkills := []string{
		"pre-mortem",
		"vibe",
		"post-mortem",
	}

	const requiredHeading = "## Council Verdict:"

	for _, skill := range wrapperSkills {
		skill := skill // capture loop variable
		t.Run(skill, func(t *testing.T) {
			skillFile := filepath.Join(skillsDir, skill, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				t.Fatalf("could not read %s: %v", skillFile, err)
			}

			if !strings.Contains(string(data), requiredHeading) {
				t.Errorf(
					"%s/SKILL.md is missing the required heading %q\n"+
						"The CLI regex in extractCouncilVerdict depends on this heading being present.\n"+
						"Regex: `(?m)^## Council Verdict:\\s*(PASS|WARN|FAIL)`",
					skill, requiredHeading,
				)
			}
		})
	}
}
