// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MemoryStore manages persistent memory for the agent.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
type MemoryStore struct {
	workspace string
	memoryDir string
}

// NewMemoryStore creates a new MemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0755)

	return &MemoryStore{
		workspace: workspace,
		memoryDir: memoryDir,
	}
}

func (ms *MemoryStore) getMemoryFile(chatID string) string {
	if chatID == "" || chatID == "default" {
		return filepath.Join(ms.memoryDir, "MEMORY.md")
	}
	safeChatID := strings.ReplaceAll(chatID, "/", "_")
	dir := filepath.Join(ms.memoryDir, "chat_"+safeChatID)
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "MEMORY.md")
}

// getTodayFile returns the path to today's daily note file (memory/YYYYMM/YYYYMMDD.md).
func (ms *MemoryStore) getTodayFile(chatID string) string {
	today := time.Now().Format("20060102") // YYYYMMDD
	monthDir := today[:6]                  // YYYYMM
	if chatID == "" || chatID == "default" {
		return filepath.Join(ms.memoryDir, monthDir, today+".md")
	}
	safeChatID := strings.ReplaceAll(chatID, "/", "_")
	return filepath.Join(ms.memoryDir, "chat_"+safeChatID, monthDir, today+".md")
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadLongTerm(chatID string) string {
	if data, err := os.ReadFile(ms.getMemoryFile(chatID)); err == nil {
		return string(data)
	}
	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *MemoryStore) WriteLongTerm(chatID, content string) error {
	return os.WriteFile(ms.getMemoryFile(chatID), []byte(content), 0644)
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadToday(chatID string) string {
	todayFile := ms.getTodayFile(chatID)
	if data, err := os.ReadFile(todayFile); err == nil {
		return string(data)
	}
	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
func (ms *MemoryStore) AppendToday(chatID, content string) error {
	todayFile := ms.getTodayFile(chatID)

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	os.MkdirAll(monthDir, 0755)

	var existingContent string
	if data, err := os.ReadFile(todayFile); err == nil {
		existingContent = string(data)
	}

	var newContent string
	if existingContent == "" {
		// Add header for new day
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		// Append to existing content
		newContent = existingContent + "\n" + content
	}

	return os.WriteFile(todayFile, []byte(newContent), 0644)
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(chatID string, days int) string {
	var notes []string

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM

		var filePath string
		if chatID == "" || chatID == "default" {
			filePath = filepath.Join(ms.memoryDir, monthDir, dateStr+".md")
		} else {
			safeChatID := strings.ReplaceAll(chatID, "/", "_")
			filePath = filepath.Join(ms.memoryDir, "chat_"+safeChatID, monthDir, dateStr+".md")
		}

		if data, err := os.ReadFile(filePath); err == nil {
			notes = append(notes, string(data))
		}
	}

	if len(notes) == 0 {
		return ""
	}

	// Join with separator
	var result string
	for i, note := range notes {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += note
	}
	return result
}

// GetMemoryContext returns formatted memory context for the agent prompt.
// Includes long-term memory and recent daily notes.
func (ms *MemoryStore) GetMemoryContext(chatID string) string {
	var parts []string

	// Long-term memory
	longTerm := ms.ReadLongTerm(chatID)
	if longTerm != "" {
		parts = append(parts, "## Long-term Memory\n\n"+longTerm)
	}

	// Recent daily notes (last 3 days)
	recentNotes := ms.GetRecentDailyNotes(chatID, 3)
	if recentNotes != "" {
		parts = append(parts, "## Recent Daily Notes\n\n"+recentNotes)
	}

	if len(parts) == 0 {
		return ""
	}

	// Join parts with separator
	var result string
	for i, part := range parts {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += part
	}
	return fmt.Sprintf("# Memory\n\n%s", result)
}
