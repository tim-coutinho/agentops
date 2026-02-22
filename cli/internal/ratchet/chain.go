package ratchet

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// ChainDir is the directory for ratchet chain files.
	ChainDir = ".agents/ao"

	// ChainFile is the JSONL file for chain entries.
	ChainFile = "chain.jsonl"

	// LegacyChainDir is the old location for chain files.
	LegacyChainDir = ".agents/provenance"

	// LegacyChainFile is the old YAML chain file.
	LegacyChainFile = "chain.yaml"
)

// LoadChain loads the ratchet chain from the nearest .agents directory.
// It first tries the new JSONL format, then falls back to legacy YAML.
func LoadChain(startDir string) (*Chain, error) {
	// Find the .agents directory
	agentsDir, err := findAgentsDir(startDir)
	if err != nil {
		// No .agents found - return empty chain (valid state)
		return &Chain{
			ID:      generateChainID(),
			Started: time.Now(),
			Entries: []ChainEntry{},
		}, nil
	}

	// Try new location first
	chainPath := filepath.Join(agentsDir, "ao", ChainFile)
	if chain, err := loadJSONLChain(chainPath); err == nil {
		chain.path = chainPath
		return chain, nil
	}

	// Try legacy YAML location
	legacyPath := filepath.Join(agentsDir, "provenance", LegacyChainFile)
	if chain, err := loadLegacyYAMLChain(legacyPath); err == nil {
		chain.path = chainPath // Will write to new location
		fmt.Fprintf(os.Stderr, "Note: Migrating chain from %s to %s\n", legacyPath, chainPath)
		return chain, nil
	}

	// No existing chain - create new
	return &Chain{
		ID:      generateChainID(),
		Started: time.Now(),
		Entries: []ChainEntry{},
		path:    chainPath,
	}, nil
}

// loadJSONLChain loads a chain from JSONL format.
func loadJSONLChain(path string) (chain *Chain, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	chain = &Chain{
		path:    path,
		Entries: []ChainEntry{},
	}

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// First line is chain metadata
		if lineNum == 1 {
			if err := json.Unmarshal(line, chain); err != nil {
				return nil, fmt.Errorf("parse chain metadata: %w", err)
			}
			chain.Entries = []ChainEntry{} // Clear entries, they follow
			continue
		}

		// Subsequent lines are entries
		var entry ChainEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // Skip malformed lines
		}
		chain.Entries = append(chain.Entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read chain: %w", err)
	}

	return chain, nil
}

// legacyChain is the old YAML format structure.
type legacyChain struct {
	ID      string `yaml:"id"`
	Started string `yaml:"started"`
	EpicID  string `yaml:"epic_id,omitempty"`
	Chain   []struct {
		Step      string `yaml:"step"`
		Timestamp string `yaml:"timestamp"`
		Input     string `yaml:"input,omitempty"`
		Output    string `yaml:"output"`
		Locked    bool   `yaml:"locked"`
		Skipped   bool   `yaml:"skipped,omitempty"`
		Reason    string `yaml:"reason,omitempty"`
	} `yaml:"chain"`
}

// loadLegacyYAMLChain loads a chain from legacy YAML format.
func loadLegacyYAMLChain(path string) (*Chain, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var legacy legacyChain
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("parse legacy chain: %w", err)
	}

	chain := &Chain{
		ID:      legacy.ID,
		EpicID:  legacy.EpicID,
		Entries: make([]ChainEntry, 0, len(legacy.Chain)),
	}

	if legacy.Started != "" {
		chain.Started, _ = time.Parse(time.RFC3339, legacy.Started)
	}
	if chain.Started.IsZero() {
		chain.Started = time.Now()
	}

	for _, e := range legacy.Chain {
		entry := ChainEntry{
			Step:    ParseStep(e.Step),
			Output:  e.Output,
			Input:   e.Input,
			Locked:  e.Locked,
			Skipped: e.Skipped,
			Reason:  e.Reason,
		}
		if e.Timestamp != "" {
			entry.Timestamp, _ = time.Parse(time.RFC3339, e.Timestamp)
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}
		chain.Entries = append(chain.Entries, entry)
	}

	return chain, nil
}

// Save writes the chain to disk using JSONL format with file locking.
func (c *Chain) Save() error {
	if c.path == "" {
		return fmt.Errorf("chain has no path set")
	}

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create chain directory: %w", err)
	}

	// Open file with exclusive lock
	f, err := os.OpenFile(c.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open chain file: %w", err)
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // sync already done via lock release
	}()

	// Acquire exclusive lock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock chain file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck // unlock best-effort
	}()

	// Write chain metadata on first line
	meta := struct {
		ID      string    `json:"id"`
		Started time.Time `json:"started"`
		EpicID  string    `json:"epic_id,omitempty"`
	}{
		ID:      c.ID,
		Started: c.Started,
		EpicID:  c.EpicID,
	}
	metaLine, _ := json.Marshal(meta)
	if _, err := f.Write(append(metaLine, '\n')); err != nil {
		return fmt.Errorf("write chain metadata: %w", err)
	}

	// Write each entry
	for _, entry := range c.Entries {
		line, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("write chain entry: %w", err)
		}
	}

	return nil
}

// Append adds a new entry to the chain with file locking.
// This is atomic and safe for concurrent access.
func (c *Chain) Append(entry ChainEntry) error {
	if c.path == "" {
		return fmt.Errorf("chain has no path set")
	}

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create chain directory: %w", err)
	}

	// Open file for append with exclusive lock
	f, err := os.OpenFile(c.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open chain file: %w", err)
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // sync already done via lock release
	}()

	// Acquire exclusive lock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock chain file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck // unlock best-effort
	}()

	// Check if file is empty (needs metadata)
	stat, _ := f.Stat()
	if stat.Size() == 0 {
		meta := struct {
			ID      string    `json:"id"`
			Started time.Time `json:"started"`
			EpicID  string    `json:"epic_id,omitempty"`
		}{
			ID:      c.ID,
			Started: c.Started,
			EpicID:  c.EpicID,
		}
		metaLine, _ := json.Marshal(meta)
		if _, err := f.Write(append(metaLine, '\n')); err != nil {
			return fmt.Errorf("write chain metadata: %w", err)
		}
	}

	// Write entry
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}

	// Update in-memory state
	c.Entries = append(c.Entries, entry)
	return nil
}

// GetLatest returns the most recent entry for a given step.
func (c *Chain) GetLatest(step Step) *ChainEntry {
	for i := len(c.Entries) - 1; i >= 0; i-- {
		if c.Entries[i].Step == step {
			return &c.Entries[i]
		}
	}
	return nil
}

// IsLocked returns true if the given step has been locked.
func (c *Chain) IsLocked(step Step) bool {
	entry := c.GetLatest(step)
	return entry != nil && entry.Locked
}

// StepStatus returns the status of a step in the chain.
type StepStatus string

const (
	StatusPending    StepStatus = "pending"
	StatusInProgress StepStatus = "in_progress"
	StatusLocked     StepStatus = "locked"
	StatusSkipped    StepStatus = "skipped"
)

// GetStatus returns the current status of a step.
func (c *Chain) GetStatus(step Step) StepStatus {
	entry := c.GetLatest(step)
	if entry == nil {
		return StatusPending
	}
	if entry.Skipped {
		return StatusSkipped
	}
	if entry.Locked {
		return StatusLocked
	}
	return StatusInProgress
}

// GetAllStatus returns status for all steps.
func (c *Chain) GetAllStatus() map[Step]StepStatus {
	status := make(map[Step]StepStatus)
	for _, step := range AllSteps() {
		status[step] = c.GetStatus(step)
	}
	return status
}

// Path returns the file path where the chain is stored.
func (c *Chain) Path() string {
	return c.path
}

// SetPath sets the file path for the chain.
func (c *Chain) SetPath(path string) {
	c.path = path
}

// findAgentsDir walks up from startDir looking for a .agents directory.
func findAgentsDir(startDir string) (string, error) {
	dir := startDir
	for {
		agentsPath := filepath.Join(dir, ".agents")
		if info, err := os.Stat(agentsPath); err == nil && info.IsDir() {
			return agentsPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".agents directory not found")
		}
		dir = parent
	}
}

// generateChainID creates a unique chain identifier.
func generateChainID() string {
	return fmt.Sprintf("chain-%d", time.Now().Unix())
}

// MigrateChain migrates from legacy YAML to JSONL format.
func MigrateChain(startDir string) error {
	agentsDir, err := findAgentsDir(startDir)
	if err != nil {
		return fmt.Errorf("no .agents directory found")
	}

	legacyPath := filepath.Join(agentsDir, "provenance", LegacyChainFile)
	newPath := filepath.Join(agentsDir, "ao", ChainFile)

	// Check if legacy exists
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return fmt.Errorf("no legacy chain found at %s", legacyPath)
	}

	// Load legacy
	chain, err := loadLegacyYAMLChain(legacyPath)
	if err != nil {
		return fmt.Errorf("load legacy chain: %w", err)
	}

	// Set new path and save
	chain.path = newPath
	if err := chain.Save(); err != nil {
		return fmt.Errorf("save migrated chain: %w", err)
	}

	fmt.Printf("Migrated %d entries from %s to %s\n", len(chain.Entries), legacyPath, newPath)
	return nil
}
