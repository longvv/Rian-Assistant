package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"telegram:123456", "telegram_123456"},
		{"discord:987654321", "discord_987654321"},
		{"slack:C01234", "slack_C01234"},
		{"no-colons-here", "no-colons-here"},
		{"multiple:colons:here", "multiple_colons_here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSave_WithColonInKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create a session with a key containing colon (typical channel session key).
	key := "telegram:123456"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	// Save should succeed even though the key contains ':'
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save(%q) failed: %v", key, err)
	}

	// The file on disk should use sanitized name.
	expectedFile := filepath.Join(tmpDir, "telegram_123456.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file %s to exist", expectedFile)
	}

	// Load into a fresh manager and verify the session round-trips.
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory(key)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after reload, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected message content %q, got %q", "hello", history[0].Content)
	}
}

func TestSave_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	badKeys := []string{"", ".", "..", "foo/bar", "foo\\bar"}
	for _, key := range badKeys {
		sm.GetOrCreate(key)
		if err := sm.Save(key); err == nil {
			t.Errorf("Save(%q) should have failed but didn't", key)
		}
	}
}

func TestSanitizeMessages(t *testing.T) {
	tests := []struct {
		name     string
		input    []providers.Message
		expected int // expected length after sanitization
	}{
		{
			name:     "Empty history",
			input:    []providers.Message{},
			expected: 0,
		},
		{
			name:     "Normal history ending with user message",
			input:    []providers.Message{{Role: "user", Content: "Hello"}},
			expected: 1,
		},
		{
			name: "Normal history ending with tool result",
			input: []providers.Message{
				{Role: "user", Content: "Search for X"},
				{Role: "assistant", ToolCalls: []providers.ToolCall{{ID: "1"}}},
				{Role: "tool", Content: "Result for X", ToolCallID: "1"},
			},
			expected: 3,
		},
		{
			name: "Orphaned tool call at tail (needs stripping)",
			input: []providers.Message{
				{Role: "user", Content: "Search for X"},
				{Role: "assistant", ToolCalls: []providers.ToolCall{{ID: "1"}}},
			},
			expected: 1, // Only the user message should remain
		},
		{
			name: "Multiple orphaned tool calls at tail",
			input: []providers.Message{
				{Role: "user", Content: "Search for X"},
				{Role: "assistant", ToolCalls: []providers.ToolCall{{ID: "1"}}},
				{Role: "assistant", ToolCalls: []providers.ToolCall{{ID: "2"}}},
			},
			expected: 1, // Both incomplete assistant messages dropped
		},
		{
			name: "Assistant message at tail but NO tool calls",
			input: []providers.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			expected: 2, // Should not strip regular text replies
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeMessages(tt.input)
			if len(got) != tt.expected {
				t.Errorf("sanitizeMessages() returned %d messages, want %d", len(got), tt.expected)
			}
		})
	}
}
