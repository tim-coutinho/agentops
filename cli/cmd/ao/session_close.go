package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boshu2/agentops/cli/internal/formatter"
	"github.com/boshu2/agentops/cli/internal/parser"
	"github.com/boshu2/agentops/cli/internal/storage"
	"github.com/boshu2/agentops/cli/internal/types"
)

// sessionCmd is the parent command for session operations.
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Session lifecycle operations",
	Long: `Session lifecycle operations.

Commands:
  close    Forge transcript, extract learnings, measure flywheel delta

Examples:
  ao session close
  ao session close --session abc123
  ao session close --dry-run
  ao session close -o json`,
}

var sessionCloseSessionID string

var sessionCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Forge transcript, extract learnings, measure flywheel impact",
	Long: `Close a session by forging its transcript, extracting learnings,
measuring the flywheel delta, and reporting impact.

Pipeline:
  1. Find transcript (--session flag or most recent)
  2. Forge: extract knowledge from transcript
  3. Extract: queue and process learnings
  4. Measure: compute flywheel metrics delta
  5. Report: summarize impact

Examples:
  ao session close                        # Close most recent session
  ao session close --session abc123       # Close specific session
  ao session close --dry-run              # Preview what would happen
  ao session close -o json                # Structured output`,
	RunE: runSessionClose,
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionCloseCmd)

	sessionCloseCmd.Flags().StringVar(&sessionCloseSessionID, "session", "", "Session ID to close (default: most recent transcript)")
}

// SessionCloseResult holds the result of a session close operation.
type SessionCloseResult struct {
	SessionID     string  `json:"session_id"`
	Transcript    string  `json:"transcript"`
	Decisions     int     `json:"decisions"`
	Knowledge     int     `json:"knowledge"`
	FilesChanged  int     `json:"files_changed"`
	Issues        int     `json:"issues"`
	VelocityDelta float64 `json:"velocity_delta"`
	Status        string  `json:"status"`
	Message       string  `json:"message"`
}

func runSessionClose(cmd *cobra.Command, args []string) error {
	// Step 1: Find transcript
	transcriptPath, usedFallback, err := resolveTranscript(sessionCloseSessionID)
	if err != nil {
		return fmt.Errorf("find transcript: %w", err)
	}

	if usedFallback {
		fmt.Fprintf(os.Stderr, "No --session specified, using most recent transcript\n")
	}

	// Step 2: Dry-run check
	if GetDryRun() {
		result := SessionCloseResult{
			SessionID:  sessionCloseSessionID,
			Transcript: transcriptPath,
			Message:    "dry-run: would forge, extract, and measure flywheel delta",
		}
		return outputCloseResult(result)
	}

	// Step 3: Forge, extract, measure, build result
	return forgeExtractAndReport(transcriptPath)
}

// forgeExtractAndReport runs the forge/extract/measure pipeline and outputs
// the session close result. Extracted from runSessionClose to reduce CC.
func forgeExtractAndReport(transcriptPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	preMetrics, err := computeMetrics(cwd, 7)
	if err != nil {
		VerbosePrintf("Warning: pre-forge metrics: %v\n", err)
	}

	session, err := forgeTranscriptForClose(transcriptPath, cwd)
	if err != nil {
		return fmt.Errorf("forge transcript: %w", err)
	}

	VerbosePrintf("Forged session %s: %d decisions, %d knowledge items\n",
		session.ID, len(session.Decisions), len(session.Knowledge))

	extractCount, err := extractForClose(session, transcriptPath, cwd)
	if err != nil {
		VerbosePrintf("Warning: extraction: %v\n", err)
	} else {
		VerbosePrintf("Extraction processed %d entries\n", extractCount)
	}

	postMetrics, err := computeMetrics(cwd, 7)
	if err != nil {
		VerbosePrintf("Warning: post-forge metrics: %v\n", err)
	}

	velocityDelta := computeVelocityDelta(preMetrics, postMetrics)
	status := classifyFlywheelStatus(postMetrics)

	result := SessionCloseResult{
		SessionID:     session.ID,
		Transcript:    transcriptPath,
		Decisions:     len(session.Decisions),
		Knowledge:     len(session.Knowledge),
		FilesChanged:  len(session.FilesChanged),
		Issues:        len(session.Issues),
		VelocityDelta: velocityDelta,
		Status:        status,
		Message:       fmt.Sprintf("Session closed: %d decisions, %d learnings extracted", len(session.Decisions), len(session.Knowledge)),
	}

	return outputCloseResult(result)
}

// computeVelocityDelta returns the velocity change between pre and post metrics.
// Returns 0.0 if either measurement is nil.
func computeVelocityDelta(pre, post *types.FlywheelMetrics) float64 {
	if pre == nil || post == nil {
		return 0.0
	}
	return post.Velocity - pre.Velocity
}

// classifyFlywheelStatus returns a human-readable flywheel status label.
func classifyFlywheelStatus(post *types.FlywheelMetrics) string {
	if post == nil || post.AboveEscapeVelocity {
		return "compounding"
	}
	if post.Velocity > -0.05 {
		return "near-escape"
	}
	return "decaying"
}

// resolveTranscript finds the transcript path from a session ID or fallback.
// Returns the path, whether fallback was used, and any error.
func resolveTranscript(sessionID string) (string, bool, error) {
	if sessionID != "" {
		path, err := findTranscriptBySessionID(sessionID)
		if err != nil {
			return "", false, err
		}
		return path, false, nil
	}

	// Fallback: most recently modified transcript
	path, err := findLastSession()
	if err != nil {
		return "", false, fmt.Errorf("no transcript found: %w", err)
	}
	return path, true, nil
}

// findTranscriptBySessionID searches for a transcript file matching the session ID.
func findTranscriptBySessionID(sessionID string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	// Search in ~/.claude/projects/*/conversations/<id>.jsonl
	conversationsPattern := filepath.Join(homeDir, ".claude", "projects", "*", "conversations", sessionID+".jsonl")
	matches, err := filepath.Glob(conversationsPattern)
	if err == nil && len(matches) > 0 {
		return matches[0], nil
	}

	// Also try direct match in project dirs (older layout)
	directPattern := filepath.Join(homeDir, ".claude", "projects", "*", sessionID+".jsonl")
	matches, err = filepath.Glob(directPattern)
	if err == nil && len(matches) > 0 {
		return matches[0], nil
	}

	return "", fmt.Errorf("no transcript found for session %s", sessionID)
}

// forgeTranscriptForClose runs the forge pipeline on a transcript.
func forgeTranscriptForClose(transcriptPath, cwd string) (*storage.Session, error) {
	baseDir := filepath.Join(cwd, storage.DefaultBaseDir)
	fs := storage.NewFileStorage(
		storage.WithBaseDir(baseDir),
		storage.WithFormatters(
			formatter.NewMarkdownFormatter(),
			formatter.NewJSONLFormatter(),
		),
	)

	if err := fs.Init(); err != nil {
		return nil, fmt.Errorf("initialize storage: %w", err)
	}

	p := parser.NewParser()
	p.MaxContentLength = 0

	extractor := parser.NewExtractor()

	session, err := processTranscript(transcriptPath, p, extractor, true, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("process transcript: %w", err)
	}

	sessionPath, err := fs.WriteSession(session)
	if err != nil {
		return nil, fmt.Errorf("write session: %w", err)
	}

	// Write index
	indexEntry := &storage.IndexEntry{
		SessionID:   session.ID,
		Date:        session.Date,
		SessionPath: sessionPath,
		Summary:     session.Summary,
	}
	if err := fs.WriteIndex(indexEntry); err != nil {
		VerbosePrintf("Warning: failed to index session: %v\n", err)
	}

	// Write provenance
	provRecord := &storage.ProvenanceRecord{
		ID:           fmt.Sprintf("prov-%s", session.ID[:7]),
		ArtifactPath: sessionPath,
		ArtifactType: "session",
		SourcePath:   transcriptPath,
		SourceType:   "transcript",
		SessionID:    session.ID,
		CreatedAt:    time.Now(),
	}
	if err := fs.WriteProvenance(provRecord); err != nil {
		VerbosePrintf("Warning: failed to write provenance: %v\n", err)
	}

	return session, nil
}

// extractForClose queues the session for extraction and processes it.
func extractForClose(session *storage.Session, transcriptPath, cwd string) (int, error) {
	// Queue session for extraction
	if err := queueForExtraction(session, "", transcriptPath, cwd); err != nil {
		return 0, fmt.Errorf("queue for extraction: %w", err)
	}

	// Read and process pending extractions
	pendingPath := filepath.Join(cwd, storage.DefaultBaseDir, "pending.jsonl")
	pending, err := readPendingExtractions(pendingPath)
	if err != nil {
		return 0, fmt.Errorf("read pending: %w", err)
	}

	if len(pending) == 0 {
		return 0, nil
	}

	// Process all pending (includes the one we just queued)
	if err := runExtractAll(pendingPath, pending, cwd); err != nil {
		return 0, fmt.Errorf("extract: %w", err)
	}

	return len(pending), nil
}

// outputCloseResult formats and prints the session close result.
func outputCloseResult(result SessionCloseResult) error {
	switch GetOutput() {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)

	default:
		printCloseTable(result)
	}

	return nil
}

// printCloseTable prints a formatted table of session close results.
func printCloseTable(r SessionCloseResult) {
	fmt.Println()
	fmt.Println("  Session Close Summary")
	fmt.Println("  =====================")
	fmt.Println()

	if r.SessionID != "" {
		displayID := r.SessionID
		if len(displayID) > 12 {
			displayID = displayID[:12]
		}
		fmt.Printf("  Session:       %s\n", displayID)
	}
	fmt.Printf("  Transcript:    %s\n", shortenPath(r.Transcript))
	fmt.Printf("  Decisions:     %d\n", r.Decisions)
	fmt.Printf("  Knowledge:     %d\n", r.Knowledge)
	fmt.Printf("  Files changed: %d\n", r.FilesChanged)
	if r.Issues > 0 {
		fmt.Printf("  Issues:        %d\n", r.Issues)
	}
	fmt.Println()

	// Flywheel impact
	sign := "+"
	if r.VelocityDelta < 0 {
		sign = ""
	}
	fmt.Printf("  Flywheel:      %s (velocity %s%.3f)\n", r.Status, sign, r.VelocityDelta)
	fmt.Println()
	fmt.Printf("  %s\n", r.Message)
	fmt.Println()
}

// shortenPath returns a display-friendly version of a file path.
func shortenPath(p string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, homeDir) {
		return "~" + p[len(homeDir):]
	}
	return p
}
