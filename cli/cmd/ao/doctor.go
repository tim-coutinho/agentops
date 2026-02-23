package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boshu2/agentops/cli/internal/storage"
)

var doctorJSON bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check AgentOps health",
	Long: `Run health checks on your AgentOps installation.

Validates that all required components are present and configured.
Optional components are reported as warnings but do not cause failure.

Examples:
  ao doctor
  ao doctor --json`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON")
	rootCmd.AddCommand(doctorCmd)
}

type doctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // "pass", "warn", "fail"
	Detail   string `json:"detail"`
	Required bool   `json:"required"`
}

type doctorOutput struct {
	Checks  []doctorCheck `json:"checks"`
	Result  string        `json:"result"` // "HEALTHY", "DEGRADED", "UNHEALTHY"
	Summary string        `json:"summary"`
}

// gatherDoctorChecks runs all doctor checks and returns the results.
func gatherDoctorChecks() []doctorCheck {
	return []doctorCheck{
		{Name: "ao CLI", Status: "pass", Detail: fmt.Sprintf("v%s", version), Required: true},
		checkCLIDependencies(),
		checkHookCoverage(),
		checkKnowledgeBase(),
		checkKnowledgeFreshness(),
		checkSearchIndex(),
		checkFlywheelHealth(),
		checkSkills(),
		checkOptionalCLI("codex", "needed for --mixed council"),
	}
}

// doctorStatusIcon returns the display icon for a check status.
func doctorStatusIcon(status string) string {
	switch status {
	case "pass":
		return "\u2713"
	case "warn":
		return "!"
	case "fail":
		return "\u2717"
	}
	return "?"
}

// renderDoctorTable writes the formatted doctor output table.
func renderDoctorTable(w io.Writer, output doctorOutput) {
	fmt.Fprintln(w, "ao doctor")
	fmt.Fprintln(w, "\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")

	maxName := 0
	for _, c := range output.Checks {
		if len(c.Name) > maxName {
			maxName = len(c.Name)
		}
	}

	for _, c := range output.Checks {
		padding := strings.Repeat(" ", maxName-len(c.Name))
		fmt.Fprintf(w, "%s %s%s  %s\n", doctorStatusIcon(c.Status), c.Name, padding, c.Detail)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", output.Summary)
}

// hasRequiredFailure returns true if any required check has failed.
func hasRequiredFailure(checks []doctorCheck) bool {
	for _, c := range checks {
		if c.Required && c.Status == "fail" {
			return true
		}
	}
	return false
}

func runDoctor(cmd *cobra.Command, args []string) error {
	output := computeResult(gatherDoctorChecks())
	w := cmd.OutOrStdout()

	if doctorJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal doctor output: %w", err)
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	renderDoctorTable(w, output)

	if hasRequiredFailure(output.Checks) {
		return fmt.Errorf("doctor failed: one or more required checks did not pass")
	}

	return nil
}

// checkCLIDependencies verifies gt and bd are available in PATH.
func checkCLIDependencies() doctorCheck {
	gtOk := false
	bdOk := false

	if _, err := exec.LookPath("gt"); err == nil {
		gtOk = true
	}
	if _, err := exec.LookPath("bd"); err == nil {
		bdOk = true
	}

	if gtOk && bdOk {
		return doctorCheck{
			Name:     "CLI Dependencies",
			Status:   "pass",
			Detail:   "gt and bd available",
			Required: false,
		}
	}

	var missing []string
	var hints []string
	if !gtOk {
		missing = append(missing, "gt")
		hints = append(hints, "install with 'brew install gastown'")
	}
	if !bdOk {
		missing = append(missing, "bd")
		hints = append(hints, "install with 'brew install beads'")
	}

	return doctorCheck{
		Name:     "CLI Dependencies",
		Status:   "warn",
		Detail:   fmt.Sprintf("%s not found \u2014 %s", strings.Join(missing, ", "), strings.Join(hints, "; ")),
		Required: false,
	}
}

// checkHookCoverage checks if Claude hooks are installed with event coverage.
func checkHookCoverage() doctorCheck {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return doctorCheck{Name: "Hook Coverage", Status: "fail", Detail: "cannot determine home directory", Required: true}
	}

	// Prefer settings.json (active Claude configuration).
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		if hooksMap, ok := extractHooksMap(data); ok {
			return evaluateHookCoverage(hooksMap)
		}
	}

	// Fallback: standalone hooks.json format.
	hooksPath := filepath.Join(homeDir, ".claude", "hooks.json")
	if data, err := os.ReadFile(hooksPath); err == nil {
		if hooksMap, ok := extractHooksMap(data); ok {
			return evaluateHookCoverage(hooksMap)
		}
	}

	return doctorCheck{
		Name:     "Hook Coverage",
		Status:   "warn",
		Detail:   "No hooks found \u2014 run 'ao hooks install --force'",
		Required: false,
	}
}

func evaluateHookCoverage(hooksMap map[string]any) doctorCheck {
	installedEvents := countInstalledEvents(hooksMap)
	if installedEvents == 0 {
		return doctorCheck{
			Name:     "Hook Coverage",
			Status:   "warn",
			Detail:   "No hooks found \u2014 run 'ao hooks install --force'",
			Required: false,
		}
	}

	if !hookGroupContainsAo(hooksMap, "SessionStart") {
		return doctorCheck{
			Name:     "Hook Coverage",
			Status:   "warn",
			Detail:   "Non-ao hooks detected \u2014 run 'ao hooks install --force'",
			Required: false,
		}
	}

	if installedEvents < len(AllEventNames()) {
		return doctorCheck{
			Name:     "Hook Coverage",
			Status:   "warn",
			Detail:   fmt.Sprintf("Partial coverage: %d/%d events \u2014 run 'ao hooks install --force'", installedEvents, len(AllEventNames())),
			Required: false,
		}
	}

	return doctorCheck{
		Name:     "Hook Coverage",
		Status:   "pass",
		Detail:   fmt.Sprintf("Full coverage: %d/%d events", installedEvents, len(AllEventNames())),
		Required: false,
	}
}

func extractHooksMap(data []byte) (map[string]any, bool) {
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, false
	}

	// settings.json shape
	if hooksRaw, ok := parsed["hooks"]; ok {
		if hooksMap, ok := hooksRaw.(map[string]any); ok {
			return hooksMap, true
		}
	}

	// hooks.json shape with top-level events
	for _, event := range AllEventNames() {
		if _, ok := parsed[event]; ok {
			return parsed, true
		}
	}

	return nil, false
}

func countHooksInMap(raw any) int {
	count := 0
	switch v := raw.(type) {
	case map[string]any:
		for _, val := range v {
			if arr, ok := val.([]any); ok {
				count += len(arr)
			} else {
				// Recurse into nested maps
				count += countHooksInMap(val)
			}
		}
	case []any:
		count += len(v)
	}
	return count
}

func countInstalledEvents(hooksMap map[string]any) int {
	installed := 0
	for _, event := range AllEventNames() {
		if groups, ok := hooksMap[event].([]any); ok && len(groups) > 0 {
			installed++
		}
	}
	return installed
}

// checkKnowledgeBase checks that the .agents/ao directory exists.
func checkKnowledgeBase() doctorCheck {
	cwd, err := os.Getwd()
	if err != nil {
		return doctorCheck{Name: "Knowledge Base", Status: "fail", Detail: "cannot determine working directory", Required: true}
	}

	baseDir := filepath.Join(cwd, storage.DefaultBaseDir)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return doctorCheck{Name: "Knowledge Base", Status: "fail", Detail: ".agents/ao not initialized", Required: true}
	}

	return doctorCheck{Name: "Knowledge Base", Status: "pass", Detail: ".agents/ao initialized", Required: true}
}

// newestFileModTime returns the most recent modification time among regular files in entries.
// Returns zero time if no regular files are found.
func newestFileModTime(entries []os.DirEntry) time.Time {
	var newest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	return newest
}

// checkKnowledgeFreshness checks the most recent file in .agents/ao/sessions/.
func checkKnowledgeFreshness() doctorCheck {
	cwd, err := os.Getwd()
	if err != nil {
		return doctorCheck{Name: "Knowledge Freshness", Status: "warn", Detail: "cannot determine working directory", Required: false}
	}

	noSessionsCheck := doctorCheck{
		Name:     "Knowledge Freshness",
		Status:   "warn",
		Detail:   "No sessions found \u2014 run 'ao forge transcript' after your next session",
		Required: false,
	}

	sessionsDir := filepath.Join(cwd, storage.DefaultBaseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil || len(entries) == 0 {
		return noSessionsCheck
	}

	newest := newestFileModTime(entries)
	if newest.IsZero() {
		return noSessionsCheck
	}

	age := time.Since(newest)
	ageStr := formatDuration(age)

	if age > 14*24*time.Hour {
		return doctorCheck{
			Name:     "Knowledge Freshness",
			Status:   "warn",
			Detail:   fmt.Sprintf("Last session: %s ago \u2014 knowledge may be stale", ageStr),
			Required: false,
		}
	}

	return doctorCheck{
		Name:     "Knowledge Freshness",
		Status:   "pass",
		Detail:   fmt.Sprintf("Last session: %s ago", ageStr),
		Required: false,
	}
}

// formatDuration produces a human-readable duration string like "2h", "5d", "3m".
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// checkSearchIndex checks if the search index exists and counts terms.
func checkSearchIndex() doctorCheck {
	cwd, err := os.Getwd()
	if err != nil {
		return doctorCheck{Name: "Search Index", Status: "warn", Detail: "cannot determine working directory", Required: false}
	}

	indexPath := filepath.Join(cwd, IndexDir, IndexFileName)
	info, err := os.Stat(indexPath)
	if err != nil {
		return doctorCheck{
			Name:     "Search Index",
			Status:   "warn",
			Detail:   "No search index \u2014 run 'ao store rebuild' for faster searches",
			Required: false,
		}
	}

	if info.Size() == 0 {
		return doctorCheck{
			Name:     "Search Index",
			Status:   "warn",
			Detail:   "Search index is empty \u2014 run 'ao store rebuild'",
			Required: false,
		}
	}

	// Count lines (each line is a term/entry)
	lines := countFileLines(indexPath)

	return doctorCheck{
		Name:     "Search Index",
		Status:   "pass",
		Detail:   fmt.Sprintf("Index exists (%s terms)", formatNumber(lines)),
		Required: false,
	}
}

// countFileLines counts non-empty lines in a file.
func countFileLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	// Increase buffer for potentially long JSONL lines
	scanner.Buffer(make([]byte, 256*1024), 1024*1024)
	for scanner.Scan() {
		if len(strings.TrimSpace(scanner.Text())) > 0 {
			count++
		}
	}
	return count
}

// formatNumber adds comma separators to an integer (e.g., 1247 -> "1,247").
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// checkFlywheelHealth checks if .agents/ao/learnings/ has files.
func checkFlywheelHealth() doctorCheck {
	cwd, err := os.Getwd()
	if err != nil {
		return doctorCheck{Name: "Flywheel Health", Status: "warn", Detail: "cannot determine working directory", Required: false}
	}

	learningsDir := filepath.Join(cwd, storage.DefaultBaseDir, "learnings")
	total := countFiles(learningsDir)

	if total == 0 {
		// Also check the older path
		altDir := filepath.Join(cwd, ".agents", "learnings")
		total = countFiles(altDir)
	}

	if total == 0 {
		return doctorCheck{
			Name:     "Flywheel Health",
			Status:   "warn",
			Detail:   "No learnings found \u2014 the flywheel hasn't started",
			Required: false,
		}
	}

	// Count established learnings (those with "established" or "promoted" in filename or content)
	established := countEstablished(filepath.Join(cwd, storage.DefaultBaseDir, "learnings"))
	if established == 0 {
		// Check alt path too
		established = countEstablished(filepath.Join(cwd, ".agents", "learnings"))
	}

	detail := fmt.Sprintf("%d learnings in flywheel", total)
	if established > 0 {
		detail = fmt.Sprintf("%d learnings (%d established)", total, established)
	}

	return doctorCheck{
		Name:     "Flywheel Health",
		Status:   "pass",
		Detail:   detail,
		Required: false,
	}
}

// countEstablished counts files in a directory whose name contains "established" or "promoted".
func countEstablished(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.Contains(lower, "established") || strings.Contains(lower, "promoted") {
			count++
		}
	}
	return count
}

func checkSkills() doctorCheck {
	// Skills are installed globally at ~/.claude/skills/, not in the local repo.
	home, err := os.UserHomeDir()
	if err != nil {
		return doctorCheck{Name: "Plugin", Status: "warn", Detail: "cannot determine home directory", Required: false}
	}

	skillsDir := filepath.Join(home, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return doctorCheck{Name: "Plugin", Status: "warn", Detail: "no skills installed — run 'npx skills@latest add <package> --all -g'", Required: false}
	}

	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			count++
		}
	}

	if count == 0 {
		return doctorCheck{Name: "Plugin", Status: "warn", Detail: "no skills found — run 'npx skills@latest add <package> --all -g'", Required: false}
	}

	return doctorCheck{
		Name:     "Plugin",
		Status:   "pass",
		Detail:   fmt.Sprintf("%d skills found in ~/.claude/skills/", count),
		Required: false,
	}
}

func checkOptionalCLI(name string, reason string) doctorCheck {
	_, err := exec.LookPath(name)
	if err != nil {
		return doctorCheck{
			Name:     strings.Title(name) + " CLI", //nolint:staticcheck
			Status:   "warn",
			Detail:   fmt.Sprintf("not found (optional \u2014 %s)", reason),
			Required: false,
		}
	}

	return doctorCheck{
		Name:     strings.Title(name) + " CLI", //nolint:staticcheck
		Status:   "pass",
		Detail:   "available",
		Required: false,
	}
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}

// countCheckStatuses tallies pass, fail, and warn counts from checks.
func countCheckStatuses(checks []doctorCheck) (passes, fails, warns int) {
	for _, c := range checks {
		switch c.Status {
		case "pass":
			passes++
		case "fail":
			fails++
		case "warn":
			warns++
		}
	}
	return passes, fails, warns
}

// buildDoctorSummary constructs a human-readable summary from check tallies.
func buildDoctorSummary(passes, fails, warns, total int) string {
	switch {
	case fails == 0 && warns == 0:
		return fmt.Sprintf("%d/%d checks passed", passes, total)
	case fails == 0:
		summary := fmt.Sprintf("%d/%d checks passed, %d warning", passes, total, warns)
		if warns > 1 {
			summary += "s"
		}
		return summary
	default:
		parts := []string{fmt.Sprintf("%d/%d checks passed", passes, total)}
		if warns > 0 {
			w := fmt.Sprintf("%d warning", warns)
			if warns > 1 {
				w += "s"
			}
			parts = append(parts, w)
		}
		if fails > 0 {
			f := fmt.Sprintf("%d failed", fails)
			parts = append(parts, f)
		}
		return strings.Join(parts, ", ")
	}
}

func computeResult(checks []doctorCheck) doctorOutput {
	passes, fails, warns := countCheckStatuses(checks)
	total := len(checks)

	result := "HEALTHY"
	if fails > 0 {
		result = "UNHEALTHY"
	}

	return doctorOutput{
		Checks:  checks,
		Result:  result,
		Summary: buildDoctorSummary(passes, fails, warns, total),
	}
}
