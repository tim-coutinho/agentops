package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendRPILedgerRecord(t *testing.T) {
	root := t.TempDir()

	first, err := AppendRPILedgerRecord(root, RPILedgerAppendInput{
		RunID:   "run-append",
		Phase:   "research",
		Action:  "start",
		Details: map[string]any{"step": 1},
	})
	if err != nil {
		t.Fatalf("append first record: %v", err)
	}
	second, err := AppendRPILedgerRecord(root, RPILedgerAppendInput{
		RunID:   "run-append",
		Phase:   "plan",
		Action:  "advance",
		Details: map[string]any{"step": 2},
	})
	if err != nil {
		t.Fatalf("append second record: %v", err)
	}

	records, err := LoadRPILedgerRecords(root)
	if err != nil {
		t.Fatalf("load records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].PrevHash != "" {
		t.Fatalf("expected first prev_hash empty, got %q", records[0].PrevHash)
	}
	if records[1].PrevHash != records[0].Hash {
		t.Fatalf("expected second prev_hash to match first hash")
	}
	if first.Hash != records[0].Hash || second.Hash != records[1].Hash {
		t.Fatalf("returned records should match file contents")
	}
	if !strings.HasSuffix(records[0].TS, "Z") {
		t.Fatalf("expected UTC timestamp with Z suffix, got %q", records[0].TS)
	}
	if _, err := time.Parse(time.RFC3339Nano, records[0].TS); err != nil {
		t.Fatalf("timestamp must be RFC3339Nano: %v", err)
	}
}

func TestVerifyRPILedgerChain_Success(t *testing.T) {
	root := t.TempDir()
	for i := range 3 {
		_, err := AppendRPILedgerRecord(root, RPILedgerAppendInput{
			RunID:   "run-ok",
			Phase:   "phase",
			Action:  "action",
			Details: map[string]any{"index": i},
		})
		if err != nil {
			t.Fatalf("append record %d: %v", i, err)
		}
	}

	if err := VerifyRPILedger(root); err != nil {
		t.Fatalf("verify should pass, got: %v", err)
	}
}

func TestVerifyRPILedgerChain_TamperFailure(t *testing.T) {
	root := t.TempDir()
	for i := range 2 {
		_, err := AppendRPILedgerRecord(root, RPILedgerAppendInput{
			RunID:   "run-tamper",
			Phase:   "phase",
			Action:  "action",
			Details: map[string]any{"index": i},
		})
		if err != nil {
			t.Fatalf("append record %d: %v", i, err)
		}
	}

	ledgerPath := RPILedgerPath(root)
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 ledger lines, got %d", len(lines))
	}

	var tampered RPILedgerRecord
	if err := json.Unmarshal([]byte(lines[0]), &tampered); err != nil {
		t.Fatalf("decode first line: %v", err)
	}
	tampered.Action = "tampered-action"
	tamperedLine, err := json.Marshal(tampered)
	if err != nil {
		t.Fatalf("re-marshal tampered line: %v", err)
	}
	lines[0] = string(tamperedLine)

	if err := os.WriteFile(ledgerPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("write tampered ledger: %v", err)
	}

	err = VerifyRPILedger(root)
	if err == nil {
		t.Fatalf("expected verification failure for tampered ledger")
	}
	if !strings.Contains(err.Error(), "payload_hash mismatch") {
		t.Fatalf("expected payload_hash mismatch error, got: %v", err)
	}
}

func TestMaterializeRPIRunCache(t *testing.T) {
	root := t.TempDir()

	events := []RPILedgerAppendInput{
		{
			RunID:   "run-a",
			Phase:   "research",
			Action:  "start",
			Details: map[string]any{"order": 1},
		},
		{
			RunID:   "run-b",
			Phase:   "research",
			Action:  "start",
			Details: map[string]any{"order": 1},
		},
		{
			RunID:   "run-a",
			Phase:   "plan",
			Action:  "finish",
			Details: map[string]any{"order": 2},
		},
	}
	for i, event := range events {
		if _, err := AppendRPILedgerRecord(root, event); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}

	if err := MaterializeRPIRunCache(root, "run-a"); err != nil {
		t.Fatalf("materialize cache: %v", err)
	}

	cachePath := filepath.Join(root, ".agents/rpi/runs/run-a.json")
	cacheBytes, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}

	var cache RPIRunCache
	if err := json.Unmarshal(cacheBytes, &cache); err != nil {
		t.Fatalf("decode cache: %v", err)
	}
	if cache.RunID != "run-a" {
		t.Fatalf("expected run_id run-a, got %q", cache.RunID)
	}
	if cache.EventCount != 2 {
		t.Fatalf("expected 2 run-a events, got %d", cache.EventCount)
	}
	if cache.Latest.Action != "finish" {
		t.Fatalf("expected latest action finish, got %q", cache.Latest.Action)
	}
	if cache.Latest.Phase != "plan" {
		t.Fatalf("expected latest phase plan, got %q", cache.Latest.Phase)
	}
}
