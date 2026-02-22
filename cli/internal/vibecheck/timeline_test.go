package vibecheck

import (
	"testing"
	"time"
)

func TestParseTimeline_Format(t *testing.T) {
	// Simulate git log --format="%H|||%aI|||%an|||%s" --numstat output.
	const delim = "|||"
	raw := `abc123|||2026-02-15T10:00:00-05:00|||Alice|||feat: add vibecheck types
3	1	cli/internal/vibecheck/types.go
2	0	cli/internal/vibecheck/timeline.go

def456|||2026-02-15T09:30:00-05:00|||Bob|||fix: correct timestamp parsing
1	1	cli/internal/vibecheck/timeline.go

`

	events, err := parseGitLog(raw, delim)
	if err != nil {
		t.Fatalf("parseGitLog returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Events should be sorted newest first.
	first := events[0]
	if first.SHA != "abc123" {
		t.Errorf("expected first event SHA abc123, got %s", first.SHA)
	}
	if first.Author != "Alice" {
		t.Errorf("expected author Alice, got %s", first.Author)
	}
	if first.Message != "feat: add vibecheck types" {
		t.Errorf("expected message 'feat: add vibecheck types', got %q", first.Message)
	}
	if first.FilesChanged != 2 {
		t.Errorf("expected 2 files changed, got %d", first.FilesChanged)
	}
	if first.Insertions != 5 {
		t.Errorf("expected 5 insertions, got %d", first.Insertions)
	}
	if first.Deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", first.Deletions)
	}

	second := events[1]
	if second.SHA != "def456" {
		t.Errorf("expected second event SHA def456, got %s", second.SHA)
	}
	if second.FilesChanged != 1 {
		t.Errorf("expected 1 file changed, got %d", second.FilesChanged)
	}
	if second.Insertions != 1 {
		t.Errorf("expected 1 insertion, got %d", second.Insertions)
	}
	if second.Deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", second.Deletions)
	}
}

func TestTimelineEvent_Fields(t *testing.T) {
	now := time.Now()
	event := TimelineEvent{
		Timestamp:    now,
		SHA:          "abc123def456",
		Author:       "Test Author",
		Message:      "feat: test message",
		FilesChanged: 3,
		Insertions:   10,
		Deletions:    5,
		Tags:         []string{"v1.0.0", "latest"},
	}

	if event.Timestamp != now {
		t.Error("Timestamp mismatch")
	}
	if event.SHA != "abc123def456" {
		t.Error("SHA mismatch")
	}
	if event.Author != "Test Author" {
		t.Error("Author mismatch")
	}
	if event.Message != "feat: test message" {
		t.Error("Message mismatch")
	}
	if event.FilesChanged != 3 {
		t.Error("FilesChanged mismatch")
	}
	if event.Insertions != 10 {
		t.Error("Insertions mismatch")
	}
	if event.Deletions != 5 {
		t.Error("Deletions mismatch")
	}
	if len(event.Tags) != 2 || event.Tags[0] != "v1.0.0" || event.Tags[1] != "latest" {
		t.Error("Tags mismatch")
	}
}

func TestParseTimeline_EmptyOutput(t *testing.T) {
	events, err := parseGitLog("", "|||")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty input, got %d", len(events))
	}
}

func TestParseTimeline_NoTrailingNewline(t *testing.T) {
	// Git log output without trailing blank line.
	raw := `aaa111|||2026-02-15T08:00:00-05:00|||Carol|||chore: cleanup
1	0	README.md`

	events, err := parseGitLog(raw, "|||")
	if err != nil {
		t.Fatalf("parseGitLog returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].SHA != "aaa111" {
		t.Errorf("expected SHA aaa111, got %s", events[0].SHA)
	}
	if events[0].FilesChanged != 1 {
		t.Errorf("expected 1 file changed, got %d", events[0].FilesChanged)
	}
}
