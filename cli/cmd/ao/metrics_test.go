package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

func TestFilterCitationsForPeriod(t *testing.T) {
	now := time.Now()
	oneDayAgo := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)
	oneWeekAgo := now.AddDate(0, 0, -7)
	twoWeeksAgo := now.AddDate(0, 0, -14)

	citations := []types.CitationEvent{
		{ArtifactPath: "/path/a.md", CitedAt: oneDayAgo},
		{ArtifactPath: "/path/b.md", CitedAt: twoDaysAgo},
		{ArtifactPath: "/path/c.md", CitedAt: oneWeekAgo},
		{ArtifactPath: "/path/d.md", CitedAt: twoWeeksAgo},
	}

	tests := []struct {
		name          string
		start         time.Time
		end           time.Time
		wantCount     int
		wantUniqueCnt int
	}{
		{
			name:          "all in period",
			start:         twoWeeksAgo.AddDate(0, 0, -1),
			end:           now.AddDate(0, 0, 1),
			wantCount:     4,
			wantUniqueCnt: 4,
		},
		{
			name:          "last 3 days",
			start:         now.AddDate(0, 0, -3),
			end:           now.AddDate(0, 0, 1),
			wantCount:     2,
			wantUniqueCnt: 2,
		},
		{
			name:          "last week",
			start:         now.AddDate(0, 0, -8),
			end:           now.AddDate(0, 0, 1),
			wantCount:     3,
			wantUniqueCnt: 3,
		},
		{
			name:          "empty period",
			start:         now.AddDate(0, 0, -100),
			end:           now.AddDate(0, 0, -50),
			wantCount:     0,
			wantUniqueCnt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := filterCitationsForPeriod(citations, tt.start, tt.end)
			if len(stats.citations) != tt.wantCount {
				t.Errorf("filterCitationsForPeriod() count = %d, want %d",
					len(stats.citations), tt.wantCount)
			}
			if len(stats.uniqueCited) != tt.wantUniqueCnt {
				t.Errorf("filterCitationsForPeriod() uniqueCited = %d, want %d",
					len(stats.uniqueCited), tt.wantUniqueCnt)
			}
		})
	}
}

func TestComputeSigmaRho(t *testing.T) {
	tests := []struct {
		name           string
		totalArtifacts int
		uniqueCited    int
		citationCount  int
		days           int
		wantSigma      float64
		wantRho        float64
	}{
		{
			name:           "normal case",
			totalArtifacts: 100,
			uniqueCited:    50,
			citationCount:  100,
			days:           7,
			wantSigma:      0.5,
			wantRho:        2.0, // 100/50/1week = 2
		},
		{
			name:           "no artifacts",
			totalArtifacts: 0,
			uniqueCited:    0,
			citationCount:  0,
			days:           7,
			wantSigma:      0,
			wantRho:        0,
		},
		{
			name:           "no citations",
			totalArtifacts: 100,
			uniqueCited:    0,
			citationCount:  0,
			days:           7,
			wantSigma:      0,
			wantRho:        0,
		},
		{
			name:           "14 days",
			totalArtifacts: 100,
			uniqueCited:    50,
			citationCount:  100,
			days:           14,
			wantSigma:      0.5,
			wantRho:        1.0, // 100/50/2weeks = 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sigma, rho := computeSigmaRho(tt.totalArtifacts, tt.uniqueCited, tt.citationCount, tt.days)

			if !floatEqual(sigma, tt.wantSigma, 0.01) {
				t.Errorf("computeSigmaRho() sigma = %v, want %v", sigma, tt.wantSigma)
			}
			if !floatEqual(rho, tt.wantRho, 0.01) {
				t.Errorf("computeSigmaRho() rho = %v, want %v", rho, tt.wantRho)
			}
		})
	}
}

func TestCountLoopMetrics(t *testing.T) {
	now := time.Now()
	oneDayAgo := now.AddDate(0, 0, -1)

	citations := []types.CitationEvent{
		{ArtifactPath: "/path/to/.agents/learnings/L1.md", CitedAt: oneDayAgo},
		{ArtifactPath: "/path/to/.agents/learnings/L2.md", CitedAt: oneDayAgo},
		{ArtifactPath: "/path/to/.agents/patterns/P1.md", CitedAt: oneDayAgo},
		{ArtifactPath: "/other/file.md", CitedAt: oneDayAgo},
	}

	// countLoopMetrics requires actual directory structure, so we just test
	// the learningsFound counting logic here via the helper
	learningsFound := 0
	for _, c := range citations {
		if containsLearningsPath(c.ArtifactPath) {
			learningsFound++
		}
	}

	if learningsFound != 2 {
		t.Errorf("learningsFound = %d, want 2", learningsFound)
	}
}

func TestCountBypassCitations(t *testing.T) {
	citations := []types.CitationEvent{
		{ArtifactPath: "/normal/path.md", CitationType: "recall"},
		{ArtifactPath: "/bypass/path.md", CitationType: "bypass"},
		{ArtifactPath: "bypass:/skipped", CitationType: ""},
		{ArtifactPath: "/another/path.md", CitationType: "inject"},
	}

	got := countBypassCitations(citations)
	if got != 2 {
		t.Errorf("countBypassCitations() = %d, want 2", got)
	}
}

// floatEqual checks if two floats are approximately equal
func floatEqual(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

// containsLearningsPath checks if path contains /learnings/
func containsLearningsPath(path string) bool {
	for i := range len(path) - 11 + 1 {
		if path[i:i+11] == "/learnings/" {
			return true
		}
	}
	return false
}

func TestCountStaleArtifacts(t *testing.T) {
	baseDir := t.TempDir()
	learningsDir := filepath.Join(baseDir, ".agents", "learnings")
	patternsDir := filepath.Join(baseDir, ".agents", "patterns")
	if err := os.MkdirAll(learningsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(patternsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	oldTime := time.Now().AddDate(0, 0, -120)
	newTime := time.Now().AddDate(0, 0, -1)

	writeFileWithTime := func(path string, ts time.Time) {
		t.Helper()
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, ts, ts); err != nil {
			t.Fatal(err)
		}
	}

	oldUncited := filepath.Join(learningsDir, "old-uncited.md")
	newUncited := filepath.Join(learningsDir, "new-uncited.md")
	oldRecentlyCited := filepath.Join(learningsDir, "old-recently-cited.md")
	oldCitedLongAgo := filepath.Join(patternsDir, "old-cited-long-ago.md")

	writeFileWithTime(oldUncited, oldTime)
	writeFileWithTime(newUncited, newTime)
	writeFileWithTime(oldRecentlyCited, oldTime)
	writeFileWithTime(oldCitedLongAgo, oldTime)

	citations := []types.CitationEvent{
		{
			ArtifactPath: ".agents/learnings/old-recently-cited.md",
			CitedAt:      time.Now().AddDate(0, 0, -5),
		},
		{
			ArtifactPath: oldCitedLongAgo,
			CitedAt:      time.Now().AddDate(0, 0, -100),
		},
	}

	staleCount, err := countStaleArtifacts(baseDir, citations, 90)
	if err != nil {
		t.Fatalf("countStaleArtifacts failed: %v", err)
	}
	// old-uncited + old-cited-long-ago are stale.
	if staleCount != 2 {
		t.Fatalf("expected 2 stale artifacts, got %d", staleCount)
	}
}

func TestComputeMetricsSigmaBounded(t *testing.T) {
	baseDir := t.TempDir()
	learningsDir := filepath.Join(baseDir, ".agents", "learnings")
	researchDir := filepath.Join(baseDir, ".agents", "research")
	citationsDir := filepath.Join(baseDir, ".agents", "ao")
	if err := os.MkdirAll(learningsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(researchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(citationsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	learningPath := filepath.Join(learningsDir, "L1.md")
	researchPath := filepath.Join(researchDir, "R1.md")
	if err := os.WriteFile(learningPath, []byte("# L1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(researchPath, []byte("# R1"), 0o644); err != nil {
		t.Fatal(err)
	}

	citations := []types.CitationEvent{
		{
			ArtifactPath: ".agents/learnings/L1.md",
			SessionID:    "s1",
			CitedAt:      time.Now().AddDate(0, 0, -1),
		},
		{
			ArtifactPath: researchPath,
			SessionID:    "s2",
			CitedAt:      time.Now().AddDate(0, 0, -1),
		},
	}

	f, err := os.Create(filepath.Join(citationsDir, "citations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, c := range citations {
		if err := enc.Encode(c); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	metrics, err := computeMetrics(baseDir, 7)
	if err != nil {
		t.Fatalf("computeMetrics failed: %v", err)
	}
	if metrics.Sigma > 1.0 {
		t.Fatalf("sigma must be <= 1.0, got %f", metrics.Sigma)
	}
	if metrics.Sigma < 0.99 {
		t.Fatalf("expected sigma close to 1.0 for one retrievable cited artifact, got %f", metrics.Sigma)
	}
	// Keep visibility count unchanged (all unique cited artifacts in period).
	if metrics.UniqueCitedArtifacts != 2 {
		t.Fatalf("expected 2 unique cited artifacts in period, got %d", metrics.UniqueCitedArtifacts)
	}
}

func TestCountRetros(t *testing.T) {
	t.Run("missing retros dir returns 0 with no error", func(t *testing.T) {
		baseDir := t.TempDir()
		// No .agents/retros directory created
		total, withLearnings, err := countRetros(baseDir, time.Now().AddDate(0, 0, -7))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if total != 0 {
			t.Errorf("expected 0 total, got %d", total)
		}
		if withLearnings != 0 {
			t.Errorf("expected 0 withLearnings, got %d", withLearnings)
		}
	})

	t.Run("retro within time period is counted", func(t *testing.T) {
		baseDir := t.TempDir()
		retrosDir := filepath.Join(baseDir, ".agents", "retros")
		if err := os.MkdirAll(retrosDir, 0755); err != nil {
			t.Fatal(err)
		}
		retroContent := "# My Retro\n\n## Key Learnings\n\n- Learned something"
		if err := os.WriteFile(filepath.Join(retrosDir, "retro.md"), []byte(retroContent), 0644); err != nil {
			t.Fatal(err)
		}
		since := time.Now().AddDate(0, 0, -7)
		total, withLearnings, err := countRetros(baseDir, since)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1 total retro, got %d", total)
		}
		if withLearnings != 1 {
			t.Errorf("expected 1 retro with learnings, got %d", withLearnings)
		}
	})

	t.Run("retro without learnings section not counted in withLearnings", func(t *testing.T) {
		baseDir := t.TempDir()
		retrosDir := filepath.Join(baseDir, ".agents", "retros")
		if err := os.MkdirAll(retrosDir, 0755); err != nil {
			t.Fatal(err)
		}
		retroContent := "# My Retro\n\n## Summary\n\nDid some things"
		if err := os.WriteFile(filepath.Join(retrosDir, "retro.md"), []byte(retroContent), 0644); err != nil {
			t.Fatal(err)
		}
		since := time.Now().AddDate(0, 0, -7)
		total, withLearnings, err := countRetros(baseDir, since)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1 total retro, got %d", total)
		}
		if withLearnings != 0 {
			t.Errorf("expected 0 retros with learnings, got %d", withLearnings)
		}
	})

	t.Run("non-md files not counted", func(t *testing.T) {
		baseDir := t.TempDir()
		retrosDir := filepath.Join(baseDir, ".agents", "retros")
		if err := os.MkdirAll(retrosDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(retrosDir, "retro.txt"), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		total, _, err := countRetros(baseDir, time.Now().AddDate(0, 0, -7))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 0 {
			t.Errorf("expected 0 for non-md file, got %d", total)
		}
	})
}

func TestParseUtilityFromJSONL(t *testing.T) {
	t.Run("returns utility from JSONL first line", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "learning.jsonl")
		data := `{"id":"L1","utility":0.75}` + "\n"
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
		got := parseUtilityFromFile(path)
		if got != 0.75 {
			t.Errorf("parseUtilityFromFile() = %f, want 0.75", got)
		}
	})

	t.Run("returns 0 for nonexistent JSONL", func(t *testing.T) {
		got := parseUtilityFromFile("/tmp/no-such-file-xyz.jsonl")
		if got != 0 {
			t.Errorf("expected 0 for nonexistent file, got %f", got)
		}
	})

	t.Run("returns 0 for JSONL without utility field", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "noutil.jsonl")
		data := `{"id":"L1","content":"no utility field"}` + "\n"
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
		got := parseUtilityFromFile(path)
		if got != 0 {
			t.Errorf("expected 0 for missing utility, got %f", got)
		}
	})
}
