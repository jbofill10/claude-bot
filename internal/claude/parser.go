package claude

import (
	"encoding/json"
	"fmt"
)

// Event represents a parsed stream-json event from the Claude CLI.
type Event struct {
	Type    string          // "assistant", "result", "error", "system"
	Content string          // text content
	Raw     json.RawMessage // original JSON for forwarding
}

// streamMessage is the top-level JSON envelope emitted by claude --output-format stream-json.
type streamMessage struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
	Result  string          `json:"result"`
}

// assistantMessage represents the "message" field for assistant-type events.
type assistantMessage struct {
	Content []contentBlock `json:"content"`
}

// contentBlock represents a single block inside an assistant message's content array.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ParseEvent parses a single line of stream-json output from the Claude CLI
// and returns an Event. Empty lines are skipped (returns nil, nil).
func ParseEvent(line []byte) (*Event, error) {
	// Skip empty lines.
	if len(line) == 0 {
		return nil, nil
	}

	var msg streamMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("parse stream-json line: %w", err)
	}

	ev := &Event{
		Type: msg.Type,
		Raw:  json.RawMessage(append([]byte(nil), line...)),
	}

	switch msg.Type {
	case "assistant":
		ev.Content = extractAssistantText(msg.Message)
	case "result":
		ev.Content = msg.Result
	case "error":
		// For error events, try to surface whatever text is available.
		ev.Content = string(msg.Message)
	default:
		// Unknown type -- keep raw JSON but leave Content empty.
	}

	return ev, nil
}

// extractAssistantText pulls all text blocks out of an assistant message payload.
func extractAssistantText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var am assistantMessage
	if err := json.Unmarshal(raw, &am); err != nil {
		return ""
	}
	var text string
	for _, block := range am.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text
}
