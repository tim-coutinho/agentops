package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boshu2/agentops/cli/internal/types"
)

const (
	// ManifestFileName is the name of the plan manifest file.
	ManifestFileName = "manifest.jsonl"

	// PlansDir is the subdirectory under .agents for plan manifests.
	PlansDir = "plans"
)

var (
	planProjectPath string
	planBeadsID     string
	planStatus      string
	planName        string
)

var plansCmd = &cobra.Command{
	Use:   "plans",
	Short: "Manage plan manifest for robust plan discovery",
	Long: `Plans manages the plan manifest at .agents/plans/manifest.jsonl.

This command group provides robust plan discovery, fixing:
  - G2: Fragile discovery of ~/.claude/plans/ files
  - G4: Transcript parsing issues
  - G5: Hardcoded path assumptions

The manifest tracks all plans with metadata for filtering and traceability.`,
}

var plansRegisterCmd = &cobra.Command{
	Use:   "register <plan-path>",
	Short: "Register a plan in the manifest",
	Long: `Register adds a plan to the manifest.jsonl for discovery.

Called automatically when Claude exits plan mode, or manually for existing plans.

Examples:
  ao plans register ~/.claude/plans/peaceful-stirring-tome.md
  ao plans register ~/.claude/plans/my-plan.md --beads-id ol-a46.2
  ao plans register ./docs/plan.md --project /path/to/project`,
	Args: cobra.ExactArgs(1),
	RunE: runPlansRegister,
}

var plansListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered plans",
	Long: `List shows all plans in the manifest.

Use --project to filter by project path.
Use --status to filter by plan status.`,
	RunE: runPlansList,
}

var plansSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search plans by name or project",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlansSearch,
}

var plansUpdateCmd = &cobra.Command{
	Use:   "update <plan-path>",
	Short: "Update a plan's status or metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlansUpdate,
}

var plansSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync manifest with beads (beads is source of truth)",
	Long: `Sync pulls plan metadata from beads to prevent drift.

F6: Beads is the source of truth. The manifest syncs FROM beads:
  1. Find all epics with linked plans in beads
  2. Update manifest status to match beads status
  3. Add missing plans that beads references
  4. Report drift (manifest entries without beads linkage)

This ensures manifest and beads stay consistent.`,
	RunE: runPlansSync,
}

var plansDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show drift between manifest and beads",
	Long: `Diff compares the plan manifest against beads reality.

F6: Shows discrepancies without modifying anything:
  - Status mismatches (manifest says active, beads says closed)
  - Orphaned plans (in manifest but no beads ID)
  - Missing plans (beads epic references plan not in manifest)

Use 'ao plans sync' to fix the drift.`,
	RunE: runPlansDiff,
}

func init() {
	rootCmd.AddCommand(plansCmd)
	plansCmd.AddCommand(plansRegisterCmd)
	plansCmd.AddCommand(plansListCmd)
	plansCmd.AddCommand(plansSearchCmd)
	plansCmd.AddCommand(plansUpdateCmd)
	plansCmd.AddCommand(plansSyncCmd)
	plansCmd.AddCommand(plansDiffCmd)

	// Register flags
	plansRegisterCmd.Flags().StringVar(&planProjectPath, "project", "", "Project path this plan applies to")
	plansRegisterCmd.Flags().StringVar(&planBeadsID, "beads-id", "", "Beads issue/epic ID this plan implements")
	plansRegisterCmd.Flags().StringVar(&planName, "name", "", "Human-readable plan name")

	// List flags
	plansListCmd.Flags().StringVar(&planProjectPath, "project", "", "Filter by project path")
	plansListCmd.Flags().StringVar(&planStatus, "status", "", "Filter by status (active, completed, abandoned, superseded)")

	// Update flags
	plansUpdateCmd.Flags().StringVar(&planStatus, "status", "", "New status for the plan")
	plansUpdateCmd.Flags().StringVar(&planBeadsID, "beads-id", "", "Update beads ID")
}

// computePlanChecksum returns first 8 bytes of SHA256 as hex
func computePlanChecksum(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	checksum := sha256.Sum256(content)
	return hex.EncodeToString(checksum[:8]), nil
}

// createPlanEntry builds a manifest entry from path and metadata
func createPlanEntry(absPath string, modTime time.Time, projectPath, name, beadsID, checksum string) types.PlanManifestEntry {
	return types.PlanManifestEntry{
		Path:        absPath,
		CreatedAt:   modTime,
		ProjectPath: projectPath,
		PlanName:    name,
		Status:      types.PlanStatusActive,
		BeadsID:     beadsID,
		UpdatedAt:   time.Now(),
		Checksum:    checksum,
	}
}

// appendManifestEntry appends an entry to the manifest file
func appendManifestEntry(manifestPath string, entry types.PlanManifestEntry) error {
	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // write complete, close best-effort
	}()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.WriteString(string(data) + "\n")
	return err
}

// resolveProjectPath returns the explicit project path or detects it from the plan file.
func resolveProjectPath(explicit, planPath string) string {
	if explicit != "" {
		return explicit
	}
	return detectProjectPath(planPath)
}

// resolvePlanName returns the explicit name or derives one from the file path.
func resolvePlanName(explicit, planPath string) string {
	if explicit != "" {
		return explicit
	}
	return strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
}

// upsertManifestEntry updates an existing entry or appends a new one.
// Returns true if an existing entry was updated.
func upsertManifestEntry(manifestPath string, existing []types.PlanManifestEntry, entry types.PlanManifestEntry) (bool, error) {
	for i, e := range existing {
		if e.Path == entry.Path {
			existing[i] = entry
			return true, saveManifest(manifestPath, existing)
		}
	}
	return false, appendManifestEntry(manifestPath, entry)
}

// printRegistrationSummary prints details after a new plan registration.
func printRegistrationSummary(entry types.PlanManifestEntry) {
	fmt.Printf("✓ Registered plan: %s\n", entry.PlanName)
	if entry.BeadsID != "" {
		fmt.Printf("  Beads ID: %s\n", entry.BeadsID)
	}
	if entry.ProjectPath != "" {
		fmt.Printf("  Project: %s\n", entry.ProjectPath)
	}
}

// loadOrCreateManifest returns the manifest path and its current entries,
// creating the directory if needed.
func loadOrCreateManifest() (string, []types.PlanManifestEntry, error) {
	manifestPath, err := getManifestPath()
	if err != nil {
		return "", nil, fmt.Errorf("get manifest path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return "", nil, fmt.Errorf("create manifest dir: %w", err)
	}
	existing, err := loadManifest(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, fmt.Errorf("load manifest: %w", err)
	}
	return manifestPath, existing, nil
}

// buildRegisterEntry validates the plan path, computes checksum, and builds the entry.
func buildRegisterEntry(planPath, projectFlag, nameFlag, beadsID string) (types.PlanManifestEntry, error) {
	info, err := os.Stat(planPath)
	if err != nil {
		return types.PlanManifestEntry{}, fmt.Errorf("plan not found: %w", err)
	}
	checksum, err := computePlanChecksum(planPath)
	if err != nil {
		return types.PlanManifestEntry{}, fmt.Errorf("checksum: %w", err)
	}
	return createPlanEntry(
		planPath, info.ModTime(),
		resolveProjectPath(projectFlag, planPath),
		resolvePlanName(nameFlag, planPath),
		beadsID, checksum,
	), nil
}

func runPlansRegister(cmd *cobra.Command, args []string) error {
	absPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	if GetDryRun() {
		if _, statErr := os.Stat(absPath); statErr != nil {
			return fmt.Errorf("plan not found: %w", statErr)
		}
		fmt.Printf("[dry-run] Would register plan: %s\n", absPath)
		return nil
	}

	entry, err := buildRegisterEntry(absPath, planProjectPath, planName, planBeadsID)
	if err != nil {
		return err
	}

	manifestPath, existing, err := loadOrCreateManifest()
	if err != nil {
		return err
	}

	updated, err := upsertManifestEntry(manifestPath, existing, entry)
	if err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if updated {
		fmt.Printf("✓ Updated plan in manifest: %s\n", absPath)
		return nil
	}

	printRegistrationSummary(entry)
	return nil
}

// planStatusSymbols maps plan status to a display symbol; unknown statuses fall through to string form.
var planStatusSymbols = map[types.PlanStatus]string{
	types.PlanStatusActive:    "○",
	types.PlanStatusCompleted: "✓",
}

// filterPlans returns entries matching the project and status filters.
func filterPlans(entries []types.PlanManifestEntry, project, status string) []types.PlanManifestEntry {
	var out []types.PlanManifestEntry
	for _, e := range entries {
		if project != "" && !strings.Contains(e.ProjectPath, project) {
			continue
		}
		if status != "" && string(e.Status) != status {
			continue
		}
		out = append(out, e)
	}
	return out
}

// printPlanEntry prints a single plan entry with optional verbose detail.
func printPlanEntry(e types.PlanManifestEntry, verbose bool) {
	sym, ok := planStatusSymbols[e.Status]
	if !ok {
		sym = string(e.Status)
	}
	fmt.Printf("%s %s", sym, e.PlanName)
	if e.BeadsID != "" {
		fmt.Printf(" [%s]", e.BeadsID)
	}
	fmt.Println()

	if verbose {
		fmt.Printf("    Path: %s\n", e.Path)
		fmt.Printf("    Project: %s\n", e.ProjectPath)
		fmt.Printf("    Created: %s\n", e.CreatedAt.Format("2006-01-02"))
	}
}

func runPlansList(cmd *cobra.Command, args []string) error {
	manifestPath, err := getManifestPath()
	if err != nil {
		return fmt.Errorf("get manifest path: %w", err)
	}

	entries, err := loadManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No plans registered. Use 'ao plans register <path>' to add plans.")
			return nil
		}
		return fmt.Errorf("load manifest: %w", err)
	}

	filtered := filterPlans(entries, planProjectPath, planStatus)
	if len(filtered) == 0 {
		fmt.Println("No plans match the filter criteria.")
		return nil
	}

	verbose := GetVerbose()
	for _, e := range filtered {
		printPlanEntry(e, verbose)
	}

	return nil
}

func runPlansSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(args[0])

	manifestPath, err := getManifestPath()
	if err != nil {
		return fmt.Errorf("get manifest path: %w", err)
	}

	entries, err := loadManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No plans registered.")
			return nil
		}
		return fmt.Errorf("load manifest: %w", err)
	}

	var matches []types.PlanManifestEntry
	for _, e := range entries {
		searchText := strings.ToLower(e.PlanName + " " + e.ProjectPath + " " + e.BeadsID)
		if strings.Contains(searchText, query) {
			matches = append(matches, e)
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No plans matching '%s'\n", query)
		return nil
	}

	fmt.Printf("Found %d plan(s) matching '%s':\n\n", len(matches), query)
	for _, e := range matches {
		fmt.Printf("  %s\n", e.PlanName)
		fmt.Printf("    Path: %s\n", e.Path)
		if e.BeadsID != "" {
			fmt.Printf("    Beads: %s\n", e.BeadsID)
		}
	}

	return nil
}

func runPlansUpdate(cmd *cobra.Command, args []string) error {
	planPath := args[0]

	absPath, err := filepath.Abs(planPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	if GetDryRun() {
		fmt.Printf("[dry-run] Would update plan: %s\n", absPath)
		return nil
	}

	manifestPath, err := getManifestPath()
	if err != nil {
		return fmt.Errorf("get manifest path: %w", err)
	}

	entries, err := loadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	found := false
	for i, e := range entries {
		if e.Path == absPath {
			if planStatus != "" {
				entries[i].Status = types.PlanStatus(planStatus)
			}
			if planBeadsID != "" {
				entries[i].BeadsID = planBeadsID
			}
			entries[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("plan not found in manifest: %s", absPath)
	}

	if err := saveManifest(manifestPath, entries); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	fmt.Printf("✓ Updated plan: %s\n", absPath)
	return nil
}

// getManifestPath returns the path to the manifest file.
func getManifestPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Look for .agents directory
	agentsDir := filepath.Join(cwd, ".agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		// Try to find rig root
		agentsDir = findAgentsDir(cwd)
		if agentsDir == "" {
			// Default to cwd
			agentsDir = filepath.Join(cwd, ".agents")
		}
	}

	return filepath.Join(agentsDir, PlansDir, ManifestFileName), nil
}

// findAgentsDir looks for .agents directory walking up to rig root.
func findAgentsDir(startDir string) string {
	dir := startDir
	for {
		candidate := filepath.Join(dir, ".agents")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		// Check for rig markers
		markers := []string{".beads", "crew", "polecats"}
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return filepath.Join(dir, ".agents")
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// loadManifest reads all entries from the manifest file.
func loadManifest(path string) ([]types.PlanManifestEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only manifest load, close error non-fatal
	}()

	var entries []types.PlanManifestEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry types.PlanManifestEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip invalid lines
		}
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// saveManifest writes all entries to the manifest file.
func saveManifest(path string, entries []types.PlanManifestEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // write complete, close best-effort
	}()

	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			continue
		}
		if _, err := f.WriteString(string(data) + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// detectProjectPath attempts to find the project path for a plan file.
func detectProjectPath(planPath string) string {
	// If plan is in ~/.claude/plans/, extract project from plan content
	if strings.Contains(planPath, ".claude/plans/") {
		content, err := os.ReadFile(planPath)
		if err != nil {
			return ""
		}

		// Look for project path in content
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Project:") || strings.Contains(line, "Working directory:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Default to cwd
	cwd, _ := os.Getwd()
	return cwd
}

// buildBeadsIDIndex creates a map of beadsID -> slice index
func buildBeadsIDIndex(entries []types.PlanManifestEntry) map[string]int {
	index := make(map[string]int)
	for i, e := range entries {
		if e.BeadsID != "" {
			index[e.BeadsID] = i
		}
	}
	return index
}

// syncEpicStatus syncs a single epic status and returns true if changed
func syncEpicStatus(entries []types.PlanManifestEntry, idx int, beadsStatus string) bool {
	newStatus := types.PlanStatusActive
	if beadsStatus == "closed" {
		newStatus = types.PlanStatusCompleted
	}
	if entries[idx].Status != newStatus {
		entries[idx].Status = newStatus
		entries[idx].UpdatedAt = time.Now()
		return true
	}
	return false
}

// countUnlinkedEntries counts entries without beads linkage
func countUnlinkedEntries(entries []types.PlanManifestEntry) int {
	count := 0
	for _, e := range entries {
		if e.BeadsID == "" {
			count++
			VerbosePrintf("Drift: %s has no beads linkage\n", e.PlanName)
		}
	}
	return count
}

// runPlansSync syncs manifest with beads (F6: beads is source of truth).
func runPlansSync(cmd *cobra.Command, args []string) error {
	if GetDryRun() {
		fmt.Println("[dry-run] Would sync manifest with beads")
		return nil
	}

	manifestPath, err := getManifestPath()
	if err != nil {
		return fmt.Errorf("get manifest path: %w", err)
	}

	entries, err := loadManifest(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load manifest: %w", err)
	}

	byBeadsID := buildBeadsIDIndex(entries)

	beadsOutput, err := queryBeadsEpics()
	if err != nil {
		VerbosePrintf("Warning: could not query beads: %v\n", err)
		fmt.Println("Beads not available. Checking manifest for drift...")
	}

	synced := 0
	for _, epic := range beadsOutput {
		if idx, ok := byBeadsID[epic.ID]; ok {
			if syncEpicStatus(entries, idx, epic.Status) {
				synced++
				VerbosePrintf("Synced %s: -> %s\n", epic.ID, entries[idx].Status)
			}
		}
	}

	drift := countUnlinkedEntries(entries)

	if synced > 0 {
		if err := saveManifest(manifestPath, entries); err != nil {
			return fmt.Errorf("save manifest: %w", err)
		}
	}

	fmt.Printf("✓ Sync complete: %d synced, %d drift\n", synced, drift)
	if drift > 0 {
		fmt.Printf("  Hint: Run 'ao plans list' to see entries without beads linkage\n")
	}

	return nil
}

// beadsEpic represents a beads epic for sync.
type beadsEpic struct {
	ID     string
	Status string
}

// queryBeadsEpics queries beads for epic statuses.
func queryBeadsEpics() ([]beadsEpic, error) {
	// Run bd list --type epic to get epics
	cmd := exec.Command("bd", "list", "--type", "epic", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list: %w", err)
	}

	// Parse JSONL output (one line per issue)
	var epics []beadsEpic
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		var data map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			continue
		}

		id, _ := data["id"].(string)
		status, _ := data["status"].(string)
		if id != "" {
			epics = append(epics, beadsEpic{ID: id, Status: status})
		}
	}

	return epics, nil
}

// driftEntry represents a single drift detection
type driftEntry struct {
	Type     string
	PlanName string
	BeadsID  string
	Manifest string
	Beads    string
}

// buildBeadsStatusIndex creates a map of epic ID -> status from beads
func buildBeadsStatusIndex(epics []beadsEpic) map[string]string {
	index := make(map[string]string)
	for _, e := range epics {
		index[e.ID] = e.Status
	}
	return index
}

// detectStatusDrifts finds status mismatches between manifest and beads
func detectStatusDrifts(byBeadsID map[string]*types.PlanManifestEntry, beadsIndex map[string]string) []driftEntry {
	var drifts []driftEntry
	for beadsID, entry := range byBeadsID {
		beadsStatus, exists := beadsIndex[beadsID]
		if !exists {
			drifts = append(drifts, driftEntry{
				Type: "missing_beads", PlanName: entry.PlanName,
				BeadsID: beadsID, Manifest: string(entry.Status), Beads: "(not found)",
			})
			continue
		}
		manifestClosed := entry.Status == types.PlanStatusCompleted
		beadsClosed := beadsStatus == "closed"
		if manifestClosed != beadsClosed {
			drifts = append(drifts, driftEntry{
				Type: "status_mismatch", PlanName: entry.PlanName,
				BeadsID: beadsID, Manifest: string(entry.Status), Beads: beadsStatus,
			})
		}
	}
	return drifts
}

// detectOrphanedEntries finds manifest entries without beads linkage
func detectOrphanedEntries(entries []types.PlanManifestEntry) []driftEntry {
	var drifts []driftEntry
	for _, e := range entries {
		if e.BeadsID == "" {
			drifts = append(drifts, driftEntry{
				Type: "orphaned", PlanName: e.PlanName,
				BeadsID: "(none)", Manifest: string(e.Status), Beads: "n/a",
			})
		}
	}
	return drifts
}

// printDrifts outputs drift entries in a formatted way
func printDrifts(drifts []driftEntry) {
	fmt.Printf("Found %d drift(s):\n\n", len(drifts))
	for _, d := range drifts {
		switch d.Type {
		case "status_mismatch":
			fmt.Printf("  ⚠ Status mismatch: %s [%s]\n", d.PlanName, d.BeadsID)
			fmt.Printf("    Manifest: %s, Beads: %s\n", d.Manifest, d.Beads)
		case "orphaned":
			fmt.Printf("  ○ Orphaned plan: %s\n", d.PlanName)
			fmt.Printf("    No beads ID linked\n")
		case "missing_beads":
			fmt.Printf("  ✗ Missing in beads: %s [%s]\n", d.PlanName, d.BeadsID)
			fmt.Printf("    Epic not found in beads\n")
		}
	}
}

// runPlansDiff shows drift between manifest and beads (F6).
func runPlansDiff(cmd *cobra.Command, args []string) error {
	manifestPath, err := getManifestPath()
	if err != nil {
		return fmt.Errorf("get manifest path: %w", err)
	}

	entries, err := loadManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No manifest found. Run 'ao plans register' to create one.")
			return nil
		}
		return fmt.Errorf("load manifest: %w", err)
	}

	// Build manifest index by beads ID
	byBeadsID := make(map[string]*types.PlanManifestEntry)
	for i := range entries {
		if entries[i].BeadsID != "" {
			byBeadsID[entries[i].BeadsID] = &entries[i]
		}
	}

	beadsOutput, err := queryBeadsEpics()
	if err != nil {
		return fmt.Errorf("query beads: %w", err)
	}

	beadsIndex := buildBeadsStatusIndex(beadsOutput)

	// Collect all drifts
	drifts := detectStatusDrifts(byBeadsID, beadsIndex)
	drifts = append(drifts, detectOrphanedEntries(entries)...)

	if len(drifts) == 0 {
		fmt.Println("✓ No drift detected. Manifest and beads are in sync.")
		return nil
	}

	printDrifts(drifts)

	fmt.Printf("\nRun 'ao plans sync' to fix status mismatches.\n")
	fmt.Printf("Run 'ao plans update <path> --beads-id <id>' to link orphaned plans.\n")

	return nil
}
