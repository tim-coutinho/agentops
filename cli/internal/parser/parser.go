// Package parser provides streaming JSONL parsing for Claude Code transcripts.
package parser

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

// DefaultMaxContentLength is the default truncation limit for content fields.
const DefaultMaxContentLength = 500

// Message type constants for transcript entries.
const (
	msgTypeUser       = "user"
	msgTypeAssistant  = "assistant"
	msgTypeToolUse    = "tool_use"
	msgTypeToolResult = "tool_result"
)

// Error classification constants for parse errors.
const (
	errClassJSON     = "json"
	errClassSchema   = "schema"
	errClassEncoding = "encoding"
)

// Parser handles streaming JSONL parsing with configurable options.
type Parser struct {
	// MaxContentLength is the maximum characters before truncation.
	MaxContentLength int

	// SkipMalformed skips malformed lines instead of erroring.
	SkipMalformed bool

	// OnProgress is called with progress updates for large files.
	OnProgress func(linesProcessed, totalLines int)
}

// NewParser creates a parser with default settings.
func NewParser() *Parser {
	return &Parser{
		MaxContentLength: DefaultMaxContentLength,
		SkipMalformed:    true,
	}
}

// rawMessage represents the raw JSON structure from Claude Code transcripts.
type rawMessage struct {
	Type       string `json:"type"`
	SessionID  string `json:"sessionId"`
	Timestamp  string `json:"timestamp"`
	UUID       string `json:"uuid"`
	ParentUUID string `json:"parentUuid,omitempty"`
	Message    *struct {
		Role    string `json:"role"`
		Content any    `json:"content"` // Can be string or array
	} `json:"message,omitempty"`
	// ToolUseResult contains structured tool output (e.g., for TodoWrite)
	ToolUseResult any `json:"toolUseResult,omitempty"`
}

// ParseResult contains the result of parsing a JSONL stream.
type ParseResult struct {
	Messages       []types.TranscriptMessage
	TotalLines     int
	MalformedLines int
	Errors         []error

	// Checksum is SHA256 hash of the parsed content (first 16 hex chars).
	// Used for detecting changes and validating re-parsing.
	Checksum string

	// FilePath is the source file path (if parsed from file).
	FilePath string

	// ParsedAt is when parsing completed.
	ParsedAt time.Time
}

// ParseError provides structured error information for transcript parsing failures.
type ParseError struct {
	Line       int    `json:"line"`
	Column     int    `json:"column,omitempty"`
	Message    string `json:"message"`
	RawContent string `json:"raw_content,omitempty"`
	ErrorType  string `json:"error_type"` // "json", "schema", "encoding"
}

func (e *ParseError) Error() string {
	if e.Column > 0 {
		return fmt.Sprintf("line %d, col %d: %s (%s)", e.Line, e.Column, e.Message, e.ErrorType)
	}
	return fmt.Sprintf("line %d: %s (%s)", e.Line, e.Message, e.ErrorType)
}

// Parse reads JSONL from the reader and returns parsed messages.
func (p *Parser) Parse(r io.Reader) (*ParseResult, error) {
	result := &ParseResult{
		Messages: make([]types.TranscriptMessage, 0),
	}

	hasher := sha256.New()
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line size

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		result.TotalLines = lineNum

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		_, _ = hasher.Write(line)
		_, _ = hasher.Write([]byte("\n"))

		p.processLine(line, lineNum, result)

		if p.OnProgress != nil && lineNum%100 == 0 {
			p.OnProgress(lineNum, 0)
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scanner error: %w", err)
	}

	hash := hasher.Sum(nil)
	result.Checksum = hex.EncodeToString(hash[:8])
	result.ParsedAt = time.Now()

	return result, nil
}

// processLine parses a single JSONL line and appends the result or error.
func (p *Parser) processLine(line []byte, lineNum int, result *ParseResult) {
	msg, err := p.parseLine(line, lineNum)
	if err != nil {
		result.MalformedLines++
		if !p.SkipMalformed {
			result.Errors = append(result.Errors, &ParseError{
				Line:       lineNum,
				Message:    err.Error(),
				ErrorType:  classifyError(err),
				RawContent: truncateForError(string(line), 100),
			})
		}
		return
	}
	if msg != nil {
		result.Messages = append(result.Messages, *msg)
	}
}

// classifyError determines the error type for structured reporting.
func classifyError(err error) string {
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "invalid character"):
		return errClassJSON
	case strings.Contains(errStr, "unexpected end"):
		return errClassJSON
	case strings.Contains(errStr, "cannot unmarshal"):
		return errClassSchema
	case strings.Contains(errStr, "invalid UTF-8"):
		return errClassEncoding
	default:
		return errClassJSON
	}
}

// truncateForError limits error context to a reasonable size.
func truncateForError(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ParseFile parses a JSONL file by path.
func (p *Parser) ParseFile(path string) (result *ParseResult, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	result, err = p.Parse(f)
	if result != nil {
		result.FilePath = path
	}
	return result, err
}

// timestampFormats lists the formats to try when parsing timestamps.
var timestampFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05.000Z",
}

// parseTimestamp parses a timestamp string, trying multiple formats.
// Returns zero time if all formats fail.
func parseTimestamp(s string) time.Time {
	for _, format := range timestampFormats {
		if ts, err := time.Parse(format, s); err == nil {
			return ts
		}
	}
	return time.Time{}
}

// isValidMessageType checks if the type is one we should parse.
func isValidMessageType(msgType string) bool {
	switch msgType {
	case msgTypeUser, msgTypeAssistant, msgTypeToolUse, msgTypeToolResult:
		return true
	default:
		return false
	}
}

// parseContentBlocks extracts text and tool calls from content block array.
func (p *Parser) parseContentBlocks(blocks []any) (string, []types.ToolCall) {
	var content string
	var tools []types.ToolCall

	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		text, tool := p.classifyBlock(blockMap)
		content += text
		if tool != nil {
			tools = append(tools, *tool)
		}
	}

	return content, tools
}

// classifyBlock dispatches a single content block into text or tool call.
func (p *Parser) classifyBlock(blockMap map[string]any) (string, *types.ToolCall) {
	blockType, _ := blockMap["type"].(string)
	switch blockType {
	case "text":
		if text, ok := blockMap["text"].(string); ok {
			return p.truncate(text), nil
		}
	case msgTypeToolUse:
		return "", p.parseToolUse(blockMap)
	case msgTypeToolResult:
		return "", p.parseToolResult(blockMap)
	}
	return "", nil
}

// parseLine parses a single JSON line.
func (p *Parser) parseLine(line []byte, lineNum int) (*types.TranscriptMessage, error) {
	var raw rawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Skip non-message types
	if !isValidMessageType(raw.Type) {
		return nil, nil
	}

	msg := &types.TranscriptMessage{
		Type:         raw.Type,
		Timestamp:    parseTimestamp(raw.Timestamp),
		SessionID:    raw.SessionID,
		MessageIndex: lineNum,
	}

	// Extract content from message
	if raw.Message != nil {
		msg.Role = raw.Message.Role
		p.extractMessageContent(raw.Message.Content, msg)
	}

	return msg, nil
}

// extractMessageContent populates msg.Content and msg.Tools from raw content.
func (p *Parser) extractMessageContent(rawContent any, msg *types.TranscriptMessage) {
	switch content := rawContent.(type) {
	case string:
		msg.Content = p.truncate(content)
	case []any:
		msg.Content, msg.Tools = p.parseContentBlocks(content)
	}
}

// parseToolUse extracts tool call information from a tool_use block.
func (p *Parser) parseToolUse(block map[string]any) *types.ToolCall {
	name, _ := block["name"].(string)
	if name == "" {
		return nil
	}

	toolCall := &types.ToolCall{
		Name: name,
	}

	// Extract input parameters
	if input, ok := block["input"].(map[string]any); ok {
		toolCall.Input = input
	}

	return toolCall
}

// parseToolResult extracts tool result information from a tool_result block.
func (p *Parser) parseToolResult(block map[string]any) *types.ToolCall {
	toolCall := &types.ToolCall{
		Name: msgTypeToolResult,
	}

	// Check if it's an error result
	if isError, ok := block["is_error"].(bool); ok && isError {
		toolCall.Error = "tool error"
	}

	toolCall.Output = p.extractToolResultContent(block["content"])
	return toolCall
}

// extractToolResultContent extracts text from a tool_result content field,
// which may be a plain string or an array of text blocks.
func (p *Parser) extractToolResultContent(content any) string {
	switch c := content.(type) {
	case string:
		return p.truncate(c)
	case []any:
		var out string
		for _, item := range c {
			if itemMap, ok := item.(map[string]any); ok {
				if text, ok := itemMap["text"].(string); ok {
					out += p.truncate(text)
				}
			}
		}
		return out
	default:
		return ""
	}
}

// truncate limits content to MaxContentLength characters.
func (p *Parser) truncate(s string) string {
	if p.MaxContentLength <= 0 || len(s) <= p.MaxContentLength {
		return s
	}
	return s[:p.MaxContentLength] + "... [truncated]"
}

// ParseChannel returns a channel that emits messages as they're parsed.
// Useful for streaming large files without loading all into memory.
func (p *Parser) ParseChannel(r io.Reader) (<-chan types.TranscriptMessage, <-chan error) {
	msgCh := make(chan types.TranscriptMessage, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)
		p.channelScanner(r, msgCh, errCh)
	}()

	return msgCh, errCh
}

// processChannelLine parses one scanner line and forwards the result to msgCh/errCh.
// Returns false if scanning should stop (fatal parse error).
func (p *Parser) processChannelLine(line []byte, lineNum int, msgCh chan<- types.TranscriptMessage, errCh chan<- error) bool {
	if len(line) == 0 {
		return true
	}
	msg, err := p.parseLine(line, lineNum)
	if err != nil {
		if !p.SkipMalformed {
			errCh <- fmt.Errorf("line %d: %w", lineNum, err)
			return false
		}
		return true
	}
	if msg != nil {
		msgCh <- *msg
	}
	return true
}

// channelScanner scans r line by line, sending parsed messages to msgCh and errors to errCh.
func (p *Parser) channelScanner(r io.Reader, msgCh chan<- types.TranscriptMessage, errCh chan<- error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if !p.processChannelLine(scanner.Bytes(), lineNum, msgCh, errCh) {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		errCh <- fmt.Errorf("scanner error: %w", err)
	}
}
