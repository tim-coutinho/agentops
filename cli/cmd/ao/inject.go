package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boshu2/agentops/cli/internal/ratchet"
	"github.com/boshu2/agentops/cli/internal/types"
)

const (
	// DefaultInjectMaxTokens is the default token budget for injection (~1500 tokens ≈ 6KB)
	DefaultInjectMaxTokens = 1500

	// InjectCharsPerToken is the approximate characters per token (conservative estimate)
	InjectCharsPerToken = 4

	// MaxLearningsToInject is the maximum number of learnings to include
	MaxLearningsToInject = 10

	// MaxPatternsToInject is the maximum number of patterns to include
	MaxPatternsToInject = 5

	// MaxSessionsToInject is the maximum number of recent sessions to summarize
	MaxSessionsToInject = 5

	// quarantineRelPath is the path to OL quarantine constraints relative to the .ol/ directory.
	quarantineRelPath = "constraints/quarantine.json"
)

var (
	injectMaxTokens  int
	injectContext    string
	injectFormat     string
	injectSessionID  string
	injectNoCite     bool
	injectApplyDecay bool
)

type olConstraint struct {
	Pattern    string  `json:"pattern"`
	Detection  string  `json:"detection"`
	Source     string  `json:"source,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Status     string  `json:"status,omitempty"`
}

type injectedKnowledge struct {
	Learnings     []learning     `json:"learnings,omitempty"`
	Patterns      []pattern      `json:"patterns,omitempty"`
	Sessions      []session      `json:"sessions,omitempty"`
	OLConstraints []olConstraint `json:"ol_constraints,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
	Query         string         `json:"query,omitempty"`
}

type learning struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Summary        string  `json:"summary"`
	Source         string  `json:"source,omitempty"`
	FreshnessScore float64 `json:"freshness_score,omitempty"`
	AgeWeeks       float64 `json:"age_weeks,omitempty"`
	Utility        float64 `json:"utility,omitempty"`         // MemRL utility value
	CompositeScore float64 `json:"composite_score,omitempty"` // Two-Phase ranking score
	Superseded     bool    `json:"-"`                         // Internal flag - not serialized
}

type pattern struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	FilePath       string  `json:"file_path,omitempty"`
	FreshnessScore float64 `json:"freshness_score,omitempty"`
	AgeWeeks       float64 `json:"age_weeks,omitempty"`
	Utility        float64 `json:"utility,omitempty"`
	CompositeScore float64 `json:"composite_score,omitempty"`
}

type session struct {
	Date    string `json:"date"`
	Summary string `json:"summary"`
}

var injectCmd = &cobra.Command{
	Use:   "inject [context]",
	Short: "Output relevant knowledge for session injection",
	Long: `Inject searches and outputs relevant knowledge for session startup.

This command is designed to be called from a SessionStart hook to
inject prior learnings, patterns, and context into new sessions.

Searches:
  1. Recent learnings (.agents/learnings/*.md)
  2. Active patterns (.agents/patterns/*.md)
  3. Recent session summaries (.agents/ao/sessions/)

Uses file-based search with Two-Phase retrieval (freshness + utility scoring).
CASS integration adds maturity weighting and confidence decay.

Examples:
  ao inject                     # Inject general knowledge
  ao inject "authentication"    # Inject knowledge about auth
  ao inject --max-tokens 2000   # Larger budget
  ao inject --format json       # JSON output
  ao inject --no-cite           # Skip citation recording
  ao inject --apply-decay       # Apply confidence decay before ranking`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInject,
}

func init() {
	injectCmd.GroupID = "knowledge"
	rootCmd.AddCommand(injectCmd)
	injectCmd.Flags().IntVar(&injectMaxTokens, "max-tokens", DefaultInjectMaxTokens, "Maximum tokens to output")
	injectCmd.Flags().StringVar(&injectContext, "context", "", "Context query for filtering (alternative to positional arg)")
	injectCmd.Flags().StringVar(&injectFormat, "format", "markdown", "Output format: markdown, json")
	injectCmd.Flags().StringVar(&injectSessionID, "session", "", "Session ID for citation tracking (auto-generated if empty)")
	injectCmd.Flags().BoolVar(&injectNoCite, "no-cite", false, "Disable citation recording")
	injectCmd.Flags().BoolVar(&injectApplyDecay, "apply-decay", false, "Apply confidence decay before ranking")
}

func runInject(cmd *cobra.Command, args []string) error {
	query := resolveInjectQuery(args)

	if GetDryRun() {
		printInjectDryRun(query)
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	sessionID := canonicalSessionID(injectSessionID)
	knowledge := gatherKnowledge(cwd, query, sessionID)

	output, err := renderKnowledge(knowledge, injectFormat)
	if err != nil {
		return err
	}

	charBudget := injectMaxTokens * InjectCharsPerToken
	if len(output) > charBudget {
		if injectFormat == "json" {
			output = trimJSONToCharBudget(knowledge, charBudget)
		} else {
			output = trimToCharBudget(output, charBudget)
		}
	}

	fmt.Println(output)
	return nil
}

// resolveInjectQuery returns the query from positional args or the --context flag.
func resolveInjectQuery(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return injectContext
}

// printInjectDryRun prints the dry-run message for inject.
func printInjectDryRun(query string) {
	fmt.Printf("[dry-run] Would inject knowledge")
	if query != "" {
		fmt.Printf(" filtered by: %s", query)
	}
	fmt.Printf(" (max %d tokens)\n", injectMaxTokens)
}

// gatherKnowledge collects all knowledge sources and records citations.
func gatherKnowledge(cwd, query, sessionID string) *injectedKnowledge {
	knowledge := &injectedKnowledge{
		Timestamp: time.Now(),
		Query:     query,
	}

	knowledge.Learnings = gatherLearnings(cwd, query, sessionID)
	knowledge.Patterns = gatherPatterns(cwd, query, sessionID)
	knowledge.Sessions = gatherSessions(cwd, query)
	knowledge.OLConstraints = gatherOLConstraints(cwd, query)

	return knowledge
}

// gatherLearnings collects learnings and records their citations.
func gatherLearnings(cwd, query, sessionID string) []learning {
	learnings, err := collectLearnings(cwd, query, MaxLearningsToInject)
	if err != nil {
		VerbosePrintf("Warning: failed to collect learnings: %v\n", err)
	}

	// Record citations for retrieved learnings (Phase 0: Critical for MemRL feedback loop)
	if !injectNoCite && len(learnings) > 0 {
		if err := recordCitations(cwd, learnings, sessionID, query); err != nil {
			VerbosePrintf("Warning: failed to record citations: %v\n", err)
		} else {
			VerbosePrintf("Recorded %d citations for session %s\n", len(learnings), sessionID)
		}
	}

	return learnings
}

// gatherPatterns collects patterns and records their citations.
func gatherPatterns(cwd, query, sessionID string) []pattern {
	patterns, err := collectPatterns(cwd, query, MaxPatternsToInject)
	if err != nil {
		VerbosePrintf("Warning: failed to collect patterns: %v\n", err)
	}

	// Record citations for retrieved patterns (closes σ gap: patterns were retrieved but never cited)
	if !injectNoCite && len(patterns) > 0 {
		if err := recordPatternCitations(cwd, patterns, sessionID, query); err != nil {
			VerbosePrintf("Warning: failed to record pattern citations: %v\n", err)
		} else {
			VerbosePrintf("Recorded %d pattern citations for session %s\n", len(patterns), sessionID)
		}
	}

	return patterns
}

// gatherSessions collects recent session summaries.
func gatherSessions(cwd, query string) []session {
	sessions, err := collectRecentSessions(cwd, query, MaxSessionsToInject)
	if err != nil {
		VerbosePrintf("Warning: failed to collect sessions: %v\n", err)
	}
	return sessions
}

// gatherOLConstraints collects Olympus constraints (no-op if .ol/ doesn't exist).
func gatherOLConstraints(cwd, query string) []olConstraint {
	olConstraints, err := collectOLConstraints(cwd, query)
	if err != nil {
		VerbosePrintf("Warning: failed to collect OL constraints: %v\n", err)
	}
	return olConstraints
}

// renderKnowledge formats the knowledge struct into the requested output format.
func renderKnowledge(knowledge *injectedKnowledge, format string) (string, error) {
	if format == "json" {
		data, err := json.MarshalIndent(knowledge, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal json: %w", err)
		}
		return string(data), nil
	}
	return formatKnowledgeMarkdown(knowledge), nil
}

// findAgentsSubdir looks for .agents/{subdir}/ walking up to rig root
func findAgentsSubdir(startDir, subdir string) string {
	dir := startDir
	for {
		candidate := filepath.Join(dir, ".agents", subdir)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		// Check if we're at rig root (has .beads, crew, or polecats)
		markers := []string{".beads", "crew", "polecats"}
		atRigRoot := false
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				atRigRoot = true
				break
			}
		}
		if atRigRoot {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func writeLearningsSection(sb *strings.Builder, learnings []learning) {
	if len(learnings) == 0 {
		return
	}
	sb.WriteString("### Recent Learnings\n")
	for _, l := range learnings {
		text := l.Title
		if l.Summary != "" {
			text = l.Summary
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", l.ID, text))
	}
	sb.WriteString("\n")
}

func writePatternsSection(sb *strings.Builder, patterns []pattern) {
	if len(patterns) == 0 {
		return
	}
	sb.WriteString("### Active Patterns\n")
	for _, p := range patterns {
		if p.Description != "" {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.Name, p.Description))
		} else {
			sb.WriteString(fmt.Sprintf("- **%s**\n", p.Name))
		}
	}
	sb.WriteString("\n")
}

func writeSessionsSection(sb *strings.Builder, sessions []session) {
	if len(sessions) == 0 {
		return
	}
	sb.WriteString("### Recent Sessions\n")
	for _, s := range sessions {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", s.Date, s.Summary))
	}
	sb.WriteString("\n")
}

func writeConstraintsSection(sb *strings.Builder, constraints []olConstraint) {
	if len(constraints) == 0 {
		return
	}
	sb.WriteString("### Olympus Constraints\n")
	for _, c := range constraints {
		sb.WriteString(fmt.Sprintf("- **[olympus constraint]** %s: %s\n", c.Pattern, c.Detection))
	}
	sb.WriteString("\n")
}

// formatKnowledgeMarkdown formats knowledge as markdown
func formatKnowledgeMarkdown(k *injectedKnowledge) string {
	var sb strings.Builder
	sb.WriteString("## Injected Knowledge (ol inject)\n\n")
	writeLearningsSection(&sb, k.Learnings)
	writePatternsSection(&sb, k.Patterns)
	writeSessionsSection(&sb, k.Sessions)
	writeConstraintsSection(&sb, k.OLConstraints)
	if len(k.Learnings) == 0 && len(k.Patterns) == 0 && len(k.Sessions) == 0 && len(k.OLConstraints) == 0 {
		sb.WriteString("*No prior knowledge found.*\n\n")
	}
	sb.WriteString(fmt.Sprintf("*Last injection: %s*\n", k.Timestamp.Format(time.RFC3339)))
	return sb.String()
}

// trimJSONToCharBudget truncates JSON output by progressively removing items
// from the knowledge struct until it fits the budget, then adds a "truncated" field.
func trimJSONToCharBudget(knowledge *injectedKnowledge, budget int) string {
	// Progressively trim: sessions first, then patterns, then learnings
	trimmed := *knowledge
	trimmed.Learnings = append([]learning(nil), knowledge.Learnings...)
	trimmed.Patterns = append([]pattern(nil), knowledge.Patterns...)
	trimmed.Sessions = append([]session(nil), knowledge.Sessions...)
	trimmed.OLConstraints = append([]olConstraint(nil), knowledge.OLConstraints...)

	type truncatedKnowledge struct {
		injectedKnowledge
		Truncated bool `json:"truncated"`
	}

	tryMarshal := func() string {
		tk := truncatedKnowledge{injectedKnowledge: trimmed, Truncated: true}
		data, err := json.MarshalIndent(tk, "", "  ")
		if err != nil {
			return "{\"truncated\": true}"
		}
		return string(data)
	}

	// Remove sessions first
	for len(trimmed.Sessions) > 0 {
		if out := tryMarshal(); len(out) <= budget {
			return out
		}
		trimmed.Sessions = trimmed.Sessions[:len(trimmed.Sessions)-1]
	}
	// Remove OL constraints
	for len(trimmed.OLConstraints) > 0 {
		if out := tryMarshal(); len(out) <= budget {
			return out
		}
		trimmed.OLConstraints = trimmed.OLConstraints[:len(trimmed.OLConstraints)-1]
	}
	// Remove patterns
	for len(trimmed.Patterns) > 0 {
		if out := tryMarshal(); len(out) <= budget {
			return out
		}
		trimmed.Patterns = trimmed.Patterns[:len(trimmed.Patterns)-1]
	}
	// Remove learnings
	for len(trimmed.Learnings) > 0 {
		if out := tryMarshal(); len(out) <= budget {
			return out
		}
		trimmed.Learnings = trimmed.Learnings[:len(trimmed.Learnings)-1]
	}

	return tryMarshal()
}

// trimToCharBudget truncates output to fit character budget
func trimToCharBudget(output string, budget int) string {
	if len(output) <= budget {
		return output
	}

	// Try to truncate at a section boundary
	lines := strings.Split(output, "\n")
	var result strings.Builder
	for _, line := range lines {
		if result.Len()+len(line)+1 > budget-50 { // Leave room for truncation marker
			break
		}
		result.WriteString(line)
		result.WriteString("\n")
	}

	result.WriteString("\n*[truncated to fit token budget]*\n")
	return result.String()
}

// truncateText truncates a string to max length with ellipsis
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// collectOLConstraints reads constraints from .ol/constraints/quarantine.json.
// Returns nil (no-op) if .ol/ directory doesn't exist.
func collectOLConstraints(cwd, query string) ([]olConstraint, error) {
	olDir := filepath.Join(cwd, ".ol")
	if _, err := os.Stat(olDir); os.IsNotExist(err) {
		return nil, nil // Not an Olympus project
	}

	quarantinePath := filepath.Join(olDir, quarantineRelPath)
	data, err := os.ReadFile(quarantinePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No quarantine file
		}
		return nil, fmt.Errorf("read quarantine.json: %w", err)
	}

	var raw []olConstraint
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse quarantine.json: %w", err)
	}

	// Filter by query if provided
	if query == "" {
		return raw, nil
	}

	queryLower := strings.ToLower(query)
	var filtered []olConstraint
	for _, c := range raw {
		content := strings.ToLower(c.Pattern + " " + c.Detection)
		if strings.Contains(content, queryLower) {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

// recordCitations records citation events for retrieved learnings.
// This is critical for closing the MemRL feedback loop (Phase 0).
// Citations link: session → learning → feedback → utility update.
func recordCitations(baseDir string, learnings []learning, sessionID, query string) error {
	canonicalSession := canonicalSessionID(sessionID)
	for _, l := range learnings {
		event := types.CitationEvent{
			ArtifactPath: canonicalArtifactPath(baseDir, l.Source),
			SessionID:    canonicalSession,
			CitedAt:      time.Now(),
			CitationType: "retrieved", // Will be upgraded to "applied" if session succeeds
			Query:        query,
		}

		if err := ratchet.RecordCitation(baseDir, event); err != nil {
			return fmt.Errorf("record citation for %s: %w", l.ID, err)
		}
	}
	return nil
}

// recordPatternCitations records citation events for retrieved patterns.
func recordPatternCitations(baseDir string, patterns []pattern, sessionID, query string) error {
	canonicalSession := canonicalSessionID(sessionID)
	for _, p := range patterns {
		if p.FilePath == "" {
			continue
		}
		event := types.CitationEvent{
			ArtifactPath: canonicalArtifactPath(baseDir, p.FilePath),
			SessionID:    canonicalSession,
			CitedAt:      time.Now(),
			CitationType: "retrieved",
			Query:        query,
		}
		if err := ratchet.RecordCitation(baseDir, event); err != nil {
			return fmt.Errorf("record citation for pattern %s: %w", p.Name, err)
		}
	}
	return nil
}
