package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Default directories to index under .agents/
var defaultIndexDirs = []string{
	".agents/learnings",
	".agents/research",
	".agents/plans",
	".agents/retros",
	".agents/patterns",
}

var dateFromFilenameRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)
var dateExtractRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// indexEntry represents one .md file's metadata for INDEX.md generation.
type indexEntry struct {
	Filename string `json:"filename"`
	Date     string `json:"date"`
	Summary  string `json:"summary"`
	Tags     string `json:"tags"`
}

// indexResult represents the result of indexing one directory.
type indexResult struct {
	Dir     string       `json:"dir"`
	Entries []indexEntry `json:"entries"`
	Written bool        `json:"written"`
	Error   string       `json:"error,omitempty"`
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Generate INDEX.md manifests for .agents/ directories",
	Long: `Generate INDEX.md manifest files for .agents/ knowledge directories.

Scans each directory for .md files, extracts frontmatter metadata (date,
summary, tags), and produces a markdown table sorted by date (newest first).

Directories indexed:
  .agents/learnings
  .agents/research
  .agents/plans
  .agents/retros
  .agents/patterns

Example:
  ao index              # Rebuild all INDEX.md files
  ao index --check      # Verify INDEX.md files are current
  ao index --dir .agents/learnings  # Rebuild one directory`,
	RunE: runIndex,
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().Bool("check", false, "Verify INDEX.md is current, exit 1 if stale")
	indexCmd.Flags().Bool("json", false, "Machine-readable output")
	indexCmd.Flags().String("dir", "", "Specific directory (default: all 5)")
	indexCmd.Flags().Bool("quiet", false, "Suppress non-error output")
}

func runIndex(cmd *cobra.Command, args []string) error {
	checkMode, _ := cmd.Flags().GetBool("check")
	jsonMode, _ := cmd.Flags().GetBool("json")
	dirFlag, _ := cmd.Flags().GetString("dir")
	quiet, _ := cmd.Flags().GetBool("quiet")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	dirs := defaultIndexDirs
	if dirFlag != "" {
		dirs = []string{dirFlag}
	}

	var results []indexResult
	stale := false

	for _, dir := range dirs {
		fullPath := filepath.Join(cwd, dir)
		info, err := os.Stat(fullPath)
		if err != nil || !info.IsDir() {
			VerbosePrintf("Warning: directory not found: %s\n", fullPath)
			continue
		}

		entries, err := scanDirectory(fullPath)
		if err != nil {
			VerbosePrintf("Warning: scanning %s: %v\n", dir, err)
			continue
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Date > entries[j].Date
		})

		if checkMode {
			isStale, msg := checkIndex(fullPath, dir, entries)
			if isStale {
				stale = true
				if !quiet {
					fmt.Fprintf(os.Stderr, "%s\n", msg)
				}
			} else if !quiet {
				fmt.Printf("%s: current (%d entries)\n", dir, len(entries))
			}
			results = append(results, indexResult{Dir: dir, Entries: entries, Written: false})
		} else {
			err := writeIndex(fullPath, dir, entries, GetDryRun())
			written := err == nil && !GetDryRun()
			result := indexResult{Dir: dir, Entries: entries, Written: written}
			if err != nil {
				result.Error = err.Error()
				fmt.Fprintf(os.Stderr, "Error writing INDEX.md for %s: %v\n", dir, err)
			} else if !quiet {
				fmt.Printf("%s: %d entries indexed\n", dir, len(entries))
			}
			results = append(results, result)
		}
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
	}

	if checkMode && stale {
		return fmt.Errorf("INDEX.md files are stale")
	}

	return nil
}

// scanDirectory reads all .md files (excluding INDEX.md) in a directory and
// extracts metadata from each.
func scanDirectory(dirPath string) ([]indexEntry, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var entries []indexEntry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, ".md") || name == "INDEX.md" {
			continue
		}

		filePath := filepath.Join(dirPath, name)
		entry := extractEntry(filePath, name)
		entries = append(entries, entry)
	}
	return entries, nil
}

// extractEntry reads a single .md file and extracts date, summary, and tags.
func extractEntry(filePath, filename string) indexEntry {
	entry := indexEntry{Filename: filename}

	content, err := os.ReadFile(filePath)
	if err != nil {
		VerbosePrintf("Warning: reading %s: %v\n", filePath, err)
		entry.Date = extractDateFromFilename(filename)
		entry.Summary = summaryFromFilename(filename)
		return entry
	}

	fm := parseFrontmatter(string(content))

	// Date: try frontmatter created_at, then date, then filename
	entry.Date = extractDateField(fm, filename)

	// Summary: frontmatter summary, then H1, then filename
	if s, ok := fm["summary"].(string); ok && s != "" {
		entry.Summary = s
	} else {
		entry.Summary = extractH1(string(content))
		if entry.Summary == "" {
			entry.Summary = summaryFromFilename(filename)
		}
	}

	// Tags
	entry.Tags = extractTagsFromFrontmatter(fm)

	// Clean for table
	entry.Summary = cleanForTable(entry.Summary)
	entry.Tags = cleanForTable(entry.Tags)

	return entry
}

// parseFrontmatter extracts YAML frontmatter between --- delimiters.
func parseFrontmatter(content string) map[string]interface{} {
	result := make(map[string]interface{})

	scanner := bufio.NewScanner(strings.NewReader(content))
	fmCount := 0
	var fmLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			fmCount++
			if fmCount == 2 {
				break
			}
			continue
		}
		if fmCount == 1 {
			fmLines = append(fmLines, line)
		}
	}

	if len(fmLines) == 0 {
		return result
	}

	fmText := strings.Join(fmLines, "\n")
	_ = yaml.Unmarshal([]byte(fmText), &result)
	return result
}

// extractDateField tries frontmatter fields then filename for a date string.
func extractDateField(fm map[string]interface{}, filename string) string {
	// Try created_at first, then date
	for _, field := range []string{"created_at", "date"} {
		if val, ok := fm[field]; ok {
			dateStr := fmt.Sprintf("%v", val)
			if m := dateExtractRe.FindString(dateStr); m != "" {
				return m
			}
			return dateStr
		}
	}
	return extractDateFromFilename(filename)
}

// extractDateFromFilename extracts YYYY-MM-DD from a filename prefix.
func extractDateFromFilename(filename string) string {
	if m := dateFromFilenameRe.FindStringSubmatch(filename); len(m) > 1 {
		return m[1]
	}
	return "unknown"
}

// summaryFromFilename derives a summary from the filename.
func summaryFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	// Remove date prefix if present
	if dateFromFilenameRe.MatchString(name) {
		name = name[11:] // len("YYYY-MM-DD-") == 11
		if len(name) == 0 {
			name = strings.TrimSuffix(filename, ".md")
		}
	}
	return name
}

// extractH1 returns the text of the first H1 heading in the content.
func extractH1(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

// extractTagsFromFrontmatter extracts tags as a space-separated string.
func extractTagsFromFrontmatter(fm map[string]interface{}) string {
	val, ok := fm["tags"]
	if !ok {
		return ""
	}

	switch v := val.(type) {
	case []interface{}:
		var tags []string
		for _, t := range v {
			tags = append(tags, fmt.Sprintf("%v", t))
		}
		return strings.Join(tags, " ")
	case string:
		// Handle inline "[tag1, tag2]" format
		s := strings.TrimSpace(v)
		s = strings.Trim(s, "[]")
		parts := strings.Split(s, ",")
		var tags []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				tags = append(tags, p)
			}
		}
		return strings.Join(tags, " ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// cleanForTable escapes pipe characters and normalizes whitespace.
func cleanForTable(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	// Normalize multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// titleCase capitalizes the first letter of a string.
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// writeIndex generates INDEX.md for a single directory.
func writeIndex(dirPath, relDir string, entries []indexEntry, dryRun bool) error {
	dirName := titleCase(filepath.Base(relDir))
	today := time.Now().Format("2006-01-02")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Index: %s\n\n", dirName))
	sb.WriteString(fmt.Sprintf("> Last rebuilt: %s | %d entries\n\n", today, len(entries)))
	sb.WriteString("## Entries\n\n")
	sb.WriteString("| File | Date | Summary | Tags |\n")
	sb.WriteString("|------|------|---------|------|\n")

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", e.Filename, e.Date, e.Summary, e.Tags))
	}

	if dryRun {
		VerbosePrintf("Would write INDEX.md to %s\n", filepath.Join(dirPath, "INDEX.md"))
		return nil
	}

	indexPath := filepath.Join(dirPath, "INDEX.md")
	return os.WriteFile(indexPath, []byte(sb.String()), 0644)
}

// checkIndex verifies INDEX.md is current by comparing file lists.
func checkIndex(dirPath, relDir string, entries []indexEntry) (bool, string) {
	indexPath := filepath.Join(dirPath, "INDEX.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return true, fmt.Sprintf("STALE %s: INDEX.md missing", relDir)
	}

	// Extract filenames from existing INDEX.md table rows
	existingFiles := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	headerPassed := false
	for scanner.Scan() {
		line := scanner.Text()
		// Skip until we find the table separator
		if strings.HasPrefix(line, "|------") || strings.HasPrefix(line, "|---") {
			headerPassed = true
			continue
		}
		if !headerPassed {
			continue
		}
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) >= 3 {
			fname := strings.TrimSpace(parts[1])
			if fname != "" && fname != "File" {
				existingFiles[fname] = true
			}
		}
	}

	// Compare: check for missing and extra files
	expectedFiles := make(map[string]bool)
	for _, e := range entries {
		expectedFiles[e.Filename] = true
	}

	var missing, extra []string
	for f := range expectedFiles {
		if !existingFiles[f] {
			missing = append(missing, f)
		}
	}
	for f := range existingFiles {
		if !expectedFiles[f] {
			extra = append(extra, f)
		}
	}

	if len(missing) == 0 && len(extra) == 0 && len(existingFiles) == len(expectedFiles) {
		return false, ""
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("STALE %s:", relDir))
	if len(missing) > 0 {
		sort.Strings(missing)
		msg.WriteString(fmt.Sprintf(" missing=[%s]", strings.Join(missing, ", ")))
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		msg.WriteString(fmt.Sprintf(" extra=[%s]", strings.Join(extra, ", ")))
	}
	return true, msg.String()
}
