package vibecheck

import "regexp"

// configFilePatterns matches files that are instructions or configuration.
var configFilePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)CLAUDE\.md$`),
	regexp.MustCompile(`(?i)SKILL\.md$`),
	regexp.MustCompile(`(?i)\.claude/`),
	regexp.MustCompile(`(?i)\.github/`),
	regexp.MustCompile(`(?i)\.agents/`),
	regexp.MustCompile(`(?i)tsconfig\.json$`),
	regexp.MustCompile(`(?i)\.eslintrc`),
	regexp.MustCompile(`(?i)\.prettierrc`),
	regexp.MustCompile(`(?i)renovate\.json`),
}

// driftMinEdits is the minimum number of config-file edits to trigger a finding.
const driftMinEdits = 3

// isConfigFile returns true if the path matches any known config/instruction file pattern.
func isConfigFile(path string) bool {
	for _, p := range configFilePatterns {
		if p.MatchString(path) {
			return true
		}
	}
	return false
}

// countConfigEdits tallies how many commits touch each config file.
func countConfigEdits(events []TimelineEvent) map[string]int {
	fileCounts := make(map[string]int)
	for _, ev := range events {
		for _, f := range ev.Files {
			if isConfigFile(f) {
				fileCounts[f]++
			}
		}
	}
	return fileCounts
}

// DetectInstructionDrift detects commits that repeatedly modify instruction
// or config files (CLAUDE.md, SKILL.md, etc.), suggesting instructions are
// being changed too often instead of stabilizing.
func DetectInstructionDrift(events []TimelineEvent) []Finding {
	fileCounts := countConfigEdits(events)

	var findings []Finding
	for file, count := range fileCounts {
		if count >= driftMinEdits {
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				Category: "instruction-drift",
				Message:  file + " modified " + itoa(count) + " times, instructions may be drifting",
				File:     file,
			})
		}
	}
	return findings
}
