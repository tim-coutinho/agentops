package main

import "encoding/json"

// Event type constants for Claude Code streaming JSON output.
const (
	EventTypeSystem    = "system"
	EventTypeAssistant = "assistant"
	EventTypeUser      = "user"
	EventTypeResult    = "result"
	EventTypeInit      = "init"
)

// StreamEvent is the top-level envelope for every JSON line emitted by
// Claude Code's streaming output (--output-format stream-json).
// The Type field determines which payload fields are populated.
type StreamEvent struct {
	// Type is one of the EventType* constants.
	Type string `json:"type"`

	// Subtype provides further classification within a type
	// (e.g. "tool_use", "tool_result").
	Subtype string `json:"subtype,omitempty"`

	// SessionID is the unique session identifier (present in init events).
	SessionID string `json:"session_id,omitempty"`

	// Tools lists available tool names (present in init events).
	Tools []string `json:"tools,omitempty"`

	// Model is the model identifier (present in init events).
	Model string `json:"model,omitempty"`

	// Message holds the text content for system, assistant, user, and
	// result events.
	Message string `json:"message,omitempty"`

	// ToolName is the tool being invoked (assistant tool_use subtype).
	ToolName string `json:"tool_name,omitempty"`

	// ToolInput holds the raw JSON input for a tool call.
	ToolInput json.RawMessage `json:"tool_input,omitempty"`

	// ToolUseID links a tool_result back to its tool_use request.
	ToolUseID string `json:"tool_use_id,omitempty"`

	// CostUSD is the cumulative cost reported in result events.
	CostUSD float64 `json:"cost_usd,omitempty"`

	// DurationMS is the total duration reported in result events.
	DurationMS float64 `json:"duration_ms,omitempty"`

	// DurationAPIMS is the API-side duration reported in result events.
	DurationAPIMS float64 `json:"duration_api_ms,omitempty"`

	// IsError indicates whether a result event represents an error.
	IsError bool `json:"is_error,omitempty"`

	// NumTurns is the number of conversation turns in a result event.
	NumTurns int `json:"num_turns,omitempty"`
}

// ParseStreamEvent unmarshals a single JSON line into a StreamEvent.
// Unknown fields are silently ignored (permissive parsing).
func ParseStreamEvent(data []byte) (StreamEvent, error) {
	var ev StreamEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return StreamEvent{}, err
	}
	return ev, nil
}
