package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// setupInboxDir creates a temp directory with a .agents/mail/ subdirectory and
// writes the given content to messages.jsonl. Returns the temp directory root.
func setupInboxDir(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	mailDir := filepath.Join(tmpDir, ".agents", "mail")
	if err := os.MkdirAll(mailDir, 0700); err != nil {
		t.Fatal(err)
	}
	messagesPath := filepath.Join(mailDir, "messages.jsonl")
	if err := os.WriteFile(messagesPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestLoadMessages(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantMessages   int
		wantCorrupted  int
	}{
		{
			name: "corrupted lines are skipped",
			content: `{"id":"msg-1","from":"agent-1","to":"mayor","body":"test 1","timestamp":"2024-01-01T00:00:00Z","read":false,"type":"progress"}
not valid json
{"id":"msg-2","from":"agent-2","to":"mayor","body":"test 2","timestamp":"2024-01-02T00:00:00Z","read":false,"type":"completion"}
{malformed
{"id":"msg-3","from":"witness","to":"mayor","body":"test 3","timestamp":"2024-01-03T00:00:00Z","read":false,"type":"blocker"}
`,
			wantMessages:  3,
			wantCorrupted: 2,
		},
		{
			name: "empty lines are not counted as corrupted",
			content: `{"id":"msg-1","from":"agent-1","to":"mayor","body":"test 1","timestamp":"2024-01-01T00:00:00Z","read":false,"type":"progress"}

{"id":"msg-2","from":"agent-2","to":"mayor","body":"test 2","timestamp":"2024-01-02T00:00:00Z","read":false,"type":"completion"}

`,
			wantMessages:  2,
			wantCorrupted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupInboxDir(t, tt.content)

			messages, corruptedCount, err := loadMessages(tmpDir)
			if err != nil {
				t.Errorf("loadMessages() error = %v", err)
			}
			if len(messages) != tt.wantMessages {
				t.Errorf("got %d messages, want %d", len(messages), tt.wantMessages)
			}
			if corruptedCount != tt.wantCorrupted {
				t.Errorf("got %d corrupted, want %d", corruptedCount, tt.wantCorrupted)
			}
		})
	}
}

func TestLoadMessagesFileNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, err := loadMessages(tmpDir)
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist error, got %v", err)
	}
}

func TestFilterMessagesInvalidDuration(t *testing.T) {
	// Use timestamps that ensure messages are within the time window
	now := time.Now()
	messages := []Message{
		{ID: "1", From: "agent-1", To: "mayor", Timestamp: now},
		{ID: "2", From: "agent-2", To: "mayor", Timestamp: now.Add(-30 * time.Second)},
	}

	tests := []struct {
		name         string
		since        string
		wantWarning  bool
		wantMsgCount int
	}{
		{"valid duration 5m", "5m", false, 2},
		{"valid duration 1h", "1h", false, 2},
		{"invalid duration 5x", "5x", true, 2},     // Invalid, should warn and show all
		{"invalid duration abc", "abc", true, 2},    // Invalid, should warn and show all
		{"valid negative duration -5m", "-5m", false, 0}, // Valid Go duration, results in future time filter
		{"empty duration", "", false, 2},            // No filter
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, warning := filterMessages(messages, tt.since, "", false)

			if tt.wantWarning && warning == "" {
				t.Errorf("expected warning for %q, got none", tt.since)
			}
			if !tt.wantWarning && warning != "" {
				t.Errorf("unexpected warning for %q: %s", tt.since, warning)
			}
			if len(filtered) != tt.wantMsgCount {
				t.Errorf("got %d messages, want %d", len(filtered), tt.wantMsgCount)
			}
		})
	}
}

func TestFilterMessagesByTime(t *testing.T) {
	now := time.Now()
	messages := []Message{
		{ID: "1", From: "agent-1", To: "mayor", Timestamp: now},
		{ID: "2", From: "agent-2", To: "mayor", Timestamp: now.Add(-30 * time.Minute)},
		{ID: "3", From: "agent-3", To: "mayor", Timestamp: now.Add(-2 * time.Hour)},
	}

	// Filter to last hour - should get 2 messages
	filtered, warning := filterMessages(messages, "1h", "", false)
	if warning != "" {
		t.Errorf("unexpected warning: %s", warning)
	}
	if len(filtered) != 2 {
		t.Errorf("got %d messages, want 2", len(filtered))
	}
}

func TestFilterMessagesBySender(t *testing.T) {
	messages := []Message{
		{ID: "1", From: "agent-1", To: "mayor", Timestamp: time.Now()},
		{ID: "2", From: "witness", To: "mayor", Timestamp: time.Now()},
		{ID: "3", From: "agent-1", To: "mayor", Timestamp: time.Now()},
	}

	filtered, _ := filterMessages(messages, "", "witness", false)
	if len(filtered) != 1 {
		t.Errorf("got %d messages, want 1", len(filtered))
	}
	if filtered[0].ID != "2" {
		t.Errorf("got message ID %s, want 2", filtered[0].ID)
	}
}

func TestFilterMessagesUnreadOnly(t *testing.T) {
	messages := []Message{
		{ID: "1", From: "agent-1", To: "mayor", Timestamp: time.Now(), Read: false},
		{ID: "2", From: "agent-2", To: "mayor", Timestamp: time.Now(), Read: true},
		{ID: "3", From: "agent-3", To: "mayor", Timestamp: time.Now(), Read: false},
	}

	filtered, _ := filterMessages(messages, "", "", true)
	if len(filtered) != 2 {
		t.Errorf("got %d messages, want 2 unread", len(filtered))
	}
}

func TestAppendMessageConcurrent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mail directory
	mailDir := filepath.Join(tmpDir, ".agents", "mail")
	if err := os.MkdirAll(mailDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Concurrently append messages
	var wg sync.WaitGroup
	numWriters := 10
	messagesPerWriter := 5

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < messagesPerWriter; j++ {
				msg := &Message{
					ID:        generateMessageID(),
					From:      "agent",
					To:        "mayor",
					Body:      "test message",
					Timestamp: time.Now(),
					Type:      "progress",
				}
				if err := appendMessage(tmpDir, msg); err != nil {
					t.Errorf("writer %d: appendMessage error: %v", writerID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages were written correctly
	messages, corruptedCount, err := loadMessages(tmpDir)
	if err != nil {
		t.Fatalf("loadMessages() error = %v", err)
	}

	expectedCount := numWriters * messagesPerWriter
	if len(messages) != expectedCount {
		t.Errorf("got %d messages, want %d", len(messages), expectedCount)
	}

	if corruptedCount > 0 {
		t.Errorf("got %d corrupted messages, want 0 (file locking should prevent corruption)", corruptedCount)
	}

	// Verify each message is valid JSON
	for i, msg := range messages {
		if msg.ID == "" {
			t.Errorf("message %d has empty ID", i)
		}
		if msg.From == "" {
			t.Errorf("message %d has empty From", i)
		}
	}
}

func TestDefaultInboxLimit(t *testing.T) {
	if DefaultInboxLimit != 100 {
		t.Errorf("DefaultInboxLimit = %d, want 100", DefaultInboxLimit)
	}
}

func TestMarkMessagesReadConcurrent(t *testing.T) {
	tmpDir := t.TempDir()

	mailDir := filepath.Join(tmpDir, ".agents", "mail")
	if err := os.MkdirAll(mailDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create initial messages
	messagesPath := filepath.Join(mailDir, "messages.jsonl")
	var initialMessages []Message
	for i := 0; i < 10; i++ {
		msg := Message{
			ID:        generateMessageID(),
			From:      "agent",
			To:        "mayor",
			Body:      "test",
			Timestamp: time.Now(),
			Read:      false,
			Type:      "progress",
		}
		initialMessages = append(initialMessages, msg)
		time.Sleep(time.Millisecond) // Ensure unique IDs
	}

	// Write initial messages
	file, err := os.Create(messagesPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range initialMessages {
		data, _ := json.Marshal(msg)
		_, _ = file.WriteString(string(data) + "\n") //nolint:errcheck // test setup
	}
	_ = file.Close() //nolint:errcheck // test setup

	// Concurrently mark messages as read and append new ones
	var wg sync.WaitGroup

	// Writers adding new messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			msg := &Message{
				ID:        generateMessageID(),
				From:      "agent",
				To:        "mayor",
				Body:      "new message",
				Timestamp: time.Now(),
				Type:      "progress",
			}
			_ = appendMessage(tmpDir, msg) //nolint:errcheck // test concurrent writer
			time.Sleep(time.Millisecond)
		}
	}()

	// Reader marking messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Mark first 5 messages as read
		_ = markMessagesRead(tmpDir, initialMessages[:5]) //nolint:errcheck // test concurrent reader
	}()

	wg.Wait()

	// Verify results
	messages, corruptedCount, err := loadMessages(tmpDir)
	if err != nil {
		t.Fatalf("loadMessages() error = %v", err)
	}

	if corruptedCount > 0 {
		t.Errorf("got %d corrupted messages after concurrent operations", corruptedCount)
	}

	// Should have at least the initial messages (some new ones might be there too)
	if len(messages) < 10 {
		t.Errorf("got %d messages, want at least 10", len(messages))
	}
}

func TestFilterMessagesRecipientFilter(t *testing.T) {
	messages := []Message{
		{ID: "1", From: "agent-1", To: "mayor", Timestamp: time.Now()},
		{ID: "2", From: "agent-2", To: "all", Timestamp: time.Now()},
		{ID: "3", From: "agent-3", To: "agent-1", Timestamp: time.Now()}, // Not to mayor/all
		{ID: "4", From: "agent-4", To: "", Timestamp: time.Now()},        // Empty recipient
	}

	filtered, _ := filterMessages(messages, "", "", false)

	// Should get messages to "mayor", "all", or empty
	if len(filtered) != 3 {
		t.Errorf("got %d messages, want 3 (mayor, all, empty)", len(filtered))
	}

	// Verify message 3 (to agent-1) is filtered out
	for _, msg := range filtered {
		if msg.ID == "3" {
			t.Error("message to agent-1 should have been filtered out")
		}
	}
}
