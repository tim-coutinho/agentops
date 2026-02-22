package main

import (
	"testing"
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
