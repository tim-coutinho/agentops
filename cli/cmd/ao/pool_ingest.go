package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boshu2/agentops/cli/internal/pool"
	"github.com/boshu2/agentops/cli/internal/taxonomy"
	"github.com/boshu2/agentops/cli/internal/types"
)

var (
	poolIngestDir string
)

type poolIngestResult struct {
	FilesScanned     int      `json:"files_scanned"`
	CandidatesFound  int      `json:"candidates_found"`
	Added            int      `json:"added"`
	SkippedExisting  int      `json:"skipped_existing"`
	SkippedMalformed int      `json:"skipped_malformed"`
	Errors           int      `json:"errors"`
	AddedIDs         []string `json:"added_ids,omitempty"`
}

var poolIngestCmd = &cobra.Command{
	Use:   "ingest [<files-or-globs...>]",
	Short: "Ingest pending markdown learnings into the pool",
	Long: `Ingest pending learnings into the quality pool.

This command bridges LLM-authored markdown learnings (typically written to
.agents/knowledge/pending/) into .agents/pool/pending/ as scored candidates.

If no args are provided, it ingests *.md from --dir (default: .agents/knowledge/pending)
and also scans legacy manual captures in .agents/knowledge/*.md.

Examples:
  ao pool ingest
  ao pool ingest --dir .agents/knowledge/pending
  ao pool ingest .agents/knowledge/pending/*.md
  ao pool ingest --dry-run -o json`,
	RunE: runPoolIngest,
}

func init() {
	poolCmd.AddCommand(poolIngestCmd)
	poolIngestCmd.Flags().StringVar(&poolIngestDir, "dir", filepath.Join(".agents", "knowledge", "pending"), "Directory to ingest from when no args are provided")
}

// ingestFileBlocks processes all learning blocks from one file, updating res.
// Returns true if any block had an add error (not skipped or malformed).
func ingestFileBlocks(p *pool.Pool, blocks []learningBlock, f string, fileDate time.Time, sessionHint string, res *poolIngestResult) bool {
	hadError := false
	for _, b := range blocks {
		cand, scoring, ok := buildCandidateFromLearningBlock(b, f, fileDate, sessionHint)
		if !ok {
			res.SkippedMalformed++
			continue
		}
		// Idempotency: skip if already present in any pool directory.
		if _, gerr := p.Get(cand.ID); gerr == nil {
			res.SkippedExisting++
			continue
		}
		if GetDryRun() {
			res.Added++
			res.AddedIDs = append(res.AddedIDs, cand.ID)
			continue
		}
		if err := p.AddAt(cand, scoring, cand.ExtractedAt); err != nil {
			res.Errors++
			hadError = true
			VerbosePrintf("Warning: add %s: %v\n", cand.ID, err)
			continue
		}
		res.Added++
		res.AddedIDs = append(res.AddedIDs, cand.ID)
	}
	return hadError
}

// moveIngestedFiles moves successfully processed files to the processed directory.
func moveIngestedFiles(cwd string, processedFiles []string) {
	processedDir := filepath.Join(cwd, ".agents", "knowledge", "processed")
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		VerbosePrintf("Warning: create processed dir: %v\n", err)
		return
	}
	for _, f := range processedFiles {
		dst := filepath.Join(processedDir, filepath.Base(f))
		if merr := os.Rename(f, dst); merr != nil {
			VerbosePrintf("Warning: move %s to processed: %v\n", filepath.Base(f), merr)
		}
	}
}

func runPoolIngest(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	p := pool.NewPool(cwd)

	files, err := resolveIngestFiles(cwd, poolIngestDir, args)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("No new files to ingest")
		return nil
	}

	res := poolIngestResult{FilesScanned: len(files)}
	var processedFiles []string

	for _, f := range files {
		data, rerr := os.ReadFile(f)
		if rerr != nil {
			res.Errors++
			VerbosePrintf("Warning: read %s: %v\n", filepath.Base(f), rerr)
			continue
		}

		fileDate, sessionHint := parsePendingFileHeader(string(data), f)
		blocks := parseLearningBlocks(string(data))
		res.CandidatesFound += len(blocks)

		hadError := ingestFileBlocks(p, blocks, f, fileDate, sessionHint, &res)
		if !hadError && !GetDryRun() {
			processedFiles = append(processedFiles, f)
		}
	}

	if len(processedFiles) > 0 {
		moveIngestedFiles(cwd, processedFiles)
	}

	return outputPoolIngestResult(res)
}

func outputPoolIngestResult(res poolIngestResult) error {
	switch GetOutput() {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	default:
		fmt.Printf("Ingested %d candidate(s) from %d file(s)\n", res.Added, res.FilesScanned)
		if res.SkippedExisting > 0 {
			fmt.Printf("Skipped (existing): %d\n", res.SkippedExisting)
		}
		if res.SkippedMalformed > 0 {
			fmt.Printf("Skipped (malformed): %d\n", res.SkippedMalformed)
		}
		if res.Errors > 0 {
			fmt.Printf("Errors: %d\n", res.Errors)
		}
		return nil
	}
}

func resolveIngestFiles(cwd, defaultDir string, args []string) ([]string, error) {
	var patterns []string
	if len(args) == 0 {
		patterns = []string{
			filepath.Join(cwd, defaultDir, "*.md"),
			// Legacy /learn captures were written directly to .agents/knowledge/.
			filepath.Join(cwd, ".agents", "knowledge", "*.md"),
		}
	} else {
		for _, a := range args {
			// Allow relative paths
			if !filepath.IsAbs(a) {
				patterns = append(patterns, filepath.Join(cwd, a))
			} else {
				patterns = append(patterns, a)
			}
		}
	}

	var files []string
	seen := make(map[string]bool)
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pat, err)
		}
		for _, m := range matches {
			if seen[m] {
				continue
			}
			seen[m] = true
			files = append(files, m)
		}
	}
	return files, nil
}

type learningBlock struct {
	Title      string
	ID         string
	Category   string
	Confidence string
	Body       string
}

var (
	reLearningHeader = regexp.MustCompile(`(?m)^# Learning:\s*(.+)\s*$`)
	// Support both "**ID**: X" and "**ID:** X" (colon outside vs inside bold).
	reIDLine         = regexp.MustCompile(`(?m)^\*\*ID:?\*\*:?\s*(.+)\s*$`)
	reCategoryLine   = regexp.MustCompile(`(?m)^\*\*Category:?\*\*:?\s*(.+)\s*$`)
	reConfidenceLine = regexp.MustCompile(`(?m)^\*\*Confidence:?\*\*:?\s*(.+)\s*$`)
	reFrontmatter    = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
	reDateMD         = regexp.MustCompile(`(?m)^\*\*Date:?\*\*:?\s*(\d{4}-\d{2}-\d{2})\s*$`)
	reDateYAML       = regexp.MustCompile(`(?m)^date:\s*(\d{4}-\d{2}-\d{2})\s*$`)
	reSessionHint    = regexp.MustCompile(`\bag-[a-z0-9]+\b`)
)

func parseLearningBlocks(md string) []learningBlock {
	locs := reLearningHeader.FindAllStringSubmatchIndex(md, -1)
	if len(locs) == 0 {
		if legacy, ok := parseLegacyFrontmatterLearning(md); ok {
			return []learningBlock{legacy}
		}
		return nil
	}

	var blocks []learningBlock
	for i, loc := range locs {
		// loc[0:2] is full match span, loc[2:4] is title group span.
		start := loc[0]
		end := len(md)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		title := strings.TrimSpace(md[loc[2]:loc[3]])
		body := strings.TrimSpace(md[start:end])

		b := learningBlock{
			Title: title,
			Body:  body,
		}

		if m := reIDLine.FindStringSubmatch(body); len(m) == 2 {
			b.ID = strings.TrimSpace(m[1])
		}
		if m := reCategoryLine.FindStringSubmatch(body); len(m) == 2 {
			b.Category = strings.TrimSpace(m[1])
		}
		if m := reConfidenceLine.FindStringSubmatch(body); len(m) == 2 {
			b.Confidence = strings.TrimSpace(m[1])
		}

		blocks = append(blocks, b)
	}
	return blocks
}

// parseYAMLFrontmatter parses a raw YAML frontmatter block into a string map.
func parseYAMLFrontmatter(raw string) map[string]string {
	fm := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		fm[key] = strings.Trim(val, `"'`)
	}
	return fm
}

// extractFirstHeadingText finds the first non-empty, non-heading-marker text line from body.
func extractFirstHeadingText(body string) string {
	for _, line := range strings.Split(body, "\n") {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		l = strings.TrimSpace(strings.TrimPrefix(l, "#"))
		if l != "" {
			return l
		}
	}
	return ""
}

func parseLegacyFrontmatterLearning(md string) (learningBlock, bool) {
	fmMatch := reFrontmatter.FindStringSubmatchIndex(md)
	if len(fmMatch) < 4 {
		return learningBlock{}, false
	}

	fmRaw := md[fmMatch[2]:fmMatch[3]]
	body := strings.TrimSpace(md[fmMatch[1]:])
	if body == "" {
		return learningBlock{}, false
	}

	frontmatter := parseYAMLFrontmatter(fmRaw)

	// Legacy /learn files include type/source/date frontmatter. Require type to
	// avoid treating arbitrary markdown files as candidates.
	category := strings.TrimSpace(frontmatter["type"])
	if category == "" {
		return learningBlock{}, false
	}

	title := extractFirstHeadingText(body)
	if title == "" {
		return learningBlock{}, false
	}

	return learningBlock{
		Title:      title,
		ID:         cmp.Or(strings.TrimSpace(frontmatter["id"]), "legacy"),
		Category:   category,
		Confidence: cmp.Or(strings.TrimSpace(frontmatter["confidence"]), "medium"),
		Body:       body,
	}, true
}

// dateStrategy is a function that attempts to extract a date from its inputs.
type dateStrategy func(md, path string) (time.Time, bool)

// dateFromFrontmatter extracts a date from YAML frontmatter.
func dateFromFrontmatter(md, _ string) (time.Time, bool) {
	fm := reFrontmatter.FindStringSubmatch(md)
	if len(fm) != 2 {
		return time.Time{}, false
	}
	m := reDateYAML.FindStringSubmatch(fm[1])
	if len(m) != 2 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(m[1]))
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// dateFromMarkdownField extracts a date from a **Date** markdown field.
func dateFromMarkdownField(md, _ string) (time.Time, bool) {
	m := reDateMD.FindStringSubmatch(md)
	if len(m) != 2 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(m[1]))
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// dateFromFilenamePrefix extracts a YYYY-MM-DD date from the filename prefix.
func dateFromFilenamePrefix(_, path string) (time.Time, bool) {
	base := filepath.Base(path)
	if len(base) < 10 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", base[:10])
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// dateFromFileMtime uses the file's modification time as a fallback.
func dateFromFileMtime(_, path string) (time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime().UTC(), true
}

// dateStrategies defines the ordered list of date extraction strategies.
var dateStrategies = []dateStrategy{
	dateFromFrontmatter,
	dateFromMarkdownField,
	dateFromFilenamePrefix,
	dateFromFileMtime,
}

func parsePendingFileHeader(md, path string) (fileDate time.Time, sessionHint string) {
	for _, strategy := range dateStrategies {
		if t, ok := strategy(md, path); ok {
			fileDate = t
			break
		}
	}
	if fileDate.IsZero() {
		fileDate = time.Now().UTC()
	}

	sessionHint = extractSessionHint(md, path)
	return fileDate, sessionHint
}

// extractSessionHint finds an ag-xxxx session ID in the first ~2KB of the content,
// falling back to the filename base.
func extractSessionHint(md, path string) string {
	head := md
	if len(head) > 2048 {
		head = head[:2048]
	}
	if m := reSessionHint.FindString(head); m != "" {
		return m
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func buildCandidateFromLearningBlock(b learningBlock, srcPath string, fileDate time.Time, sessionHint string) (types.Candidate, types.Scoring, bool) {
	if strings.TrimSpace(b.Title) == "" || strings.TrimSpace(b.Body) == "" {
		return types.Candidate{}, types.Scoring{}, false
	}

	// Stable ID: prefer (file base + learning ID). Otherwise fall back to a content hash.
	base := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	learningID := cmp.Or(strings.ToLower(strings.TrimSpace(b.ID)), "noid")

	id := slugify(fmt.Sprintf("pend-%s-%s-%s", base, sessionHint, learningID))
	if len(id) > 120 {
		// Keep a stable prefix, add a short hash to preserve uniqueness.
		h := sha256.Sum256([]byte(b.Body))
		id = slugify(id[:90] + "-" + hex.EncodeToString(h[:4]))
	}

	confDim := confidenceToScore(b.Confidence)
	rubric := computeRubricScores(b.Body, confDim)
	weighted := rubricWeightedSum(rubric, taxonomy.DefaultRubricWeights)
	raw := (taxonomy.GetBaseScore(types.KnowledgeTypeLearning) + weighted) / 2.0

	// Pending learnings already reflect some human/LLM filtering (they were written intentionally),
	// so bias score upwards based on the declared confidence to reduce false "bronze" assignments.
	switch strings.ToLower(strings.TrimSpace(b.Confidence)) {
	case "high":
		raw += 0.15
	case "medium":
		raw += 0.07
	}

	if raw > 1.0 {
		raw = 1.0
	}
	if raw < 0.0 {
		raw = 0.0
	}

	tier := taxonomy.AssignTier(raw, taxonomy.DefaultTierConfigs)
	gateRequired := taxonomy.RequiresHumanGate(tier, taxonomy.DefaultTierConfigs)

	cand := types.Candidate{
		ID:          id,
		Type:        types.KnowledgeTypeLearning,
		Content:     strings.TrimSpace(b.Body),
		Source:      types.Source{TranscriptPath: srcPath, Timestamp: fileDate, SessionID: sessionHint, MessageIndex: 0},
		RawScore:    raw,
		Tier:        tier,
		ExtractedAt: fileDate,
		Metadata: map[string]any{
			"pending_category":   b.Category,
			"pending_confidence": b.Confidence,
			"pending_title":      b.Title,
		},
		IsCurrent:    true,
		ExpiryStatus: types.ExpiryStatusActive,
		Utility:      types.InitialUtility,
		Maturity:     types.MaturityProvisional,
		Confidence:   taxonomy.GetConfidence(tier, taxonomy.DefaultTierConfigs),
		LastDecayAt:  fileDate,
		DecayCount:   0,
		HelpfulCount: 0,
		HarmfulCount: 0,
		RewardCount:  0,
		LastReward:   0,
		LastRewardAt: time.Time{},
		ValidUntil:   "",
		Location:     "",
		LocationPath: "",
	}

	scoring := types.Scoring{
		RawScore:       raw,
		TierAssignment: tier,
		Rubric:         rubric,
		GateRequired:   gateRequired,
		ScoredAt:       time.Now(),
	}

	return cand, scoring, true
}

func confidenceToScore(s string) float64 {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high":
		return 0.9
	case "medium":
		return 0.7
	case "low":
		return 0.5
	default:
		return 0.6
	}
}

func computeSpecificityScore(body, lower string) float64 {
	spec := 0.4
	if strings.Contains(body, "`") || strings.Contains(body, "```") {
		spec += 0.2
	}
	if regexp.MustCompile(`\d`).MatchString(body) {
		spec += 0.2
	}
	if regexp.MustCompile(`\b[a-zA-Z0-9_./-]+\.(go|ts|js|py|sh|yaml|yml|json|md)\b`).MatchString(body) {
		spec += 0.2
	}
	if strings.Contains(lower, "line ") {
		spec += 0.1
	}
	if spec > 1.0 {
		spec = 1.0
	}
	return spec
}

func computeActionabilityScore(body string) float64 {
	act := 0.4
	if regexp.MustCompile(`(?m)^\s*[-*]\s+`).MatchString(body) {
		act += 0.2
	}
	if regexp.MustCompile(`(?i)\b(run|add|remove|use|ensure|check|grep|rg|fix|avoid|prefer|must|should)\b`).MatchString(body) {
		act += 0.2
	}
	if strings.Contains(body, "```") {
		act += 0.2
	}
	if act > 1.0 {
		act = 1.0
	}
	return act
}

func computeNoveltyScore(body string) float64 {
	nov := 0.5
	if len(body) > 800 {
		nov += 0.1
	}
	if len(body) < 250 {
		nov -= 0.1
	}
	if nov > 1.0 {
		nov = 1.0
	}
	if nov < 0.0 {
		nov = 0.0
	}
	return nov
}

func computeContextScore(lower string) float64 {
	ctx := 0.5
	if strings.Contains(lower, "## source") || strings.Contains(lower, "**source**") {
		ctx += 0.2
	}
	if strings.Contains(lower, "## why it matters") {
		ctx += 0.1
	}
	if ctx > 1.0 {
		ctx = 1.0
	}
	return ctx
}

func computeRubricScores(body string, confidence float64) types.RubricScores {
	lower := strings.ToLower(body)
	return types.RubricScores{
		Specificity:   computeSpecificityScore(body, lower),
		Actionability: computeActionabilityScore(body),
		Novelty:       computeNoveltyScore(body),
		Context:       computeContextScore(lower),
		Confidence:    confidence,
	}
}

func rubricWeightedSum(r types.RubricScores, w taxonomy.RubricWeights) float64 {
	return r.Specificity*w.Specificity +
		r.Actionability*w.Actionability +
		r.Novelty*w.Novelty +
		r.Context*w.Context +
		r.Confidence*w.Confidence
}

// isSlugAlphanumeric returns true if the rune should be kept as-is in a slug.
func isSlugAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if isSlugAlphanumeric(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return cmp.Or(strings.Trim(b.String(), "-"), "cand")
}
