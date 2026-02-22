package main

import (
	"bufio"
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

const (
	// DefaultInboxLimit is the default number of messages to display.
	DefaultInboxLimit = 100
)

// Message represents an inter-agent message.
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
	Type      string    `json:"type"` // progress, completion, blocker, farm_complete
}

var (
	inboxSince    string
	inboxFrom     string
	inboxUnread   bool
	inboxMarkRead bool
	inboxLimit    int
	mailTo        string
	mailBody      string
	mailType      string
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Check messages from agents",
	Long: `View messages from the Agent Farm.

Messages include:
  - Progress summaries from witness
  - Completion notifications from agents
  - Blocker escalations
  - Farm complete signal

Examples:
  ao inbox
  ao inbox --since 5m
  ao inbox --from witness
  ao inbox --unread
  ao inbox --limit 50`,
	RunE: runInbox,
}

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Send and receive agent messages",
	Long: `Inter-agent messaging for the Agent Farm.

Commands:
  send    Send a message
  inbox   View received messages (alias for ao inbox)

Examples:
  ao mail send --to mayor --body "Issue complete"
  ao mail send --to mayor --body "FARM COMPLETE" --type farm_complete`,
}

var mailSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message",
	Long: `Send a message to another agent or the mayor.

Examples:
  ao mail send --to mayor --body "Completed issue gt-123"
  ao mail send --to witness --body "Agent 1 stuck"
  ao mail send --to mayor --body "FARM COMPLETE" --type farm_complete`,
	RunE: runMailSend,
}

func init() {
	rootCmd.AddCommand(inboxCmd)
	rootCmd.AddCommand(mailCmd)

	mailCmd.AddCommand(mailSendCmd)

	// Inbox flags
	inboxCmd.Flags().StringVar(&inboxSince, "since", "", "Show messages from last duration (e.g., 5m, 1h)")
	inboxCmd.Flags().StringVar(&inboxFrom, "from", "", "Filter by sender")
	inboxCmd.Flags().BoolVar(&inboxUnread, "unread", false, "Show only unread messages")
	inboxCmd.Flags().BoolVar(&inboxMarkRead, "mark-read", false, "Mark displayed messages as read")
	inboxCmd.Flags().IntVar(&inboxLimit, "limit", DefaultInboxLimit, "Maximum messages to display (0 for all)")

	// Mail send flags
	mailSendCmd.Flags().StringVar(&mailTo, "to", "", "Recipient (mayor, witness, agent-N)")
	mailSendCmd.Flags().StringVar(&mailBody, "body", "", "Message body")
	mailSendCmd.Flags().StringVar(&mailType, "type", "progress", "Message type (progress, completion, blocker, farm_complete)")

	_ = mailSendCmd.MarkFlagRequired("to")
	_ = mailSendCmd.MarkFlagRequired("body")
}

// renderInboxJSON outputs inbox messages in JSON format.
func renderInboxJSON(limited []Message, totalMatching, corruptedCount int) error {
	output := struct {
		Messages  []Message `json:"messages"`
		Total     int       `json:"total"`
		Showing   int       `json:"showing"`
		Corrupted int       `json:"corrupted,omitempty"`
	}{
		Messages:  limited,
		Total:     totalMatching,
		Showing:   len(limited),
		Corrupted: corruptedCount,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// renderInboxTable outputs inbox messages in a human-readable table.
func renderInboxTable(limited []Message, totalMatching int) {
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	//nolint:errcheck // CLI tabwriter output to stdout, errors unlikely and non-recoverable
	fmt.Fprintln(w, "TIME\tFROM\tTYPE\tMESSAGE")
	//nolint:errcheck // CLI tabwriter output to stdout
	fmt.Fprintln(w, "----\t----\t----\t-------")

	for _, msg := range limited {
		age := formatAge(msg.Timestamp)
		body := truncateMessage(msg.Body, 60)
		unreadMark := ""
		if !msg.Read {
			unreadMark = "*"
		}
		//nolint:errcheck // CLI tabwriter output to stdout
		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\n", unreadMark, age, msg.From, msg.Type, body)
	}

	_ = w.Flush()

	if len(limited) < totalMatching {
		fmt.Printf("\nShowing %d of %d message(s) (use --limit 0 for all)\n", len(limited), totalMatching)
	} else {
		fmt.Printf("\n%d message(s)\n", totalMatching)
	}
}

// applyLimit returns at most limit items from the slice. A limit of 0 or less
// means no limit.
func applyLimit(msgs []Message, limit int) []Message {
	if limit > 0 && len(msgs) > limit {
		return msgs[:limit]
	}
	return msgs
}

func runInbox(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Load messages (returns messages and corruption count)
	messages, corruptedCount, err := loadMessages(cwd)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No messages")
			return nil
		}
		return fmt.Errorf("load messages: %w", err)
	}

	if corruptedCount > 0 {
		fmt.Fprintf(os.Stderr, "Warning: %d corrupted message(s) skipped\n", corruptedCount)
	}

	// Filter messages (with duration validation)
	filtered, durationWarning := filterMessages(messages, inboxSince, inboxFrom, inboxUnread)
	if durationWarning != "" {
		fmt.Fprintf(os.Stderr, "Warning: %s, using no time filter\n", durationWarning)
	}

	totalMatching := len(filtered)
	if totalMatching == 0 {
		fmt.Println("No messages")
		return nil
	}

	// Sort by timestamp descending (newest first)
	slices.SortFunc(filtered, func(a, b Message) int {
		return b.Timestamp.Compare(a.Timestamp)
	})

	limited := applyLimit(filtered, inboxLimit)

	// Output based on format
	if GetOutput() == "json" {
		return renderInboxJSON(limited, totalMatching, corruptedCount)
	}

	renderInboxTable(limited, totalMatching)

	// Mark as read if requested
	if inboxMarkRead {
		if err := markMessagesRead(cwd, limited); err != nil {
			VerbosePrintf("Warning: failed to mark messages as read: %v\n", err)
		}
	}

	return nil
}

func runMailSend(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Determine sender identity
	from := cmp.Or(os.Getenv("AO_AGENT_NAME"), "unknown")

	// Create message
	msg := Message{
		ID:        generateMessageID(),
		From:      from,
		To:        mailTo,
		Body:      mailBody,
		Timestamp: time.Now(),
		Read:      false,
		Type:      mailType,
	}

	if GetDryRun() {
		fmt.Printf("[dry-run] Would send message:\n")
		fmt.Printf("  From: %s\n", msg.From)
		fmt.Printf("  To: %s\n", msg.To)
		fmt.Printf("  Type: %s\n", msg.Type)
		fmt.Printf("  Body: %s\n", msg.Body)
		return nil
	}

	// Append to messages file
	if err := appendMessage(cwd, &msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	fmt.Printf("Message sent to %s\n", mailTo)
	VerbosePrintf("ID: %s\n", msg.ID)

	return nil
}

// Helper functions

func loadMessages(cwd string) (messages []Message, corruptedCount int, err error) {
	messagesPath := filepath.Join(cwd, ".agents", "mail", "messages.jsonl")
	file, err := os.Open(messagesPath)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Acquire shared lock for reading
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); err != nil {
		return nil, 0, fmt.Errorf("lock messages file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) //nolint:errcheck // unlock best-effort
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		// Skip empty lines
		if len(line) == 0 {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			corruptedCount++
			continue
		}
		messages = append(messages, msg)
	}

	return messages, corruptedCount, scanner.Err()
}

// parseSinceDuration parses a duration string and returns the cutoff time.
// Returns zero time and a warning if the duration is invalid, or zero time
// and empty warning if since is empty (no filter).
func parseSinceDuration(since string) (time.Time, string) {
	if since == "" {
		return time.Time{}, ""
	}
	duration, err := time.ParseDuration(since)
	if err != nil {
		return time.Time{}, fmt.Sprintf("invalid duration %q", since)
	}
	return time.Now().Add(-duration), ""
}

// messageMatchesFilters returns true if the message passes all filter criteria.
func messageMatchesFilters(msg Message, sinceTime time.Time, from string, unreadOnly bool) bool {
	if !sinceTime.IsZero() && msg.Timestamp.Before(sinceTime) {
		return false
	}
	if from != "" && msg.From != from {
		return false
	}
	if unreadOnly && msg.Read {
		return false
	}
	// Default: show messages to "mayor", "all", or empty
	if msg.To != "mayor" && msg.To != "all" && msg.To != "" {
		return false
	}
	return true
}

func filterMessages(messages []Message, since, from string, unreadOnly bool) ([]Message, string) {
	sinceTime, durationWarning := parseSinceDuration(since)

	filtered := make([]Message, 0, len(messages))
	for _, msg := range messages {
		if messageMatchesFilters(msg, sinceTime, from, unreadOnly) {
			filtered = append(filtered, msg)
		}
	}

	return filtered, durationWarning
}

func appendMessage(cwd string, msg *Message) (err error) {
	mailDir := filepath.Join(cwd, ".agents", "mail")
	if err := os.MkdirAll(mailDir, 0700); err != nil {
		return err
	}

	messagesPath := filepath.Join(mailDir, "messages.jsonl")

	// Open file for append with exclusive lock
	file, err := os.OpenFile(messagesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Acquire exclusive lock to prevent concurrent write corruption
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock messages file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) //nolint:errcheck // unlock best-effort
	}()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if _, err := file.WriteString(string(data) + "\n"); err != nil {
		return err
	}

	return nil
}

// scanMessagesFromReader reads JSONL messages from a scanner, skipping empty
// and corrupted lines. Returns the parsed messages.
func scanMessagesFromReader(scanner *bufio.Scanner) []Message {
	var msgs []Message
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue // Skip corrupted
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

// buildIDSet creates a set of message IDs from a slice for O(1) lookups.
func buildIDSet(messages []Message) map[string]bool {
	ids := make(map[string]bool, len(messages))
	for _, msg := range messages {
		ids[msg.ID] = true
	}
	return ids
}

// writeMessagesJSONL writes messages to a file in JSONL format.
// The file must already be positioned at the desired write offset.
func writeMessagesJSONL(file *os.File, messages []Message) error {
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if _, werr := file.WriteString(string(data) + "\n"); werr != nil {
			return werr
		}
	}
	return nil
}

func markMessagesRead(cwd string, messages []Message) (err error) {
	messagesPath := filepath.Join(cwd, ".agents", "mail", "messages.jsonl")

	// Open file with exclusive lock for read-modify-write
	file, err := os.OpenFile(messagesPath, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Acquire exclusive lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock messages file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) //nolint:errcheck // unlock best-effort
	}()

	// Read all messages while holding lock
	scanner := bufio.NewScanner(file)
	allMessages := scanMessagesFromReader(scanner)
	if err := scanner.Err(); err != nil {
		return err
	}

	// Mark matching messages as read
	toMark := buildIDSet(messages)
	for i := range allMessages {
		if toMark[allMessages[i].ID] {
			allMessages[i].Read = true
		}
	}

	// Truncate and rewrite file while still holding lock
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	return writeMessagesJSONL(file, allMessages)
}

func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

func formatAge(t time.Time) string {
	age := time.Since(t)

	if age < time.Minute {
		return fmt.Sprintf("%ds ago", int(age.Seconds()))
	}
	if age < time.Hour {
		return fmt.Sprintf("%dm ago", int(age.Minutes()))
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(age.Hours()))
	}
	return t.Format("Jan 2")
}

func truncateMessage(s string, max int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
