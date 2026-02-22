package main

import (
	"os"
	"testing"
	"time"

	"github.com/boshu2/agentops/cli/internal/pool"
	"github.com/boshu2/agentops/cli/internal/types"
)

func TestPoolListEmpty(t *testing.T) {
	tmp := t.TempDir()
	p := pool.NewPool(tmp)

	entries, err := p.List(pool.ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries=%d, want 0", len(entries))
	}
}

func TestPoolStagePromoteWorkflow(t *testing.T) {
	tmp := t.TempDir()
	// Create learnings dir for promote target
	if err := os.MkdirAll(tmp+"/.agents/learnings", 0755); err != nil {
		t.Fatal(err)
	}

	p := pool.NewPool(tmp)

	cand := types.Candidate{
		ID:         "cand-test-001",
		Type:       "learning",
		Tier:       types.TierSilver,
		Content:    "Test learning content",
		Utility:    0.8,
		Confidence: 0.9,
		Maturity:   "established",
		Source: types.Source{
			SessionID:      "session-abc",
			TranscriptPath: "/tmp/t.md",
			MessageIndex:   5,
		},
	}

	if err := p.Add(cand, types.Scoring{RawScore: 0.85, TierAssignment: types.TierSilver}); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Verify candidate is pending
	entries, _ := p.List(pool.ListOptions{Status: types.PoolStatusPending})
	if len(entries) != 1 {
		t.Fatalf("pending=%d, want 1", len(entries))
	}

	// Stage
	if err := p.Stage(cand.ID, types.TierBronze); err != nil {
		t.Fatalf("stage: %v", err)
	}

	// Verify staged
	entries, _ = p.List(pool.ListOptions{Status: types.PoolStatusStaged})
	if len(entries) != 1 {
		t.Fatalf("staged=%d, want 1", len(entries))
	}

	// Promote
	artifactPath, err := p.Promote(cand.ID)
	if err != nil {
		t.Fatalf("promote: %v", err)
	}

	if _, err := os.Stat(artifactPath); err != nil {
		t.Errorf("artifact not created: %v", err)
	}
}

func TestPoolRejectRequiresCandidate(t *testing.T) {
	tmp := t.TempDir()
	p := pool.NewPool(tmp)

	// Reject a nonexistent candidate
	err := p.Reject("nonexistent-id", "test reason", "tester")
	if err == nil {
		t.Error("expected error rejecting nonexistent candidate")
	}
}

func TestPoolBulkApproveThreshold(t *testing.T) {
	tmp := t.TempDir()
	p := pool.NewPool(tmp)

	cand := types.Candidate{
		ID:         "cand-bulk-001",
		Type:       "learning",
		Tier:       types.TierSilver,
		Content:    "Bulk test content",
		Utility:    0.7,
		Confidence: 0.8,
		Maturity:   "emerging",
	}

	// Add with a past timestamp so the candidate qualifies for the 1h threshold
	pastTime := time.Now().Add(-2 * time.Hour)
	if err := p.AddAt(cand, types.Scoring{RawScore: 0.75, TierAssignment: types.TierSilver}, pastTime); err != nil {
		t.Fatalf("add: %v", err)
	}

	// BulkApprove with 1h threshold — candidate was added 2h ago so it qualifies
	approved, err := p.BulkApprove(time.Hour, "tester", false)
	if err != nil {
		t.Fatalf("bulk approve: %v", err)
	}
	if len(approved) != 1 {
		t.Errorf("approved=%d, want 1", len(approved))
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		max  int
		want string
	}{
		{"short id unchanged", "abc-123", 10, "abc-123"},
		{"exact length unchanged", "abc-123", 7, "abc-123"},
		{"truncated with ellipsis", "abcdefghij", 7, "abcd..."},
		{"single char max", "hello", 4, "h..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateID(tt.id, tt.max)
			if got != tt.want {
				t.Errorf("truncateID(%q, %d) = %q, want %q", tt.id, tt.max, got, tt.want)
			}
		})
	}
}

func TestRepeat(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"repeat 0 times", "abc", 0, ""},
		{"repeat 1 time", "abc", 1, "abc"},
		{"repeat 3 times", "ab", 3, "ababab"},
		{"empty string repeated", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repeat(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("repeat(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}
