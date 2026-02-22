package main

import (
	"testing"
	"time"
)

func TestParseBeadIDs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantFirst string
		wantErr   bool
	}{
		{
			name:      "array of beads",
			input:     `[{"id":"ol-001"},{"id":"ol-002"},{"id":"ol-003"}]`,
			wantCount: 3,
			wantFirst: "ol-001",
		},
		{
			name:      "single bead object",
			input:     `{"id":"ol-001"}`,
			wantCount: 1,
			wantFirst: "ol-001",
		},
		{
			name:      "empty array",
			input:     `[]`,
			wantCount: 0,
		},
		{
			name:      "empty input",
			input:     "",
			wantCount: 0,
		},
		{
			name:    "invalid JSON",
			input:   "not json",
			wantErr: true,
		},
		{
			name:      "single object with empty ID",
			input:     `{"id":""}`,
			wantCount: 0,
		},
		{
			name:      "array with extra fields",
			input:     `[{"id":"ag-m0r","title":"test","status":"open"}]`,
			wantCount: 1,
			wantFirst: "ag-m0r",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBeadIDs([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("got %d IDs, want %d; got %v", len(got), tt.wantCount, got)
			}
			if tt.wantFirst != "" && len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("first ID = %q, want %q", got[0], tt.wantFirst)
			}
		})
	}
}

func TestDefaultFireConfig(t *testing.T) {
	cfg := DefaultFireConfig()
	if cfg.MaxPolecats != 4 {
		t.Errorf("MaxPolecats = %d, want 4", cfg.MaxPolecats)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want 30s", cfg.PollInterval)
	}
	if cfg.BackoffBase != 30*time.Second {
		t.Errorf("BackoffBase = %v, want 30s", cfg.BackoffBase)
	}
}

func TestIsComplete(t *testing.T) {
	t.Run("empty ready and burning is complete", func(t *testing.T) {
		state := &FireState{
			Ready:   []string{},
			Burning: []string{},
			Reaped:  []string{"ol-001"},
		}
		if !isComplete(state) {
			t.Error("expected complete when ready and burning are empty")
		}
	})

	t.Run("has ready issues means not complete", func(t *testing.T) {
		state := &FireState{
			Ready:   []string{"ol-001"},
			Burning: []string{},
		}
		if isComplete(state) {
			t.Error("expected not complete when ready has issues")
		}
	})

	t.Run("has burning issues means not complete", func(t *testing.T) {
		state := &FireState{
			Ready:   []string{},
			Burning: []string{"ol-002"},
		}
		if isComplete(state) {
			t.Error("expected not complete when burning has issues")
		}
	})

	t.Run("nil slices treated as empty (complete)", func(t *testing.T) {
		state := &FireState{}
		if !isComplete(state) {
			t.Error("expected complete for empty state")
		}
	})
}
