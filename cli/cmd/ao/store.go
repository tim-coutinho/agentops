package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/boshu2/agentops/cli/internal/types"
)

var (
	storeLimit      int
	storeCategorize bool
)

const (
	// IndexFileName is the name of the search index file.
	IndexFileName = "search-index.jsonl"

	// IndexDir is the directory for index files.
	IndexDir = ".agents/ao/index"
)

// IndexEntry represents a single entry in the search index.
type IndexEntry struct {
	// Path is the absolute path to the artifact.
	Path string `json:"path"`

	// ID is the artifact identifier.
	ID string `json:"id"`

	// Type is the artifact type (learning, pattern, research, etc).
	Type string `json:"type"`

	// Title is the artifact title or first line.
	Title string `json:"title"`

	// Content is the full text content for search.
	Content string `json:"content"`

	// Keywords are extracted keywords for search.
	Keywords []string `json:"keywords,omitempty"`

	// Category is the artifact's category (best-effort), when --categorize is enabled.
	Category string `json:"category,omitempty"`

	// Tags are best-effort extracted tags, when --categorize is enabled.
	Tags []string `json:"tags,omitempty"`

	// Utility is the MemRL utility score.
	Utility float64 `json:"utility,omitempty"`

	// Maturity is the CASS maturity level.
	Maturity string `json:"maturity,omitempty"`

	// IndexedAt is when this entry was indexed.
	IndexedAt time.Time `json:"indexed_at"`

	// ModifiedAt is when the source file was last modified.
	ModifiedAt time.Time `json:"modified_at"`
}

// SearchResult represents a search match.
type SearchResult struct {
	Entry    IndexEntry `json:"entry"`
	Score    float64    `json:"score"`
	Snippet  string     `json:"snippet,omitempty"`
}

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "STORE phase - index for retrieval",
	Long: `The STORE phase indexes artifacts for retrieval and search.

In the metallurgical metaphor:
  FORGE  → Extract raw knowledge from transcripts
  TEMPER → Validate, harden, and lock for storage
  STORE  → Index for retrieval and search

The store command manages the knowledge index that enables fast
semantic search across all artifacts.

Commands:
  index    Add files to the search index
  search   Query the index
  rebuild  Rebuild index from .agents/`,
}

func init() {
	rootCmd.AddCommand(storeCmd)

	// index subcommand
		indexCmd := &cobra.Command{
		Use:   "index <files...>",
		Short: "Add files to search index",
		Long: `Add artifacts to the search index.

Indexes:
  - Full text content
  - Extracted keywords
  - MemRL utility scores
  - CASS maturity levels

Examples:
  ao store index .agents/learnings/*.md
  ao store index .agents/patterns/error-handling.md
  ao store index --rebuild .agents/`,
		Args: cobra.MinimumNArgs(1),
			RunE: runStoreIndex,
		}
		indexCmd.Flags().BoolVar(&storeCategorize, "categorize", false, "Extract and store category/tags for retrieval")
		storeCmd.AddCommand(indexCmd)

	// search subcommand
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the index",
		Long: `Search for artifacts matching a query.

Returns results ranked by relevance with snippets.

Examples:
  ao store search "mutex pattern"
  ao store search "error handling" --limit 5
  ao store search "authentication" -o json`,
		Args: cobra.ExactArgs(1),
		RunE: runStoreSearch,
	}
	searchCmd.Flags().IntVar(&storeLimit, "limit", 10, "Maximum results to return")
	storeCmd.AddCommand(searchCmd)

	// rebuild subcommand
		rebuildCmd := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild search index",
		Long: `Rebuild the search index from scratch.

Scans all .agents/ directories and re-indexes:
  - learnings/
  - patterns/
  - research/
  - retros/

Examples:
  ao store rebuild
  ao store rebuild --verbose`,
			RunE: runStoreRebuild,
		}
		rebuildCmd.Flags().BoolVar(&storeCategorize, "categorize", false, "Extract and store category/tags for retrieval")
		storeCmd.AddCommand(rebuildCmd)

	// stats subcommand
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show index statistics",
		Long: `Display statistics about the search index.

Shows:
  - Total indexed entries
  - Breakdown by type
  - Index freshness
  - Coverage metrics

Examples:
  ao store stats
  ao store stats -o json`,
		RunE: runStoreStats,
	}
	storeCmd.AddCommand(statsCmd)
}

func runStoreIndex(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Expand file patterns
	files, err := expandFilePatterns(cwd, args)
	if err != nil {
		return fmt.Errorf("expand patterns: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found matching patterns")
	}

	if GetDryRun() {
		fmt.Printf("[dry-run] Would index %d file(s)\n", len(files))
		for _, f := range files {
			fmt.Printf("  - %s\n", f)
		}
		return nil
	}

	indexed := 0
	for _, path := range files {
		entry, err := createIndexEntry(path, storeCategorize)
		if err != nil {
			VerbosePrintf("Warning: skip %s: %v\n", filepath.Base(path), err)
			continue
		}

		if err := appendToIndex(cwd, entry); err != nil {
			VerbosePrintf("Warning: index %s: %v\n", filepath.Base(path), err)
			continue
		}

		indexed++
		VerbosePrintf("Indexed: %s\n", filepath.Base(path))
	}

	fmt.Printf("Indexed %d artifact(s)\n", indexed)
	return nil
}

func runStoreSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	results, err := searchIndex(cwd, query, storeLimit)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	switch GetOutput() {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)

	case "yaml":
		enc := yaml.NewEncoder(os.Stdout)
		return enc.Encode(results)

	default:
		printSearchResults(query, results)
	}

	return nil
}

func runStoreRebuild(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if GetDryRun() {
		fmt.Println("[dry-run] Would rebuild search index")
		return nil
	}

	// Remove existing index
	indexPath := filepath.Join(cwd, IndexDir, IndexFileName)
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		VerbosePrintf("Warning: remove old index: %v\n", err)
	}

	// Scan all artifact directories
	dirs := []string{
		filepath.Join(cwd, ".agents", "learnings"),
		filepath.Join(cwd, ".agents", "patterns"),
		filepath.Join(cwd, ".agents", "research"),
		filepath.Join(cwd, ".agents", "retros"),
		filepath.Join(cwd, ".agents", "candidates"),
	}

	var files []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".jsonl") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			VerbosePrintf("Warning: scan %s: %v\n", dir, err)
		}
	}

	indexed := 0
	for _, path := range files {
		entry, err := createIndexEntry(path, storeCategorize)
		if err != nil {
			VerbosePrintf("Warning: skip %s: %v\n", filepath.Base(path), err)
			continue
		}

		if err := appendToIndex(cwd, entry); err != nil {
			VerbosePrintf("Warning: index %s: %v\n", filepath.Base(path), err)
			continue
		}

		indexed++
	}

	fmt.Printf("Rebuilt index: %d artifacts\n", indexed)
	return nil
}

func runStoreStats(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	stats, err := computeIndexStats(cwd)
	if err != nil {
		return fmt.Errorf("compute stats: %w", err)
	}

	switch GetOutput() {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(stats)

	case "yaml":
		enc := yaml.NewEncoder(os.Stdout)
		return enc.Encode(stats)

	default:
		printIndexStats(stats)
	}

	return nil
}

// createIndexEntry creates an index entry from a file.
func createIndexEntry(path string, categorize bool) (*IndexEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(content)

	// Determine type from path
	var artifactType string
	switch {
	case strings.Contains(path, "/learnings/"):
		artifactType = "learning"
	case strings.Contains(path, "/patterns/"):
		artifactType = "pattern"
	case strings.Contains(path, "/research/"):
		artifactType = "research"
	case strings.Contains(path, "/retros/"):
		artifactType = "retro"
	case strings.Contains(path, "/candidates/"):
		artifactType = "candidate"
	default:
		artifactType = "unknown"
	}

	// Extract title from first heading
	title := extractTitle(text)

	// Extract keywords
	keywords := extractKeywords(text)

	var category string
	var tags []string
	if categorize {
		category, tags = extractCategoryAndTags(text)
		if category != "" {
			keywords = append(keywords, strings.ToLower(category))
		}
		for _, t := range tags {
			tt := strings.TrimSpace(t)
			if tt != "" {
				keywords = append(keywords, strings.ToLower(tt))
			}
		}
	}

	// Parse MemRL metadata if present
	utility, maturity := parseMemRLMetadata(text)

	entry := &IndexEntry{
		Path:       path,
		ID:         filepath.Base(path),
		Type:       artifactType,
		Title:      title,
		Content:    text,
		Keywords:   keywords,
		Category:   category,
		Tags:       tags,
		Utility:    utility,
		Maturity:   maturity,
		IndexedAt:  time.Now(),
		ModifiedAt: info.ModTime(),
	}

	return entry, nil
}

// appendToIndex adds an entry to the index file.
func appendToIndex(baseDir string, entry *IndexEntry) error {
	indexDir := filepath.Join(baseDir, IndexDir)
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return err
	}

	indexPath := filepath.Join(indexDir, IndexFileName)

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // write complete, close best-effort
	}()

	_, err = f.Write(append(data, '\n'))
	return err
}

// searchIndex searches the index for matching entries.
func searchIndex(baseDir, query string, limit int) ([]SearchResult, error) {
	indexPath := filepath.Join(baseDir, IndexDir, IndexFileName)

	f, err := os.Open(indexPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("index not found - run 'ao store rebuild' first")
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only index search, close error non-fatal
	}()

	queryTerms := strings.Fields(strings.ToLower(query))
	var results []SearchResult

	scanner := bufio.NewScanner(f)
	// Increase buffer size for large entries
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var entry IndexEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		score := computeSearchScore(entry, queryTerms)
		if score > 0 {
			snippet := createSearchSnippet(entry.Content, query, 150)
			results = append(results, SearchResult{
				Entry:   entry,
				Score:   score,
				Snippet: snippet,
			})
		}
	}

	// Sort by score (descending) then by utility (descending)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Entry.Utility > results[j].Entry.Utility
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, scanner.Err()
}

// computeSearchScore calculates relevance score for a query.
func computeSearchScore(entry IndexEntry, queryTerms []string) float64 {
	var score float64

	lowerContent := strings.ToLower(entry.Content)
	lowerTitle := strings.ToLower(entry.Title)

	for _, term := range queryTerms {
		// Title matches are worth more
		if strings.Contains(lowerTitle, term) {
			score += 3.0
		}

		// Content matches
		if strings.Contains(lowerContent, term) {
			score += 1.0
		}

		// Keyword matches
		for _, kw := range entry.Keywords {
			if strings.Contains(strings.ToLower(kw), term) {
				score += 2.0
				break
			}
		}
	}

	// Boost by utility (MemRL integration)
	// Lambda = 0.5 (balanced weighting)
	lambda := types.DefaultLambda
	if entry.Utility > 0 {
		score = (1-lambda)*score + lambda*entry.Utility*score
	}

	return score
}

// extractTitle gets the title from markdown content.
func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	// Fall back to first non-empty line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "---") {
			if len(line) > 80 {
				return line[:77] + "..."
			}
			return line
		}
	}
	return "Untitled"
}

// extractKeywords extracts keywords from content.
func extractKeywords(content string) []string {
	keywords := make(map[string]bool)

	// Look for common patterns
	patterns := []string{
		"pattern:", "solution:", "learning:", "decision:",
		"fix:", "issue:", "error:", "warning:",
		"config:", "setup:", "install:", "deploy:",
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range patterns {
		if strings.Contains(lowerContent, pattern) {
			keywords[strings.TrimSuffix(pattern, ":")] = true
		}
	}

	// Extract from metadata lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**Tags**:") || strings.HasPrefix(line, "**Keywords**:") {
			tags := strings.TrimPrefix(strings.TrimPrefix(line, "**Tags**:"), "**Keywords**:")
			for _, tag := range strings.Split(tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					keywords[tag] = true
				}
			}
		}
	}

	var result []string
	for kw := range keywords {
		result = append(result, kw)
	}
	return result
}

// extractCategoryAndTags tries to derive category/tags from either YAML frontmatter or common markdown metadata lines.
// Best-effort: missing/malformed metadata should not fail indexing.
func extractCategoryAndTags(content string) (category string, tags []string) {
	lines := strings.Split(content, "\n")

	// YAML frontmatter (only if it starts at top of file).
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "---" {
				break
			}
			if strings.HasPrefix(line, "category:") {
				category = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "category:")), "\"'")
				continue
			}
			if strings.HasPrefix(line, "tags:") {
				rest := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
				// Support: tags: [a, b, c]
				if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
					inner := strings.TrimSuffix(strings.TrimPrefix(rest, "["), "]")
					for _, t := range strings.Split(inner, ",") {
						tt := strings.TrimSpace(strings.Trim(t, "\"'"))
						if tt != "" {
							tags = append(tags, tt)
						}
					}
				}
			}
		}
	}

	// Markdown metadata lines (learning format).
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if category == "" && strings.HasPrefix(line, "**Category**:") {
			category = strings.TrimSpace(strings.TrimPrefix(line, "**Category**:"))
			continue
		}
		if strings.HasPrefix(line, "**Tags**:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "**Tags**:"))
			for _, t := range strings.Split(rest, ",") {
				tt := strings.TrimSpace(t)
				if tt != "" {
					tags = append(tags, tt)
				}
			}
		}
	}

	return category, tags
}

// parseMemRLMetadata extracts utility and maturity from content.
func parseMemRLMetadata(content string) (utility float64, maturity string) {
	utility = types.InitialUtility
	maturity = "provisional"

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "**Utility**:") || strings.HasPrefix(line, "- **Utility**:") {
			utilStr := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "**Utility**:"), "- **Utility**:"))
			//nolint:errcheck // parsing optional metadata, zero value is acceptable default
			fmt.Sscanf(utilStr, "%f", &utility)
		}
		if strings.HasPrefix(line, "**Maturity**:") || strings.HasPrefix(line, "- **Maturity**:") {
			maturity = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "**Maturity**:"), "- **Maturity**:"))
		}
	}

	return utility, maturity
}

// createSearchSnippet creates a context snippet around query matches.
func createSearchSnippet(content, query string, maxLen int) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	// Find first occurrence
	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		// Try first query term
		terms := strings.Fields(lowerQuery)
		if len(terms) > 0 {
			idx = strings.Index(lowerContent, terms[0])
		}
	}

	if idx == -1 {
		// Return start of content
		if len(content) > maxLen {
			return content[:maxLen-3] + "..."
		}
		return content
	}

	// Extract window around match
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + maxLen
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]

	// Clean up
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	snippet = strings.TrimSpace(snippet)

	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet += "..."
	}

	return snippet
}

// IndexStats holds index statistics.
type IndexStats struct {
	TotalEntries int            `json:"total_entries"`
	ByType       map[string]int `json:"by_type"`
	MeanUtility  float64        `json:"mean_utility"`
	OldestEntry  time.Time      `json:"oldest_entry"`
	NewestEntry  time.Time      `json:"newest_entry"`
	IndexPath    string         `json:"index_path"`
}

// computeIndexStats calculates index statistics.
func computeIndexStats(baseDir string) (*IndexStats, error) {
	indexPath := filepath.Join(baseDir, IndexDir, IndexFileName)

	stats := &IndexStats{
		ByType:    make(map[string]int),
		IndexPath: indexPath,
	}

	f, err := os.Open(indexPath)
	if os.IsNotExist(err) {
		return stats, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only index stats, close error non-fatal
	}()

	var totalUtility float64
	var utilityCount int

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var entry IndexEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		stats.TotalEntries++
		stats.ByType[entry.Type]++

		if entry.Utility > 0 {
			totalUtility += entry.Utility
			utilityCount++
		}

		if stats.OldestEntry.IsZero() || entry.IndexedAt.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.IndexedAt
		}
		if entry.IndexedAt.After(stats.NewestEntry) {
			stats.NewestEntry = entry.IndexedAt
		}
	}

	if utilityCount > 0 {
		stats.MeanUtility = totalUtility / float64(utilityCount)
	}

	return stats, scanner.Err()
}

// printSearchResults prints search results in table format.
func printSearchResults(query string, results []SearchResult) {
	fmt.Println()
	fmt.Printf("Search Results for: %s\n", query)
	fmt.Println("======================")
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("No results found")
		return
	}

	for i, r := range results {
		fmt.Printf("%d. %s [%s]\n", i+1, r.Entry.Title, r.Entry.Type)
		fmt.Printf("   Score: %.2f | Utility: %.2f\n", r.Score, r.Entry.Utility)
		fmt.Printf("   Path: %s\n", r.Entry.Path)
		if r.Snippet != "" {
			fmt.Printf("   %s\n", r.Snippet)
		}
		fmt.Println()
	}
}

// printIndexStats prints index statistics.
func printIndexStats(stats *IndexStats) {
	fmt.Println()
	fmt.Println("Search Index Statistics")
	fmt.Println("=======================")
	fmt.Println()

	fmt.Printf("Total entries: %d\n", stats.TotalEntries)
	fmt.Printf("Mean utility:  %.2f\n", stats.MeanUtility)
	fmt.Printf("Index path:    %s\n", stats.IndexPath)
	fmt.Println()

	if len(stats.ByType) > 0 {
		fmt.Println("By Type:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for t, count := range stats.ByType {
			//nolint:errcheck // CLI tabwriter output to stdout
			fmt.Fprintf(w, "  %s:\t%d\n", t, count)
		}
		_ = w.Flush()
	}

	if !stats.OldestEntry.IsZero() {
		fmt.Println()
		fmt.Printf("Oldest indexed: %s\n", stats.OldestEntry.Format("2006-01-02 15:04"))
		fmt.Printf("Newest indexed: %s\n", stats.NewestEntry.Format("2006-01-02 15:04"))
	}
}
