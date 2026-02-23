package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:   "quick-start",
	Short: "Set up AgentOps in your project (5 minutes)",
	Long: `Initialize AgentOps in your current project.

This command:
  1. Creates .agents/ directory structure
  2. Optionally initializes beads (git-native issues)
  3. Creates starter knowledge pack
  4. Shows next steps

Examples:
  ao quick-start              # Full setup with beads
  ao quick-start --no-beads   # Skip beads initialization
  ao quick-start --minimal    # Just .agents/ structure`,
	RunE: runQuickstart,
}

var (
	noBeads bool
	minimal bool
)

func init() {
	quickstartCmd.GroupID = "start"
	rootCmd.AddCommand(quickstartCmd)
	quickstartCmd.Flags().BoolVar(&noBeads, "no-beads", false, "Skip beads initialization")
	quickstartCmd.Flags().BoolVar(&minimal, "minimal", false, "Minimal setup (just directories)")
}

// quickstartBeadsStep handles step 3: beads initialization or skip.
func quickstartBeadsStep(cwd string) {
	if !noBeads {
		fmt.Println("\n━━━ STEP 3: Beads initialization ━━━")
		if err := initBeads(cwd); err != nil {
			fmt.Printf("  ⚠ Beads init skipped: %v\n", err)
			fmt.Println("  → You can run 'bd init' later to enable git-native issues")
		}
	} else {
		fmt.Println("\n━━━ STEP 3: Skipping beads (--no-beads) ━━━")
		fmt.Println("  → Issues will be tracked in .agents/tasks.json instead")
		createTasksFile(cwd)
	}
}

// quickstartClaudeMdStep handles step 4: create CLAUDE.md if missing.
func quickstartClaudeMdStep(cwd string) {
	fmt.Println("\n━━━ STEP 4: Project configuration ━━━")
	claudeMdPath := filepath.Join(cwd, "CLAUDE.md")
	if _, err := os.Stat(claudeMdPath); os.IsNotExist(err) {
		if err := createProjectClaudeMd(cwd); err != nil {
			fmt.Printf("  ⚠ Warning: %v\n", err)
		} else {
			fmt.Println("  ✓ Created CLAUDE.md (project instructions)")
		}
	} else {
		fmt.Println("  ✓ CLAUDE.md already exists")
	}
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Println(`
╔══════════════════════════════════════════════════════════════════╗
║                 AGENTOPS QUICK START                               ║
║           Setting up your project for knowledge compounding       ║
╚══════════════════════════════════════════════════════════════════╝`)
	fmt.Printf("Project: %s\n\n", cwd)

	fmt.Println("━━━ STEP 1: Creating .agents/ structure ━━━")
	dirs := []string{
		".agents/research",
		".agents/synthesis",
		".agents/specs",
		".agents/learnings",
		".agents/patterns",
		".agents/retros",
		".agents/handoff",
	}

	for _, dir := range dirs {
		path := filepath.Join(cwd, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
		fmt.Printf("  ✓ %s/\n", dir)
	}

	if minimal {
		fmt.Println("\n✓ Minimal setup complete!")
		showNextSteps(false)
		return nil
	}

	fmt.Println("\n━━━ STEP 2: Creating starter knowledge pack ━━━")
	if err := createStarterPack(cwd); err != nil {
		fmt.Printf("  ⚠ Warning: %v\n", err)
	}

	quickstartBeadsStep(cwd)
	quickstartClaudeMdStep(cwd)

	fmt.Println("\n━━━ SETUP COMPLETE ━━━")
	showNextSteps(!noBeads)

	return nil
}

func createStarterPack(cwd string) error {
	// Create a few starter patterns that are universally useful
	patterns := map[string]string{
		".agents/patterns/context-boundaries.md": `# Pattern: Fresh Context Per Phase

**Tier:** 2 (Pattern)
**Source:** AgentOps multi-epic post-mortem

## Problem

Long sessions accumulate errors. Context pollution causes drift.

## Solution

Fresh Claude session for each RPI phase:
- /research → new session
- /plan → new session
- /implement → new session
- /post-mortem → new session

## The 40% Rule

| Context % | Success Rate |
|-----------|--------------|
| <40%      | 98%          |
| 40-60%    | ~50%         |
| >60%      | ~1%          |

At 35% context, checkpoint and consider new session.
`,
		".agents/patterns/pre-mortem-first.md": `# Pattern: Pre-Mortem Before Implementation

**Tier:** 2 (Pattern)
**Source:** Knowledge Flywheel post-mortem (2026-01-22)

## Problem

Implementation failures are expensive. Debugging takes longer than preventing.

## Solution

Run /pre-mortem on P0/P1 work BEFORE /crank:

` + "```bash" + `
/pre-mortem .agents/specs/my-feature.md
# Review findings
# Then implement
/crank
` + "```" + `

## Evidence

Pre-mortem caught 6 critical issues before implementation:
- API group mismatches
- Path resolution errors
- Migration assumptions
- Schema drift

## When to Skip

- Bug fixes (already understood)
- Single-file changes (<50 lines)
- P2/P3 priority work
`,
		".agents/learnings/session-hygiene.md": `# Learning: Session Hygiene

**Date:** Starter Pack
**Tier:** 1 (Learning)

## Key Practices

1. **Always push before saying done**
   - Work that isn't pushed didn't happen
   - ` + "`git push`" + ` is the final step

2. **Run /post-mortem after epics**
   - Captures learnings for the flywheel
   - Creates patterns from experience

3. **Check Smart Connections before starting**
   - Search for prior art: ` + "`mcp__smart-connections-work__lookup`" + `
   - Don't reinvent what exists

4. **Use beads for state**
   - ` + "`bd ready`" + ` shows unblocked work
   - ` + "`bd sync`" + ` commits issue changes
`,
	}

	for path, content := range patterns {
		fullPath := filepath.Join(cwd, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
		fmt.Printf("  ✓ %s\n", path)
	}

	return nil
}

func initBeads(cwd string) error {
	// Check if beads is available
	if _, err := exec.LookPath("bd"); err != nil {
		return fmt.Errorf("bd command not found (install: brew install beads)")
	}

	// Check if already initialized
	beadsDir := filepath.Join(cwd, ".beads")
	if _, err := os.Stat(beadsDir); err == nil {
		fmt.Println("  ✓ Beads already initialized")
		return nil
	}

	// Determine prefix from directory name
	dirName := filepath.Base(cwd)
	prefix := strings.ToLower(dirName)
	if len(prefix) > 4 {
		prefix = prefix[:4]
	}

	fmt.Printf("  Initializing beads with prefix '%s'...\n", prefix)

	// Ask for confirmation
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("  Use prefix '%s'? [Y/n]: ", prefix)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "n" || response == "no" {
		fmt.Print("  Enter prefix: ")
		prefix, _ = reader.ReadString('\n')
		prefix = strings.TrimSpace(prefix)
	}

	// Run bd init
	cmd := exec.Command("bd", "init", "--prefix", prefix)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd init failed: %s", string(output))
	}

	fmt.Printf("  ✓ Beads initialized with prefix '%s'\n", prefix)
	return nil
}

func createTasksFile(cwd string) {
	tasksPath := filepath.Join(cwd, ".agents/tasks.json")
	content := `{
  "tasks": [],
  "note": "Beads-optional mode. Use 'bd init' to enable full git-native issues."
}
`
	//nolint:errcheck // quickstart setup, errors shown implicitly by missing output
	os.WriteFile(tasksPath, []byte(content), 0644)
	fmt.Println("  ✓ Created .agents/tasks.json (beads-optional mode)")
}

func createProjectClaudeMd(cwd string) error {
	dirName := filepath.Base(cwd)
	content := fmt.Sprintf(`# %s

## Quick Start

`+"```bash"+`
bd ready              # See unblocked issues
/implement <issue>    # Work on an issue
/crank                # Autonomous epic execution
`+"```"+`

## Session Protocol

`+"```bash"+`
# Start
gt hook               # Check for hooked work
bd ready              # Find available work

# End
bd sync               # Sync beads
git add .
git commit -m "..."
git push              # NEVER stop before pushing
`+"```"+`

## JIT Loading

| Working On | Load |
|------------|------|
| Research | .agents/research/ |
| Implementation | Check existing patterns first |
| Debugging | .agents/learnings/ |

## Beads Prefix

Issue prefix: (set during ol quick-start)
`, dirName)

	return os.WriteFile(filepath.Join(cwd, "CLAUDE.md"), []byte(content), 0644)
}

func showNextSteps(hasBeads bool) {
	fmt.Print(`
═══════════════════════════════════════════════════════════════════
                          NEXT STEPS
═══════════════════════════════════════════════════════════════════
`)

	if hasBeads {
		fmt.Println(`  1. Create your first issue:
     $ bd create "My first task"

  2. Start working:
     $ claude
     > /implement <issue-id>

  3. When done, extract learnings:
     > /post-mortem`)
	} else {
		fmt.Println(`  1. Start Claude in your project:
     $ claude

  2. Work normally - learnings go to .agents/

  3. When ready for full power:
     $ bd init
     $ bd create "My first task"`)
	}

	fmt.Print(`
  4. Your knowledge compounds automatically via Smart Connections

═══════════════════════════════════════════════════════════════════

  "Others forget. AgentOps learns. Zero infrastructure."

═══════════════════════════════════════════════════════════════════
`)
}
