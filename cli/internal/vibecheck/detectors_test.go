package vibecheck

import (
	"testing"
	"time"
)

// makeEvent is a helper to create test TimelineEvents.
func makeEvent(sha string, minutesOffset int, msg string, files []string, ins, del int) TimelineEvent {
	base := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	return TimelineEvent{
		SHA:          sha,
		Timestamp:    base.Add(time.Duration(minutesOffset) * time.Minute),
		Author:       "Test Author",
		Message:      msg,
		FilesChanged: len(files),
		Files:        files,
		Insertions:   ins,
		Deletions:    del,
	}
}

// --- Tests Passing Lie ---

func TestDetectTestsLie_ClaimThenFix(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: auth working now", []string{"auth.go"}, 10, 2),
		makeEvent("bbb", 15, "fix: auth edge case", []string{"auth.go"}, 3, 1),
	}

	findings := DetectTestsLie(events)
	if len(findings) == 0 {
		t.Fatal("expected at least one finding for claim+fix pattern")
	}
	if findings[0].Category != "tests-passing-lie" {
		t.Errorf("expected category tests-passing-lie, got %s", findings[0].Category)
	}
	if findings[0].Severity != "critical" {
		t.Errorf("expected severity critical, got %s", findings[0].Severity)
	}
}

func TestDetectTestsLie_TentativeNotFlagged(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "WIP: trying auth working", []string{"auth.go"}, 10, 2),
		makeEvent("bbb", 15, "fix: auth edge case", []string{"auth.go"}, 3, 1),
	}

	findings := DetectTestsLie(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for tentative commit, got %d", len(findings))
	}
}

func TestDetectTestsLie_DifferentFilesNoLie(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: auth done", []string{"auth.go"}, 10, 2),
		makeEvent("bbb", 15, "fix: unrelated bug", []string{"other.go"}, 3, 1),
	}

	findings := DetectTestsLie(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for different files, got %d", len(findings))
	}
}

func TestDetectTestsLie_OutsideWindow(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: all tests pass", []string{"auth.go"}, 10, 2),
		makeEvent("bbb", 60, "fix: auth bug", []string{"auth.go"}, 3, 1),
	}

	findings := DetectTestsLie(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings outside 30min window, got %d", len(findings))
	}
}

func TestDetectTestsLie_TooFewCommits(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: all tests pass", []string{"auth.go"}, 10, 2),
	}

	findings := DetectTestsLie(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for single commit, got %d", len(findings))
	}
}

// --- Context Amnesia ---

func TestDetectContextAmnesia_RepeatedEdits(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: add handler", []string{"handler.go"}, 20, 0),
		makeEvent("bbb", 10, "fix: handler null check", []string{"handler.go"}, 5, 2),
		makeEvent("ccc", 20, "fix: handler again", []string{"handler.go"}, 5, 3),
		makeEvent("ddd", 30, "fix: handler edge case", []string{"handler.go"}, 4, 2),
	}

	findings := DetectContextAmnesia(events)
	if len(findings) == 0 {
		t.Fatal("expected findings for repeated edits to same file")
	}
	if findings[0].Category != "context-amnesia" {
		t.Errorf("expected category context-amnesia, got %s", findings[0].Category)
	}
	if findings[0].File != "handler.go" {
		t.Errorf("expected file handler.go, got %s", findings[0].File)
	}
}

func TestDetectContextAmnesia_DifferentFiles(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: add handler", []string{"handler.go"}, 20, 0),
		makeEvent("bbb", 10, "feat: add router", []string{"router.go"}, 15, 0),
		makeEvent("ccc", 20, "feat: add service", []string{"service.go"}, 25, 0),
	}

	findings := DetectContextAmnesia(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for different files, got %d", len(findings))
	}
}

func TestDetectContextAmnesia_OutsideWindow(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: add handler", []string{"handler.go"}, 20, 0),
		makeEvent("bbb", 90, "fix: handler", []string{"handler.go"}, 5, 2),
		makeEvent("ccc", 180, "fix: handler again", []string{"handler.go"}, 5, 3),
	}

	findings := DetectContextAmnesia(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings outside 1hr window, got %d", len(findings))
	}
}

func TestDetectContextAmnesia_TooFewEvents(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: add handler", []string{"handler.go"}, 20, 0),
		makeEvent("bbb", 10, "fix: handler", []string{"handler.go"}, 5, 2),
	}

	findings := DetectContextAmnesia(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for too few events, got %d", len(findings))
	}
}

// --- Instruction Drift ---

func TestDetectInstructionDrift_RepeatedConfigEdits(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "chore: update CLAUDE.md", []string{"CLAUDE.md"}, 5, 2),
		makeEvent("bbb", 10, "chore: fix CLAUDE.md", []string{"CLAUDE.md"}, 3, 1),
		makeEvent("ccc", 20, "chore: tweak CLAUDE.md", []string{"CLAUDE.md"}, 2, 1),
	}

	findings := DetectInstructionDrift(events)
	if len(findings) == 0 {
		t.Fatal("expected findings for repeated CLAUDE.md edits")
	}
	if findings[0].Category != "instruction-drift" {
		t.Errorf("expected category instruction-drift, got %s", findings[0].Category)
	}
}

func TestDetectInstructionDrift_SKILLmd(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "chore: update skill", []string{"skills/vibe/SKILL.md"}, 5, 2),
		makeEvent("bbb", 10, "chore: fix skill", []string{"skills/vibe/SKILL.md"}, 3, 1),
		makeEvent("ccc", 20, "chore: tweak skill", []string{"skills/vibe/SKILL.md"}, 2, 1),
	}

	findings := DetectInstructionDrift(events)
	if len(findings) == 0 {
		t.Fatal("expected findings for repeated SKILL.md edits")
	}
}

func TestDetectInstructionDrift_NormalCode(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: add handler", []string{"handler.go"}, 20, 0),
		makeEvent("bbb", 10, "fix: handler", []string{"handler.go"}, 5, 2),
		makeEvent("ccc", 20, "feat: add router", []string{"router.go"}, 15, 0),
	}

	findings := DetectInstructionDrift(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for normal code, got %d", len(findings))
	}
}

func TestDetectInstructionDrift_BelowThreshold(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "chore: update CLAUDE.md", []string{"CLAUDE.md"}, 5, 2),
		makeEvent("bbb", 10, "feat: add handler", []string{"handler.go"}, 20, 0),
	}

	findings := DetectInstructionDrift(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings below threshold, got %d", len(findings))
	}
}

// --- Logging Only ---

func TestDetectLoggingOnly_ConsecutiveDebug(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "fix: add debug logging", nil, 5, 0),
		makeEvent("bbb", 5, "fix: more console.log", nil, 3, 0),
		makeEvent("ccc", 10, "fix: temp debug trace", nil, 4, 0),
	}

	findings := DetectLoggingOnly(events)
	if len(findings) == 0 {
		t.Fatal("expected findings for consecutive logging commits")
	}
	if findings[0].Category != "logging-only" {
		t.Errorf("expected category logging-only, got %s", findings[0].Category)
	}
}

func TestDetectLoggingOnly_InterruptedByFeature(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "fix: add debug logging", nil, 5, 0),
		makeEvent("bbb", 5, "fix: console.log", nil, 3, 0),
		makeEvent("ccc", 10, "feat: add new endpoint", nil, 50, 0),
		makeEvent("ddd", 15, "fix: temp debug", nil, 4, 0),
	}

	findings := DetectLoggingOnly(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings when streak interrupted, got %d", len(findings))
	}
}

func TestDetectLoggingOnly_LargeDiffNotFlagged(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "fix: add debug logging", nil, 50, 10),
		makeEvent("bbb", 5, "fix: more debug logging", nil, 40, 5),
		makeEvent("ccc", 10, "fix: logging investigation", nil, 60, 20),
	}

	findings := DetectLoggingOnly(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for large diffs, got %d", len(findings))
	}
}

func TestDetectLoggingOnly_HealthyBaseline(t *testing.T) {
	events := []TimelineEvent{
		makeEvent("aaa", 0, "feat: add handler", nil, 30, 0),
		makeEvent("bbb", 10, "feat: add router", nil, 25, 0),
		makeEvent("ccc", 20, "test: add unit tests", nil, 40, 0),
	}

	findings := DetectLoggingOnly(events)
	if len(findings) != 0 {
		t.Errorf("expected no findings for healthy baseline, got %d", len(findings))
	}
}

// --- Aggregator ---

func TestDetectRunDetectors_AggregatesAll(t *testing.T) {
	events := []TimelineEvent{
		// Tests lie pattern: claim + fix.
		makeEvent("a1", 0, "feat: auth working now", []string{"auth.go"}, 10, 2),
		makeEvent("a2", 15, "fix: auth edge case", []string{"auth.go"}, 3, 1),
		// Logging pattern (3 consecutive small debug commits).
		makeEvent("b1", 30, "fix: add debug logging", nil, 5, 0),
		makeEvent("b2", 35, "fix: more console.log", nil, 3, 0),
		makeEvent("b3", 40, "fix: temp debug trace", nil, 4, 0),
	}

	findings := RunDetectors(events)
	if len(findings) < 2 {
		t.Errorf("expected at least 2 findings from aggregation, got %d", len(findings))
	}

	// Check that we have findings from multiple categories.
	cats := make(map[string]bool)
	for _, f := range findings {
		cats[f.Category] = true
	}
	if !cats["tests-passing-lie"] {
		t.Error("expected tests-passing-lie finding")
	}
	if !cats["logging-only"] {
		t.Error("expected logging-only finding")
	}
}

func TestDetectClassifyHealth_Critical(t *testing.T) {
	findings := []Finding{
		{Severity: "critical", Category: "tests-passing-lie", Message: "lie detected"},
		{Severity: "warning", Category: "logging-only", Message: "debug spiral"},
	}

	health := ClassifyHealth(findings)
	if health != "critical" {
		t.Errorf("expected critical, got %s", health)
	}
}

func TestDetectClassifyHealth_Warning(t *testing.T) {
	findings := []Finding{
		{Severity: "warning", Category: "logging-only", Message: "debug spiral"},
	}

	health := ClassifyHealth(findings)
	if health != "warning" {
		t.Errorf("expected warning, got %s", health)
	}
}

func TestDetectClassifyHealth_Healthy(t *testing.T) {
	findings := []Finding{}

	health := ClassifyHealth(findings)
	if health != "healthy" {
		t.Errorf("expected healthy, got %s", health)
	}
}
