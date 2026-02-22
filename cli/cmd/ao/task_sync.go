package main

import (
	"bufio"
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

// TaskEvent represents a captured task from Claude Code's Task tool.
type TaskEvent struct {
	// TaskID is the unique identifier from Claude Code's TaskCreate.
	TaskID string `json:"task_id"`

	// Subject is the task title.
	Subject string `json:"subject"`

	// Description is the detailed task description.
	Description string `json:"description,omitempty"`

	// Status is the task state: pending, in_progress, completed.
	Status string `json:"status"`

	// SessionID links to the Claude Code session.
	SessionID string `json:"session_id"`

	// CreatedAt is when the task was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the task was last modified.
	UpdatedAt time.Time `json:"updated_at"`

	// CompletedAt is when the task was marked completed (if applicable).
	CompletedAt time.Time `json:"completed_at,omitempty"`

	// LearningID links to any learning generated from this task.
	LearningID string `json:"learning_id,omitempty"`

	// Maturity is the CASS maturity level derived from task status.
	Maturity types.Maturity `json:"maturity,omitempty"`

	// Utility is the learned value from feedback signals.
	Utility float64 `json:"utility,omitempty"`

	// Owner is the agent/human assigned to the task.
	Owner string `json:"owner,omitempty"`

	// BlockedBy lists task IDs this task depends on.
	BlockedBy []string `json:"blocked_by,omitempty"`

	// Metadata contains additional task metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TaskFilePath is the relative path to the task events log.
const TaskFilePath = ".agents/ao/tasks.jsonl"

var taskSyncCmd = &cobra.Command{
	Use:   "task-sync",
	Short: "Sync tasks from Claude Code sessions to CASS",
	Long: `Import and sync tasks from Claude Code's Task tool to CASS maturity tracking.

This command bridges Claude Code's built-in TaskCreate/TaskUpdate/TaskList tools
with the CASS (Contextual Agent Session Search) system, enabling:

  - Task → Learning promotion: Completed tasks can become learnings
  - Status → Maturity mapping: pending→provisional, in_progress→candidate, completed→established
  - Feedback loop closure: Task completion signals update learning utilities

The sync process:
  1. Reads Claude Code transcript for TaskCreate/TaskUpdate tool calls
  2. Extracts task events and stores them in .agents/ao/tasks.jsonl
  3. Maps task status to CASS maturity levels
  4. Links task completion to the feedback loop

Examples:
  ao task-sync                                # Sync from most recent transcript
  ao task-sync --transcript ~/.claude/projects/*/abc.jsonl
  ao task-sync --session session-20260125    # Filter by session
  ao task-sync --promote                     # Promote completed tasks to learnings`,
	RunE: runTaskSync,
}

var (
	taskSyncTranscript string
	taskSyncSessionID  string
	taskSyncPromote    bool
)

func init() {
	rootCmd.AddCommand(taskSyncCmd)
	taskSyncCmd.Flags().StringVar(&taskSyncTranscript, "transcript", "", "Path to Claude Code transcript")
	taskSyncCmd.Flags().StringVar(&taskSyncSessionID, "session", "", "Filter tasks by session ID")
	taskSyncCmd.Flags().BoolVar(&taskSyncPromote, "promote", false, "Promote completed tasks to learnings")
}

func runTaskSync(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Find transcript
	transcriptPath := taskSyncTranscript
	if transcriptPath == "" {
		homeDir, _ := os.UserHomeDir()
		transcriptsDir := filepath.Join(homeDir, ".claude", "projects")
		transcriptPath = findMostRecentTranscript(transcriptsDir)
	}

	if transcriptPath == "" {
		return fmt.Errorf("no transcript found; use --transcript to specify")
	}

	if GetDryRun() {
		fmt.Printf("[dry-run] Would sync tasks from: %s\n", transcriptPath)
		return nil
	}

	// Extract task events from transcript
	tasks, err := extractTaskEvents(transcriptPath, taskSyncSessionID)
	if err != nil {
		return fmt.Errorf("extract tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No task events found in transcript.")
		return nil
	}

	// Assign maturity based on status
	for i := range tasks {
		tasks[i].Maturity = statusToMaturity(tasks[i].Status)
		if tasks[i].Utility == 0 {
			tasks[i].Utility = types.InitialUtility
		}
	}

	// Write task events
	if err := writeTaskEvents(cwd, tasks); err != nil {
		return fmt.Errorf("write tasks: %w", err)
	}

	// Optionally promote completed tasks to learnings
	promoted := 0
	if taskSyncPromote {
		for _, t := range tasks {
			if t.Status == "completed" && t.LearningID == "" {
				if err := promoteTaskToLearning(cwd, &t); err != nil {
					VerbosePrintf("Warning: failed to promote task %s: %v\n", t.TaskID, err)
					continue
				}
				promoted++
			}
		}
	}

	// Output summary
	if GetOutput() == "json" {
		result := map[string]interface{}{
			"transcript": transcriptPath,
			"tasks":      tasks,
			"promoted":   promoted,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("Task Sync Complete\n")
	fmt.Printf("==================\n")
	fmt.Printf("Transcript:  %s\n", transcriptPath)
	fmt.Printf("Tasks found: %d\n", len(tasks))

	// Show status breakdown
	statusCounts := make(map[string]int)
	for _, t := range tasks {
		statusCounts[t.Status]++
	}
	fmt.Printf("\nStatus breakdown:\n")
	for status, count := range statusCounts {
		maturity := statusToMaturity(status)
		fmt.Printf("  %-12s → %-12s: %d\n", status, maturity, count)
	}

	if promoted > 0 {
		fmt.Printf("\nPromoted to learnings: %d\n", promoted)
	}

	return nil
}

// extractTaskEvents parses a Claude Code transcript for Task tool calls.
func extractTaskEvents(transcriptPath, filterSession string) ([]TaskEvent, error) {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only transcript extraction, close error non-fatal
	}()

	var tasks []TaskEvent
	taskMap := make(map[string]*TaskEvent) // Track by task ID for updates
	var currentSessionID string

	scanner := bufio.NewScanner(f)
	// Use larger buffer for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		// Extract session ID if present
		if sid, ok := data["sessionId"].(string); ok && sid != "" {
			currentSessionID = sid
		}

		// Filter by session if requested
		if filterSession != "" && currentSessionID != filterSession {
			continue
		}

		// Look for tool calls
		message, ok := data["message"].(map[string]interface{})
		if !ok {
			continue
		}

		content, ok := message["content"].([]interface{})
		if !ok {
			continue
		}

		for _, item := range content {
			block, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			blockType, _ := block["type"].(string)
			if blockType != "tool_use" {
				continue
			}

			toolName, _ := block["name"].(string)
			input, _ := block["input"].(map[string]interface{})

			switch toolName {
			case "TaskCreate":
				task := parseTaskCreate(input, currentSessionID)
				if task != nil {
					taskMap[task.TaskID] = task
				}

			case "TaskUpdate":
				taskID, _ := input["taskId"].(string)
				if existing, ok := taskMap[taskID]; ok {
					updateTask(existing, input)
				}

			case "TaskList":
				// TaskList returns current state; we could use this to validate
				// but for now we rely on Create/Update events
			}
		}
	}

	// Convert map to slice
	for _, t := range taskMap {
		tasks = append(tasks, *t)
	}

	return tasks, nil
}

// parseTaskCreate extracts a TaskEvent from TaskCreate input.
func parseTaskCreate(input map[string]interface{}, sessionID string) *TaskEvent {
	subject, _ := input["subject"].(string)
	if subject == "" {
		return nil
	}

	task := &TaskEvent{
		TaskID:    generateTaskID(),
		Subject:   subject,
		Status:    "pending",
		SessionID: sessionID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Utility:   types.InitialUtility,
	}

	if desc, ok := input["description"].(string); ok {
		task.Description = desc
	}

	if activeForm, ok := input["activeForm"].(string); ok {
		if task.Metadata == nil {
			task.Metadata = make(map[string]interface{})
		}
		task.Metadata["active_form"] = activeForm
	}

	return task
}

// updateTask applies updates from TaskUpdate to an existing task.
func updateTask(task *TaskEvent, input map[string]interface{}) {
	task.UpdatedAt = time.Now()

	if status, ok := input["status"].(string); ok {
		task.Status = status
		task.Maturity = statusToMaturity(status)
		if status == "completed" {
			task.CompletedAt = time.Now()
		}
	}

	if subject, ok := input["subject"].(string); ok {
		task.Subject = subject
	}

	if desc, ok := input["description"].(string); ok {
		task.Description = desc
	}

	if owner, ok := input["owner"].(string); ok {
		task.Owner = owner
	}
}

// statusToMaturity maps Task status to CASS maturity.
func statusToMaturity(status string) types.Maturity {
	switch status {
	case "completed":
		return types.MaturityEstablished
	case "in_progress":
		return types.MaturityCandidate
	default: // "pending"
		return types.MaturityProvisional
	}
}

// generateTaskID creates a unique task identifier.
func generateTaskID() string {
	return fmt.Sprintf("task-%s", time.Now().Format("20060102-150405"))
}

// writeTaskEvents appends task events to the task log.
func writeTaskEvents(baseDir string, tasks []TaskEvent) error {
	if len(tasks) == 0 {
		return nil
	}

	taskPath := filepath.Join(baseDir, TaskFilePath)

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(taskPath), 0755); err != nil {
		return fmt.Errorf("create task directory: %w", err)
	}

	// Load existing tasks to merge
	existing, _ := loadTaskEvents(baseDir)
	existingMap := make(map[string]bool)
	for _, t := range existing {
		existingMap[t.TaskID] = true
	}

	// Open for append
	f, err := os.OpenFile(taskPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open task file: %w", err)
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // write complete, close best-effort
	}()

	// Write only new tasks
	written := 0
	for _, task := range tasks {
		if existingMap[task.TaskID] {
			continue
		}
		data, err := json.Marshal(task)
		if err != nil {
			continue
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("write task event: %w", err)
		}
		written++
	}

	VerbosePrintf("Wrote %d new task events\n", written)
	return nil
}

// loadTaskEvents reads all task events from the log.
func loadTaskEvents(baseDir string) ([]TaskEvent, error) {
	taskPath := filepath.Join(baseDir, TaskFilePath)

	f, err := os.Open(taskPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only task load, close error non-fatal
	}()

	var tasks []TaskEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var task TaskEvent
		if err := json.Unmarshal(scanner.Bytes(), &task); err != nil {
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// promoteTaskToLearning creates a learning from a completed task.
func promoteTaskToLearning(baseDir string, task *TaskEvent) error {
	learningsDir := filepath.Join(baseDir, ".agents", "learnings")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		return fmt.Errorf("create learnings directory: %w", err)
	}

	// Create learning candidate
	learningID := fmt.Sprintf("L-%s", strings.TrimPrefix(task.TaskID, "task-"))
	learningPath := filepath.Join(learningsDir, learningID+".jsonl")

	learning := map[string]interface{}{
		"id":           learningID,
		"type":         "learning",
		"content":      fmt.Sprintf("Task completed: %s", task.Subject),
		"context":      task.Description,
		"maturity":     string(types.MaturityEstablished),
		"utility":      task.Utility,
		"confidence":   0.7, // Initial confidence for promoted tasks
		"extracted_at": time.Now().Format(time.RFC3339),
		"source": map[string]interface{}{
			"session_id": task.SessionID,
			"task_id":    task.TaskID,
			"type":       "task_promotion",
		},
	}

	data, err := json.Marshal(learning)
	if err != nil {
		return fmt.Errorf("marshal learning: %w", err)
	}

	if err := os.WriteFile(learningPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write learning: %w", err)
	}

	// Update task with learning link
	task.LearningID = learningID

	VerbosePrintf("Promoted task %s to learning %s\n", task.TaskID, learningID)
	return nil
}

// taskFeedbackCmd applies feedback from task outcomes to learnings.
var taskFeedbackCmd = &cobra.Command{
	Use:   "task-feedback",
	Short: "Apply task completion signals to CASS feedback loop",
	Long: `Process task completion events and apply feedback to associated learnings.

This integrates with the CASS feedback loop by:
  - Finding tasks completed in the session
  - Computing reward from task completion signals
  - Updating utility of learnings associated with completed tasks
  - Triggering maturity transitions based on task outcomes

Examples:
  ao task-feedback --session session-20260125
  ao task-feedback --all                      # Process all pending tasks`,
	RunE: runTaskFeedback,
}

var (
	taskFeedbackSessionID string
	taskFeedbackAll       bool
)

func init() {
	rootCmd.AddCommand(taskFeedbackCmd)
	taskFeedbackCmd.Flags().StringVar(&taskFeedbackSessionID, "session", "", "Session ID to process")
	taskFeedbackCmd.Flags().BoolVar(&taskFeedbackAll, "all", false, "Process all tasks without feedback")
}

func runTaskFeedback(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Load tasks
	tasks, err := loadTaskEvents(cwd)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No tasks found. Run 'ao task-sync' first.")
			return nil
		}
		return fmt.Errorf("load tasks: %w", err)
	}

	// Filter tasks
	var processable []TaskEvent
	for _, t := range tasks {
		if taskFeedbackSessionID != "" && t.SessionID != taskFeedbackSessionID {
			continue
		}
		if t.Status == "completed" && t.LearningID != "" {
			processable = append(processable, t)
		}
	}

	if len(processable) == 0 {
		fmt.Println("No completed tasks with learnings found.")
		return nil
	}

	if GetDryRun() {
		fmt.Printf("[dry-run] Would process feedback for %d tasks\n", len(processable))
		return nil
	}

	// Process feedback for each task
	processed := 0
	for _, task := range processable {
		// Compute reward: completed tasks get positive reward
		reward := 0.8 // Completed task = positive signal

		// Find and update the learning
		learningPath, err := findLearningFile(cwd, task.LearningID)
		if err != nil {
			VerbosePrintf("Warning: learning not found for task %s: %v\n", task.TaskID, err)
			continue
		}

		oldUtility, newUtility, err := updateLearningUtility(learningPath, reward, types.DefaultAlpha)
		if err != nil {
			VerbosePrintf("Warning: failed to update %s: %v\n", learningPath, err)
			continue
		}

		// Check for maturity transition
		result, err := ratchet.CheckMaturityTransition(learningPath)
		if err == nil && result.Transitioned {
			_, _ = ratchet.ApplyMaturityTransition(learningPath)
			VerbosePrintf("Maturity transition: %s → %s\n", result.OldMaturity, result.NewMaturity)
		}

		fmt.Printf("  ✓ %s: %.3f → %.3f (task: %s)\n",
			task.LearningID, oldUtility, newUtility, task.Subject)
		processed++
	}

	fmt.Printf("\nProcessed feedback for %d tasks\n", processed)
	return nil
}

// taskStatusCmd shows the status of tasks and their CASS maturity.
var taskStatusCmd = &cobra.Command{
	Use:   "task-status",
	Short: "Show task status and CASS maturity distribution",
	Long: `Display the status of tracked tasks and their CASS maturity levels.

Shows:
  - Task count by status (pending, in_progress, completed)
  - CASS maturity distribution
  - Tasks pending feedback
  - Tasks promoted to learnings

Examples:
  ao task-status
  ao task-status --session session-20260125
  ao task-status --format json`,
	RunE: runTaskStatus,
}

var taskStatusSessionID string

func init() {
	rootCmd.AddCommand(taskStatusCmd)
	taskStatusCmd.Flags().StringVar(&taskStatusSessionID, "session", "", "Filter by session ID")
}

func runTaskStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	tasks, err := loadTaskEvents(cwd)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No tasks found. Run 'ao task-sync' first.")
			return nil
		}
		return fmt.Errorf("load tasks: %w", err)
	}

	// Filter by session if requested
	if taskStatusSessionID != "" {
		var filtered []TaskEvent
		for _, t := range tasks {
			if t.SessionID == taskStatusSessionID {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	// Compute distributions
	statusCounts := make(map[string]int)
	maturityCounts := make(map[types.Maturity]int)
	withLearnings := 0

	for _, t := range tasks {
		statusCounts[t.Status]++
		maturityCounts[t.Maturity]++
		if t.LearningID != "" {
			withLearnings++
		}
	}

	if GetOutput() == "json" {
		result := map[string]interface{}{
			"total":          len(tasks),
			"status_counts":  statusCounts,
			"maturity_counts": maturityCounts,
			"with_learnings": withLearnings,
			"tasks":          tasks,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("Task Status\n")
	fmt.Printf("===========\n")
	fmt.Printf("Total tasks: %d\n\n", len(tasks))

	fmt.Printf("By Status:\n")
	for status, count := range statusCounts {
		fmt.Printf("  %-12s: %d\n", status, count)
	}

	fmt.Printf("\nBy CASS Maturity:\n")
	for _, m := range []types.Maturity{types.MaturityProvisional, types.MaturityCandidate, types.MaturityEstablished, types.MaturityAntiPattern} {
		if count, ok := maturityCounts[m]; ok {
			fmt.Printf("  %-12s: %d\n", m, count)
		}
	}

	fmt.Printf("\nPromoted to learnings: %d\n", withLearnings)

	return nil
}
