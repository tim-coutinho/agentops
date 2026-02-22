package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	rpiLedgerSchemaVersion = 1
	rpiLedgerRelativePath  = ".agents/ledger/rpi-events.jsonl"
	rpiRunCacheRelativeDir = ".agents/rpi/runs"
)

// RPILedgerRecord is a single append-only event in the RPI ledger.
type RPILedgerRecord struct {
	SchemaVersion int             `json:"schema_version"`
	EventID       string          `json:"event_id"`
	RunID         string          `json:"run_id"`
	TS            string          `json:"ts"`
	Phase         string          `json:"phase"`
	Action        string          `json:"action"`
	Details       json.RawMessage `json:"details"`
	PrevHash      string          `json:"prev_hash"`
	PayloadHash   string          `json:"payload_hash"`
	Hash          string          `json:"hash"`
}

// RPILedgerAppendInput contains fields needed for appending an event.
type RPILedgerAppendInput struct {
	RunID   string
	Phase   string
	Action  string
	Details any
}

// RPIRunCache is a materialized cache of the latest state for one run.
type RPIRunCache struct {
	RunID      string          `json:"run_id"`
	EventCount int             `json:"event_count"`
	Latest     RPILedgerRecord `json:"latest"`
	UpdatedAt  string          `json:"updated_at"`
}

// rpiLedgerEvent is the internal event shape used by rpi orchestration code.
// It intentionally mirrors RPILedgerAppendInput while keeping a stable
// package-local API for callers in this package.
type rpiLedgerEvent struct {
	RunID   string
	Phase   string
	Action  string
	Details any
}

// rpiLedgerRecord is the internal alias used by rpi orchestration code.
type rpiLedgerRecord = RPILedgerRecord

// rpiLedgerVerifyResult is the machine-readable verify output contract.
type rpiLedgerVerifyResult struct {
	Pass             bool   `json:"pass"`
	RecordCount      int    `json:"record_count"`
	FirstBrokenIndex int    `json:"first_broken_index"`
	Message          string `json:"message,omitempty"`
}

type rpiLedgerPayload struct {
	SchemaVersion int             `json:"schema_version"`
	EventID       string          `json:"event_id"`
	RunID         string          `json:"run_id"`
	TS            string          `json:"ts"`
	Phase         string          `json:"phase"`
	Action        string          `json:"action"`
	Details       json.RawMessage `json:"details"`
	PrevHash      string          `json:"prev_hash"`
}

// RPILedgerPath returns the absolute ledger file path for a repo root.
func RPILedgerPath(rootDir string) string {
	return filepath.Join(rootDir, rpiLedgerRelativePath)
}

// AppendRPILedgerRecord appends one event with lock + fsync durability.
func AppendRPILedgerRecord(rootDir string, input RPILedgerAppendInput) (RPILedgerRecord, error) {
	if strings.TrimSpace(input.RunID) == "" {
		return RPILedgerRecord{}, fmt.Errorf("run_id is required")
	}
	if strings.TrimSpace(input.Phase) == "" {
		return RPILedgerRecord{}, fmt.Errorf("phase is required")
	}
	if strings.TrimSpace(input.Action) == "" {
		return RPILedgerRecord{}, fmt.Errorf("action is required")
	}

	ledgerPath := RPILedgerPath(rootDir)
	ledgerDir := filepath.Dir(ledgerPath)
	if err := os.MkdirAll(ledgerDir, 0755); err != nil {
		return RPILedgerRecord{}, fmt.Errorf("create ledger dir: %w", err)
	}

	lockFile, err := os.OpenFile(ledgerPath+".lock", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return RPILedgerRecord{}, fmt.Errorf("open ledger lock: %w", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return RPILedgerRecord{}, fmt.Errorf("lock ledger: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	}()

	ledgerFile, err := os.OpenFile(ledgerPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return RPILedgerRecord{}, fmt.Errorf("open ledger: %w", err)
	}
	defer ledgerFile.Close()

	prevHash, err := readLastLedgerHash(ledgerFile)
	if err != nil {
		return RPILedgerRecord{}, err
	}

	details, err := normalizeDetails(input.Details)
	if err != nil {
		return RPILedgerRecord{}, err
	}

	record := RPILedgerRecord{
		SchemaVersion: rpiLedgerSchemaVersion,
		EventID:       newRPILedgerEventID(),
		RunID:         input.RunID,
		TS:            time.Now().UTC().Format(time.RFC3339Nano),
		Phase:         input.Phase,
		Action:        input.Action,
		Details:       details,
		PrevHash:      prevHash,
	}

	payloadHash, hashValue, err := computeLedgerHashes(record)
	if err != nil {
		return RPILedgerRecord{}, err
	}
	record.PayloadHash = payloadHash
	record.Hash = hashValue

	line, err := json.Marshal(record)
	if err != nil {
		return RPILedgerRecord{}, fmt.Errorf("marshal ledger record: %w", err)
	}

	if _, err := ledgerFile.Seek(0, io.SeekEnd); err != nil {
		return RPILedgerRecord{}, fmt.Errorf("seek ledger end: %w", err)
	}
	if _, err := ledgerFile.Write(append(line, '\n')); err != nil {
		return RPILedgerRecord{}, fmt.Errorf("append ledger record: %w", err)
	}
	if err := ledgerFile.Sync(); err != nil {
		return RPILedgerRecord{}, fmt.Errorf("fsync ledger: %w", err)
	}
	if err := syncDirectory(ledgerDir); err != nil {
		return RPILedgerRecord{}, err
	}

	return record, nil
}

// LoadRPILedgerRecords loads all ledger events in append order.
func LoadRPILedgerRecords(rootDir string) ([]RPILedgerRecord, error) {
	return loadRPILedgerRecordsFromPath(RPILedgerPath(rootDir))
}

// VerifyRPILedger verifies the on-disk ledger chain end-to-end.
func VerifyRPILedger(rootDir string) error {
	records, err := LoadRPILedgerRecords(rootDir)
	if err != nil {
		return err
	}
	return VerifyRPILedgerChain(records)
}

// appendRPILedgerEvent appends a single run event to the on-disk ledger.
func appendRPILedgerEvent(rootDir string, event rpiLedgerEvent) (rpiLedgerRecord, error) {
	return AppendRPILedgerRecord(rootDir, RPILedgerAppendInput(event))
}

// verifyRPILedger verifies on-disk ledger integrity and reports the first
// broken index without failing the call for chain mismatches.
func verifyRPILedger(rootDir string) (rpiLedgerVerifyResult, error) {
	records, err := LoadRPILedgerRecords(rootDir)
	if err != nil {
		return rpiLedgerVerifyResult{}, err
	}

	result := rpiLedgerVerifyResult{
		Pass:             true,
		RecordCount:      len(records),
		FirstBrokenIndex: -1,
	}

	prevHash := ""
	for i, record := range records {
		if err := validateLedgerRecord(record); err != nil {
			result.Pass = false
			result.FirstBrokenIndex = i + 1
			result.Message = err.Error()
			return result, nil
		}
		if record.PrevHash != prevHash {
			result.Pass = false
			result.FirstBrokenIndex = i + 1
			result.Message = fmt.Sprintf("prev_hash mismatch: got %q want %q", record.PrevHash, prevHash)
			return result, nil
		}

		payloadHash, hashValue, err := computeLedgerHashes(record)
		if err != nil {
			result.Pass = false
			result.FirstBrokenIndex = i + 1
			result.Message = err.Error()
			return result, nil
		}
		if record.PayloadHash != payloadHash {
			result.Pass = false
			result.FirstBrokenIndex = i + 1
			result.Message = "payload_hash mismatch"
			return result, nil
		}
		if record.Hash != hashValue {
			result.Pass = false
			result.FirstBrokenIndex = i + 1
			result.Message = "hash mismatch"
			return result, nil
		}
		prevHash = record.Hash
	}

	return result, nil
}

// materializeRPIRunCache refreshes the mutable run cache for one run.
func materializeRPIRunCache(rootDir, runID string) error {
	return MaterializeRPIRunCache(rootDir, runID)
}

// VerifyRPILedgerChain verifies hashes and prev-hash links for all records.
func VerifyRPILedgerChain(records []RPILedgerRecord) error {
	prevHash := ""
	for i, record := range records {
		if err := validateLedgerRecord(record); err != nil {
			return fmt.Errorf("record %d: %w", i+1, err)
		}
		if record.PrevHash != prevHash {
			return fmt.Errorf("record %d: prev_hash mismatch: got %q want %q", i+1, record.PrevHash, prevHash)
		}

		payloadHash, hashValue, err := computeLedgerHashes(record)
		if err != nil {
			return fmt.Errorf("record %d: %w", i+1, err)
		}
		if record.PayloadHash != payloadHash {
			return fmt.Errorf("record %d: payload_hash mismatch", i+1)
		}
		if record.Hash != hashValue {
			return fmt.Errorf("record %d: hash mismatch", i+1)
		}
		prevHash = record.Hash
	}
	return nil
}

// MaterializeRPIRunCache writes .agents/rpi/runs/<run_id>.json for one run.
func MaterializeRPIRunCache(rootDir, runID string) error {
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run_id is required")
	}
	if strings.Contains(runID, string(os.PathSeparator)) || strings.Contains(runID, "..") {
		return fmt.Errorf("run_id contains invalid path elements")
	}

	records, err := LoadRPILedgerRecords(rootDir)
	if err != nil {
		return err
	}
	if err := VerifyRPILedgerChain(records); err != nil {
		return err
	}

	var latest RPILedgerRecord
	count := 0
	for _, record := range records {
		if record.RunID != runID {
			continue
		}
		latest = record
		count++
	}
	if count == 0 {
		return os.ErrNotExist
	}

	cache := RPIRunCache{
		RunID:      runID,
		EventCount: count,
		Latest:     latest,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	cacheBytes, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run cache: %w", err)
	}
	cacheBytes = append(cacheBytes, '\n')

	cachePath := filepath.Join(rootDir, rpiRunCacheRelativeDir, runID+".json")
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("create run cache dir: %w", err)
	}
	return writeFileAtomic(cachePath, cacheBytes, 0644)
}

func readLastLedgerHash(file *os.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek ledger start: %w", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	lastHash := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record RPILedgerRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return "", fmt.Errorf("decode existing ledger record: %w", err)
		}
		lastHash = record.Hash
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan ledger: %w", err)
	}
	return lastHash, nil
}

func loadRPILedgerRecordsFromPath(path string) ([]RPILedgerRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open ledger: %w", err)
	}
	defer file.Close()

	var records []RPILedgerRecord
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record RPILedgerRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("decode ledger line %d: %w", lineNum, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan ledger: %w", err)
	}
	return records, nil
}

func validateLedgerRecord(record RPILedgerRecord) error {
	if record.SchemaVersion != rpiLedgerSchemaVersion {
		return fmt.Errorf("schema_version mismatch: got %d want %d", record.SchemaVersion, rpiLedgerSchemaVersion)
	}
	if strings.TrimSpace(record.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(record.RunID) == "" {
		return fmt.Errorf("run_id is required")
	}
	if strings.TrimSpace(record.Phase) == "" {
		return fmt.Errorf("phase is required")
	}
	if strings.TrimSpace(record.Action) == "" {
		return fmt.Errorf("action is required")
	}
	if strings.TrimSpace(record.TS) == "" {
		return fmt.Errorf("ts is required")
	}
	t, err := time.Parse(time.RFC3339Nano, record.TS)
	if err != nil {
		return fmt.Errorf("invalid ts: %w", err)
	}
	if t.UTC().Format(time.RFC3339Nano) != record.TS {
		return fmt.Errorf("ts must be UTC RFC3339Nano")
	}
	if strings.TrimSpace(record.PayloadHash) == "" {
		return fmt.Errorf("payload_hash is required")
	}
	if strings.TrimSpace(record.Hash) == "" {
		return fmt.Errorf("hash is required")
	}
	if _, err := normalizeDetails(record.Details); err != nil {
		return err
	}
	return nil
}

func computeLedgerHashes(record RPILedgerRecord) (payloadHash string, hashValue string, err error) {
	details, err := normalizeDetails(record.Details)
	if err != nil {
		return "", "", err
	}
	payload := rpiLedgerPayload{
		SchemaVersion: record.SchemaVersion,
		EventID:       record.EventID,
		RunID:         record.RunID,
		TS:            record.TS,
		Phase:         record.Phase,
		Action:        record.Action,
		Details:       details,
		PrevHash:      record.PrevHash,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal payload: %w", err)
	}
	payloadHash = hashHex(payloadBytes)
	hashValue = hashHex([]byte(payloadHash + "\n" + record.PrevHash))
	return payloadHash, hashValue, nil
}

func normalizeDetails(details any) (json.RawMessage, error) {
	if details == nil {
		return json.RawMessage([]byte("{}")), nil
	}

	if raw, ok := details.(json.RawMessage); ok {
		details = []byte(raw)
	}

	switch v := details.(type) {
	case []byte:
		if len(bytes.TrimSpace(v)) == 0 {
			return json.RawMessage([]byte("{}")), nil
		}
		var parsed any
		if err := json.Unmarshal(v, &parsed); err != nil {
			return nil, fmt.Errorf("details must be valid JSON: %w", err)
		}
		normalized, err := json.Marshal(parsed)
		if err != nil {
			return nil, fmt.Errorf("marshal details: %w", err)
		}
		return json.RawMessage(normalized), nil
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal details: %w", err)
		}
		var parsed any
		if err := json.Unmarshal(encoded, &parsed); err != nil {
			return nil, fmt.Errorf("details must be valid JSON: %w", err)
		}
		normalized, err := json.Marshal(parsed)
		if err != nil {
			return nil, fmt.Errorf("marshal details: %w", err)
		}
		return json.RawMessage(normalized), nil
	}
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		_ = tempFile.Close()
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tempFile.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("fsync temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return err
	}

	cleanup = false
	return nil
}

func syncDirectory(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open directory for fsync: %w", err)
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			return nil
		}
		return fmt.Errorf("fsync directory: %w", err)
	}
	return nil
}

func newRPILedgerEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	return "evt-" + hex.EncodeToString(b[:])
}

func hashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
