package main

import (
	"sort"
	"testing"
)

func TestComputeSearchScore(t *testing.T) {
	tests := []struct {
		name       string
		entry      IndexEntry
		queryTerms []string
		wantMin    float64
	}{
		{
			name: "title match scores highest",
			entry: IndexEntry{
				Title:   "Mutex Pattern",
				Content: "unrelated content",
			},
			queryTerms: []string{"mutex"},
			wantMin:    3.0, // title match = 3.0
		},
		{
			name: "content match",
			entry: IndexEntry{
				Title:   "Some Title",
				Content: "Use mutex for shared state",
			},
			queryTerms: []string{"mutex"},
			wantMin:    1.0, // content match = 1.0
		},
		{
			name: "keyword match",
			entry: IndexEntry{
				Title:    "Title",
				Content:  "content",
				Keywords: []string{"mutex", "concurrency"},
			},
			queryTerms: []string{"mutex"},
			wantMin:    2.0, // keyword match = 2.0
		},
		{
			name: "all three match",
			entry: IndexEntry{
				Title:    "Mutex Pattern",
				Content:  "Use mutex for shared state",
				Keywords: []string{"mutex"},
			},
			queryTerms: []string{"mutex"},
			wantMin:    6.0, // 3 + 1 + 2 = 6.0
		},
		{
			name: "no match",
			entry: IndexEntry{
				Title:   "Database Pattern",
				Content: "pooling connections",
			},
			queryTerms: []string{"auth"},
			wantMin:    0.0,
		},
		{
			name: "utility boost",
			entry: IndexEntry{
				Title:   "Mutex Pattern",
				Content: "content",
				Utility: 0.9,
			},
			queryTerms: []string{"mutex"},
			// title match (3.0) weighted by utility: (1-0.5)*3 + 0.5*0.9*3 = 1.5 + 1.35 = 2.85
			wantMin: 2.8,
		},
		{
			name: "multiple query terms",
			entry: IndexEntry{
				Title:   "Mutex Pattern for Go",
				Content: "Use mutex and channels for concurrency",
			},
			queryTerms: []string{"mutex", "channels"},
			wantMin:    4.0, // mutex: title(3) + content(1), channels: content(1) = 5.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeSearchScore(tt.entry, tt.queryTerms)
			if got < tt.wantMin {
				t.Errorf("computeSearchScore() = %v, want >= %v", got, tt.wantMin)
			}
		})
	}

	// Test relative ordering
	t.Run("title match ranks higher than content match", func(t *testing.T) {
		titleEntry := IndexEntry{Title: "Mutex Pattern", Content: "something else"}
		contentEntry := IndexEntry{Title: "Something", Content: "mutex is useful"}

		titleScore := computeSearchScore(titleEntry, []string{"mutex"})
		contentScore := computeSearchScore(contentEntry, []string{"mutex"})

		if titleScore <= contentScore {
			t.Errorf("title match (%.2f) should rank higher than content match (%.2f)",
				titleScore, contentScore)
		}
	})
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "heading",
			content: "# My Document\n\nSome content",
			want:    "My Document",
		},
		{
			name:    "heading after front matter",
			content: "---\nid: test\n---\n# Real Title\n\nContent",
			want:    "Real Title",
		},
		{
			name:    "no heading falls back to first line",
			content: "This is the content\nSecond line",
			want:    "This is the content",
		},
		{
			name:    "empty content",
			content: "",
			want:    "Untitled",
		},
		{
			name:    "only separators",
			content: "---\n---\n",
			want:    "Untitled",
		},
		{
			name:    "long first line truncated",
			content: "This is an extremely long line that exceeds the eighty character limit and should be truncated to prevent overly long titles from appearing",
			want:    "This is an extremely long line that exceeds the eighty character limit and sh...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	t.Run("from content patterns", func(t *testing.T) {
		content := "This is a pattern: for handling errors. Also a fix: for the issue."
		got := extractKeywords(content)
		if len(got) == 0 {
			t.Error("expected keywords from content patterns")
		}
		// Should find "pattern" and "fix"
		gotMap := make(map[string]bool)
		for _, k := range got {
			gotMap[k] = true
		}
		if !gotMap["pattern"] {
			t.Error("expected 'pattern' keyword")
		}
		if !gotMap["fix"] {
			t.Error("expected 'fix' keyword")
		}
	})

	t.Run("from metadata tags", func(t *testing.T) {
		content := "# Title\n\n**Tags**: auth, database, security\n\nContent"
		got := extractKeywords(content)
		sort.Strings(got)
		found := false
		for _, k := range got {
			if k == "auth" || k == "database" || k == "security" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected tags from metadata, got %v", got)
		}
	})

	t.Run("no keywords", func(t *testing.T) {
		content := "Just some plain text without any special markers."
		got := extractKeywords(content)
		if len(got) != 0 {
			t.Errorf("expected no keywords, got %v", got)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		got := extractKeywords("")
		if len(got) != 0 {
			t.Errorf("expected no keywords for empty content, got %v", got)
		}
	})
}
