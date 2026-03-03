package claude

import (
	"testing"
)

func TestParseEventAssistant(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello world"}]}}`)
	ev, err := ParseEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != "assistant" {
		t.Errorf("got type %q, want %q", ev.Type, "assistant")
	}
	if ev.Content != "Hello world" {
		t.Errorf("got content %q, want %q", ev.Content, "Hello world")
	}
}

func TestParseEventResult(t *testing.T) {
	line := []byte(`{"type":"result","result":"Final output"}`)
	ev, err := ParseEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != "result" {
		t.Errorf("got type %q, want %q", ev.Type, "result")
	}
	if ev.Content != "Final output" {
		t.Errorf("got content %q, want %q", ev.Content, "Final output")
	}
}

func TestParseEventContentBlockDelta(t *testing.T) {
	line := []byte(`{"type":"content_block_delta","message":{"delta":{"type":"text_delta","text":"streaming token"}}}`)
	ev, err := ParseEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != "content_block_delta" {
		t.Errorf("got type %q, want %q", ev.Type, "content_block_delta")
	}
	if ev.Content != "streaming token" {
		t.Errorf("got content %q, want %q", ev.Content, "streaming token")
	}
}

func TestParseEventContentBlockDeltaEmptyMessage(t *testing.T) {
	line := []byte(`{"type":"content_block_delta"}`)
	ev, err := ParseEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Content != "" {
		t.Errorf("expected empty content for delta with no message, got %q", ev.Content)
	}
}

func TestParseEventUnknownType(t *testing.T) {
	line := []byte(`{"type":"ping"}`)
	ev, err := ParseEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != "ping" {
		t.Errorf("got type %q, want %q", ev.Type, "ping")
	}
	if ev.Content != "" {
		t.Errorf("expected empty content for unknown type, got %q", ev.Content)
	}
}

func TestParseEventEmptyLine(t *testing.T) {
	ev, err := ParseEvent([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev != nil {
		t.Errorf("expected nil event for empty line, got %+v", ev)
	}
}

func TestParseEventInvalidJSON(t *testing.T) {
	_, err := ParseEvent([]byte(`{not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseEventError(t *testing.T) {
	line := []byte(`{"type":"error","message":"something went wrong"}`)
	ev, err := ParseEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != "error" {
		t.Errorf("got type %q, want %q", ev.Type, "error")
	}
	if ev.Content != `"something went wrong"` {
		t.Errorf("got content %q, want %q", ev.Content, `"something went wrong"`)
	}
}

func TestExtractDeltaText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid delta",
			input: `{"delta":{"type":"text_delta","text":"hello"}}`,
			want:  "hello",
		},
		{
			name:  "empty delta text",
			input: `{"delta":{"type":"text_delta","text":""}}`,
			want:  "",
		},
		{
			name:  "missing delta field",
			input: `{"other":"value"}`,
			want:  "",
		},
		{
			name:  "invalid json",
			input: `{bad`,
			want:  "",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDeltaText([]byte(tt.input))
			if got != tt.want {
				t.Errorf("extractDeltaText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
