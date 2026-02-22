package main

import "testing"

func TestExtractSnippet(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog and continues running"

	tests := []struct {
		name     string
		startIdx int
		maxLen   int
		wantNot  string
		check    func(string) bool
		desc     string
	}{
		{
			name:     "from start",
			startIdx: 0,
			maxLen:   15,
			check:    func(s string) bool { return len(s) > 0 && len(s) <= 20 },
			desc:     "should return short snippet with ellipsis",
		},
		{
			name:     "from middle",
			startIdx: 10,
			maxLen:   20,
			check:    func(s string) bool { return len(s) > 0 },
			desc:     "should return snippet from middle",
		},
		{
			name:     "past end",
			startIdx: len(content) + 10,
			maxLen:   20,
			check:    func(s string) bool { return s == "" },
			desc:     "should return empty for past-end index",
		},
		{
			name:     "negative start clamped",
			startIdx: -5,
			maxLen:   10,
			check:    func(s string) bool { return len(s) > 0 },
			desc:     "should clamp negative start to 0",
		},
		{
			name:     "maxLen covers entire content",
			startIdx: 0,
			maxLen:   1000,
			check:    func(s string) bool { return s == content },
			desc:     "should return full content when maxLen exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSnippet(content, tt.startIdx, tt.maxLen)
			if !tt.check(got) {
				t.Errorf("extractSnippet(%d, %d) = %q; %s", tt.startIdx, tt.maxLen, got, tt.desc)
			}
		})
	}
}

func TestLastSpaceIndex(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"with spaces", "hello world", 5},
		{"trailing space", "hello ", 5},
		{"no spaces", "helloworld", -1},
		{"empty", "", -1},
		{"single space", " ", 0},
		{"multiple spaces", "a b c", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastSpaceIndex(tt.s)
			if got != tt.want {
				t.Errorf("lastSpaceIndex(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestExtractIssueIDs(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst string
	}{
		{"single issue", "Fixed ol-0001 in commit", 1, "ol-0001"},
		{"multiple issues", "Closes ol-0001 and at-v123", 2, "ol-0001"},
		{"no issues", "This has no issue references", 0, ""},
		{"compound ID", "See gt-abc-def for details", 1, "gt-abc-def"},
		{"uppercase ignored", "Not OL-0001", 0, ""},
		{"mixed text", "Working on ag-m0r with issues ag-oke and cm-0012", 3, "ag-m0r"},
		{"empty string", "", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIssueIDs(tt.content)
			if len(got) != tt.wantCount {
				t.Errorf("extractIssueIDs(%q) returned %d IDs, want %d; got %v", tt.content, len(got), tt.wantCount, got)
			}
			if tt.wantFirst != "" && len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("first ID = %q, want %q", got[0], tt.wantFirst)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world!", 8, "hello..."},
		{"empty", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestDedup(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{"no duplicates", []string{"a", "b", "c"}, 3},
		{"all duplicates", []string{"a", "a", "a"}, 1},
		{"mixed", []string{"a", "b", "a", "c", "b"}, 3},
		{"empty", []string{}, 0},
		{"single", []string{"x"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedup(tt.input)
			if len(got) != tt.want {
				t.Errorf("dedup() returned %d items, want %d; got %v", len(got), tt.want, got)
			}
			// Verify order preserved
			if len(tt.input) > 0 && len(got) > 0 && got[0] != tt.input[0] {
				t.Errorf("first element = %q, want %q (order should be preserved)", got[0], tt.input[0])
			}
		})
	}
}
