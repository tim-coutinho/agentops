package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendRPILedgerRecord_HappyPath(t *testing.T) {
	dir := t.TempDir()

	rec, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		RunID:  "run-001",
		Phase:  "research",
		Action: "started",
	})
	if err != nil {
		t.Fatalf("AppendRPILedgerRecord: %v", err)
	}
	if rec.RunID != "run-001" {
		t.Errorf("RunID = %q, want run-001", rec.RunID)
	}
	if rec.Phase != "research" {
		t.Errorf("Phase = %q, want research", rec.Phase)
	}
	if rec.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if rec.Hash == "" {
		t.Error("Hash should not be empty")
	}
	if rec.PayloadHash == "" {
		t.Error("PayloadHash should not be empty")
	}
	if rec.TS == "" {
		t.Error("TS should not be empty")
	}
}

func TestAppendRPILedgerRecord_MissingRunID(t *testing.T) {
	dir := t.TempDir()
	_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		Phase:  "research",
		Action: "started",
	})
	if err == nil {
		t.Fatal("expected error for missing run_id")
	}
	if !strings.Contains(err.Error(), "run_id") {
		t.Errorf("error = %v, want 'run_id'", err)
	}
}

func TestAppendRPILedgerRecord_MissingPhase(t *testing.T) {
	dir := t.TempDir()
	_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		RunID:  "run-001",
		Action: "started",
	})
	if err == nil {
		t.Fatal("expected error for missing phase")
	}
	if !strings.Contains(err.Error(), "phase") {
		t.Errorf("error = %v, want 'phase'", err)
	}
}

func TestAppendRPILedgerRecord_ChainedPrevHash(t *testing.T) {
	dir := t.TempDir()

	rec1, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		RunID: "run-001", Phase: "research", Action: "started",
	})
	if err != nil {
		t.Fatalf("first append: %v", err)
	}

	rec2, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		RunID: "run-001", Phase: "research", Action: "completed",
	})
	if err != nil {
		t.Fatalf("second append: %v", err)
	}

	// rec2.PrevHash should be rec1.Hash
	if rec2.PrevHash != rec1.Hash {
		t.Errorf("rec2.PrevHash = %q, want %q (rec1.Hash)", rec2.PrevHash, rec1.Hash)
	}
}

func TestLoadRPILedgerRecords_Empty(t *testing.T) {
	dir := t.TempDir()
	records, err := LoadRPILedgerRecords(dir)
	if err != nil {
		t.Fatalf("LoadRPILedgerRecords: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records for empty dir, got %d", len(records))
	}
}

func TestLoadRPILedgerRecords_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 3; i++ {
		_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
			RunID: "run-001", Phase: "research", Action: "event",
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	records, err := LoadRPILedgerRecords(dir)
	if err != nil {
		t.Fatalf("LoadRPILedgerRecords: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}
}

func TestVerifyRPILedgerChain_ValidChain(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 3; i++ {
		_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
			RunID: "run-001", Phase: "research", Action: "event",
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	records, err := LoadRPILedgerRecords(dir)
	if err != nil {
		t.Fatalf("LoadRPILedgerRecords: %v", err)
	}

	if err := VerifyRPILedgerChain(records); err != nil {
		t.Errorf("VerifyRPILedgerChain for valid chain: %v", err)
	}
}

func TestVerifyRPILedgerChain_TamperedRecord(t *testing.T) {
	dir := t.TempDir()

	_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		RunID: "run-001", Phase: "research", Action: "started",
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	// Tamper with the ledger file
	ledgerPath := RPILedgerPath(dir)
	data, _ := os.ReadFile(ledgerPath)
	tampered := strings.Replace(string(data), "research", "tampered", 1)
	_ = os.WriteFile(ledgerPath, []byte(tampered), 0644)

	records, err := LoadRPILedgerRecords(dir)
	if err != nil {
		t.Fatalf("LoadRPILedgerRecords: %v", err)
	}

	if err := VerifyRPILedgerChain(records); err == nil {
		t.Fatal("expected error for tampered record")
	}
}

func TestVerifyRPILedgerChain_EmptyChain(t *testing.T) {
	if err := VerifyRPILedgerChain(nil); err != nil {
		t.Errorf("expected no error for empty chain, got: %v", err)
	}
}

func TestVerifyRPILedger_HappyPath(t *testing.T) {
	dir := t.TempDir()
	_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		RunID: "run-001", Phase: "plan", Action: "completed",
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := VerifyRPILedger(dir); err != nil {
		t.Errorf("VerifyRPILedger: %v", err)
	}
}

func TestValidateLedgerRecord_MissingFields(t *testing.T) {
	cases := []struct {
		name   string
		record RPILedgerRecord
		errMsg string
	}{
		{
			name:   "wrong schema version",
			record: RPILedgerRecord{SchemaVersion: 99, EventID: "evt-1", RunID: "r", Phase: "p", Action: "a", TS: time.Now().UTC().Format(time.RFC3339Nano), PayloadHash: "x", Hash: "y"},
			errMsg: "schema_version",
		},
		{
			name:   "missing event_id",
			record: RPILedgerRecord{SchemaVersion: 1, RunID: "r", Phase: "p", Action: "a", TS: time.Now().UTC().Format(time.RFC3339Nano), PayloadHash: "x", Hash: "y"},
			errMsg: "event_id",
		},
		{
			name:   "missing run_id",
			record: RPILedgerRecord{SchemaVersion: 1, EventID: "evt-1", Phase: "p", Action: "a", TS: time.Now().UTC().Format(time.RFC3339Nano), PayloadHash: "x", Hash: "y"},
			errMsg: "run_id",
		},
		{
			name:   "missing phase",
			record: RPILedgerRecord{SchemaVersion: 1, EventID: "evt-1", RunID: "r", Action: "a", TS: time.Now().UTC().Format(time.RFC3339Nano), PayloadHash: "x", Hash: "y"},
			errMsg: "phase",
		},
		{
			name:   "missing action",
			record: RPILedgerRecord{SchemaVersion: 1, EventID: "evt-1", RunID: "r", Phase: "p", TS: time.Now().UTC().Format(time.RFC3339Nano), PayloadHash: "x", Hash: "y"},
			errMsg: "action",
		},
		{
			name:   "missing ts",
			record: RPILedgerRecord{SchemaVersion: 1, EventID: "evt-1", RunID: "r", Phase: "p", Action: "a", PayloadHash: "x", Hash: "y"},
			errMsg: "ts",
		},
		{
			name:   "missing payload_hash",
			record: RPILedgerRecord{SchemaVersion: 1, EventID: "evt-1", RunID: "r", Phase: "p", Action: "a", TS: time.Now().UTC().Format(time.RFC3339Nano), Hash: "y"},
			errMsg: "payload_hash",
		},
		{
			name:   "missing hash",
			record: RPILedgerRecord{SchemaVersion: 1, EventID: "evt-1", RunID: "r", Phase: "p", Action: "a", TS: time.Now().UTC().Format(time.RFC3339Nano), PayloadHash: "x"},
			errMsg: "hash",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateLedgerRecord(tc.record)
			if err == nil {
				t.Fatalf("expected error for %q", tc.name)
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tc.errMsg)
			}
		})
	}
}

func TestValidateLedgerRecord_InvalidTS(t *testing.T) {
	record := RPILedgerRecord{
		SchemaVersion: 1, EventID: "evt-1", RunID: "r", Phase: "p", Action: "a",
		TS: "not-a-timestamp", PayloadHash: "x", Hash: "y",
	}
	if err := validateLedgerRecord(record); err == nil {
		t.Fatal("expected error for invalid ts")
	}
}

func TestValidateLedgerRecord_NonUTCTS(t *testing.T) {
	// Create a non-UTC formatted timestamp
	loc, _ := time.LoadLocation("America/New_York")
	nonUTC := time.Now().In(loc).Format(time.RFC3339Nano)
	record := RPILedgerRecord{
		SchemaVersion: 1, EventID: "evt-1", RunID: "r", Phase: "p", Action: "a",
		TS: nonUTC, PayloadHash: "x", Hash: "y",
	}
	if err := validateLedgerRecord(record); err == nil {
		t.Fatal("expected error for non-UTC ts")
	}
}

func TestNormalizeDetails_Nil(t *testing.T) {
	result, err := normalizeDetails(nil)
	if err != nil {
		t.Fatalf("normalizeDetails(nil): %v", err)
	}
	if string(result) != "{}" {
		t.Errorf("normalizeDetails(nil) = %q, want {}", string(result))
	}
}

func TestNormalizeDetails_RawMessage(t *testing.T) {
	raw := json.RawMessage(`{"key":"value"}`)
	result, err := normalizeDetails(raw)
	if err != nil {
		t.Fatalf("normalizeDetails(RawMessage): %v", err)
	}
	if !strings.Contains(string(result), "key") {
		t.Errorf("normalizeDetails(RawMessage) = %q, want to contain 'key'", string(result))
	}
}

func TestNormalizeDetails_Struct(t *testing.T) {
	type myDetails struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}
	details := myDetails{Field1: "hello", Field2: 42}
	result, err := normalizeDetails(details)
	if err != nil {
		t.Fatalf("normalizeDetails(struct): %v", err)
	}
	if !strings.Contains(string(result), "hello") {
		t.Errorf("normalizeDetails(struct) = %q, want to contain 'hello'", string(result))
	}
}

func TestNormalizeDetails_InvalidJSON(t *testing.T) {
	_, err := normalizeDetails([]byte("{invalid json}"))
	if err == nil {
		t.Fatal("expected error for invalid JSON bytes")
	}
}

func TestNormalizeDetails_EmptyBytes(t *testing.T) {
	result, err := normalizeDetails([]byte("   "))
	if err != nil {
		t.Fatalf("normalizeDetails(empty bytes): %v", err)
	}
	if string(result) != "{}" {
		t.Errorf("normalizeDetails(empty) = %q, want {}", string(result))
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []byte(`{"key":"value"}`)

	if err := writeFileAtomic(path, data, 0644); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after atomic write: %v", err)
	}
	if string(read) != string(data) {
		t.Errorf("content = %q, want %q", string(read), string(data))
	}
}

func TestSyncDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := syncDirectory(dir); err != nil {
		t.Errorf("syncDirectory: %v", err)
	}
}

func TestNewRPILedgerEventID(t *testing.T) {
	id := newRPILedgerEventID()
	if !strings.HasPrefix(id, "evt-") {
		t.Errorf("event ID should start with 'evt-', got %q", id)
	}
	if len(id) < 10 {
		t.Errorf("event ID seems too short: %q", id)
	}
	// IDs should be unique
	id2 := newRPILedgerEventID()
	if id == id2 {
		t.Logf("Warning: two consecutive event IDs match (unlikely): %q", id)
	}
}

func TestHashHex(t *testing.T) {
	hash := hashHex([]byte("hello world"))
	if len(hash) != 64 {
		t.Errorf("hashHex length = %d, want 64 (SHA256 hex)", len(hash))
	}
	// Should be deterministic
	hash2 := hashHex([]byte("hello world"))
	if hash != hash2 {
		t.Error("hashHex is not deterministic")
	}
	// Different input → different hash
	hash3 := hashHex([]byte("different"))
	if hash == hash3 {
		t.Error("hashHex should produce different hashes for different inputs")
	}
}

func TestVerifyRPILedger_Internal(t *testing.T) {
	dir := t.TempDir()

	// Empty ledger should pass
	result, err := verifyRPILedger(dir)
	if err != nil {
		t.Fatalf("verifyRPILedger on empty: %v", err)
	}
	if !result.Pass {
		t.Error("expected pass for empty ledger")
	}
	if result.RecordCount != 0 {
		t.Errorf("RecordCount = %d, want 0", result.RecordCount)
	}

	// Add valid records
	for i := 0; i < 2; i++ {
		_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
			RunID: "run-001", Phase: "plan", Action: "event",
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	result, err = verifyRPILedger(dir)
	if err != nil {
		t.Fatalf("verifyRPILedger: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected pass for valid chain, got message: %s", result.Message)
	}
	if result.RecordCount != 2 {
		t.Errorf("RecordCount = %d, want 2", result.RecordCount)
	}
}

func TestMaterializeRPIRunCache_EmptyRunID(t *testing.T) {
	err := MaterializeRPIRunCache(t.TempDir(), "")
	if err == nil {
		t.Fatal("expected error for empty run_id")
	}
	if !strings.Contains(err.Error(), "run_id") {
		t.Errorf("error = %v, want 'run_id'", err)
	}
}

func TestMaterializeRPIRunCache_InvalidRunID(t *testing.T) {
	err := MaterializeRPIRunCache(t.TempDir(), "../escape")
	if err == nil {
		t.Fatal("expected error for path-traversal run_id")
	}
}

func TestMaterializeRPIRunCache_HappyPath(t *testing.T) {
	dir := t.TempDir()

	// Append a record
	_, err := AppendRPILedgerRecord(dir, RPILedgerAppendInput{
		RunID: "run-001", Phase: "plan", Action: "started",
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := MaterializeRPIRunCache(dir, "run-001"); err != nil {
		t.Fatalf("MaterializeRPIRunCache: %v", err)
	}

	// Cache file should exist
	cachePath := filepath.Join(dir, rpiRunCacheRelativeDir, "run-001.json")
	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("cache file should exist at %s: %v", cachePath, err)
	}
}
