package ratchet

import (
	"encoding/json"
	"testing"
	"time"
)

func TestChainEntryCycleSerialize(t *testing.T) {
	entry := ChainEntry{
		Step:       StepResearch,
		Timestamp:  time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Output:     "test-output",
		Locked:     true,
		Cycle:      2,
		ParentEpic: "ag-abc",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if v, ok := m["cycle"]; !ok {
		t.Error("expected 'cycle' field in JSON")
	} else if int(v.(float64)) != 2 {
		t.Errorf("expected cycle=2, got %v", v)
	}

	if v, ok := m["parent_epic"]; !ok {
		t.Error("expected 'parent_epic' field in JSON")
	} else if v != "ag-abc" {
		t.Errorf("expected parent_epic='ag-abc', got %v", v)
	}
}

func TestChainEntryBackwardCompatDeserialize(t *testing.T) {
	// JSON without cycle or parent_epic (legacy entry)
	legacy := `{"step":"research","timestamp":"2026-02-10T12:00:00Z","output":"test","locked":true}`

	var entry ChainEntry
	if err := json.Unmarshal([]byte(legacy), &entry); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}

	if entry.Cycle != 0 {
		t.Errorf("expected Cycle=0 for legacy entry, got %d", entry.Cycle)
	}
	if entry.ParentEpic != "" {
		t.Errorf("expected ParentEpic='' for legacy entry, got %q", entry.ParentEpic)
	}
}

func TestChainEntryDeserializeWithCycleFields(t *testing.T) {
	input := `{"step":"plan","timestamp":"2026-02-10T12:00:00Z","output":"test","locked":true,"cycle":3,"parent_epic":"ag-xyz"}`

	var entry ChainEntry
	if err := json.Unmarshal([]byte(input), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if entry.Cycle != 3 {
		t.Errorf("expected Cycle=3, got %d", entry.Cycle)
	}
	if entry.ParentEpic != "ag-xyz" {
		t.Errorf("expected ParentEpic='ag-xyz', got %q", entry.ParentEpic)
	}
}

func TestChainEntryOmitemptyZeroValues(t *testing.T) {
	// Cycle=0 and ParentEpic="" should be omitted from JSON
	entry := ChainEntry{
		Step:      StepResearch,
		Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Output:    "test",
		Locked:    true,
		Cycle:     0,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if _, ok := m["cycle"]; ok {
		t.Error("expected 'cycle' to be omitted when 0")
	}
	if _, ok := m["parent_epic"]; ok {
		t.Error("expected 'parent_epic' to be omitted when empty")
	}
}
