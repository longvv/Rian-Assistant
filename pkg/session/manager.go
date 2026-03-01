package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type Session struct {
	Key      string              `json:"key"`
	Messages []providers.Message `json:"messages"`
	Summary  string              `json:"summary,omitempty"`
	Created  time.Time           `json:"created"`
	Updated  time.Time           `json:"updated"`
}

type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	storage  string
	ttl      time.Duration // sessions older than this are pruned on load
}

func NewSessionManager(storage string) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		storage:  storage,
		ttl:      7 * 24 * time.Hour, // default: 7-day TTL
	}

	if storage != "" {
		os.MkdirAll(storage, 0755)
		sm.loadSessions()
	}

	return sm
}

// StartPeriodicCleanup launches a background goroutine that evicts expired
// in-memory sessions every interval. It stops when ctx is cancelled.
// This prevents unbounded memory growth in long-running processes where
// loadSessions (startup-only TTL check) is never re-run.
func (sm *SessionManager) StartPeriodicCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sm.evictExpired()
			}
		}
	}()
}

func (sm *SessionManager) evictExpired() {
	if sm.ttl <= 0 {
		return
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for key, s := range sm.sessions {
		if time.Since(s.Updated) > sm.ttl {
			delete(sm.sessions, key)
		}
	}
}

func (sm *SessionManager) GetOrCreate(key string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		return session
	}

	session = &Session{
		Key:      key,
		Messages: []providers.Message{},
		Created:  time.Now(),
		Updated:  time.Now(),
	}
	sm.sessions[key] = session

	return session
}

func (sm *SessionManager) AddMessage(sessionKey, role, content string) {
	sm.AddFullMessage(sessionKey, providers.Message{
		Role:    role,
		Content: content,
	})
}

// AddFullMessage adds a complete message with tool calls and tool call ID to the session.
// This is used to save the full conversation flow including tool calls and tool results.
func (sm *SessionManager) AddFullMessage(sessionKey string, msg providers.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionKey]
	if !ok {
		session = &Session{
			Key:      sessionKey,
			Messages: []providers.Message{},
			Created:  time.Now(),
		}
		sm.sessions[sessionKey] = session
	}

	session.Messages = append(session.Messages, msg)
	session.Updated = time.Now()
}

func (sm *SessionManager) GetHistory(key string) []providers.Message {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[key]
	if !ok {
		return []providers.Message{}
	}

	history := make([]providers.Message, len(session.Messages))
	copy(history, session.Messages)
	return history
}

func (sm *SessionManager) GetSummary(key string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[key]
	if !ok {
		return ""
	}
	return session.Summary
}

func (sm *SessionManager) SetSummary(key string, summary string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		session.Summary = summary
		session.Updated = time.Now()
	}
}

func (sm *SessionManager) TruncateHistory(key string, keepLast int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if !ok {
		return
	}

	if keepLast <= 0 {
		session.Messages = []providers.Message{}
		session.Updated = time.Now()
		return
	}

	if len(session.Messages) <= keepLast {
		return
	}

	// Copy to a new backing array so the dropped elements can be GC'd.
	// A plain reslice keeps the original array alive despite the smaller len.
	tail := session.Messages[len(session.Messages)-keepLast:]
	fresh := make([]providers.Message, keepLast)
	copy(fresh, tail)
	session.Messages = fresh
	session.Updated = time.Now()
}

// sanitizeFilename converts a session key into a cross-platform safe filename.
// Session keys use "channel:chatID" (e.g. "telegram:123456") but ':' is the
// volume separator on Windows, so filepath.Base would misinterpret the key.
// We replace it with '_'. The original key is preserved inside the JSON file,
// so loadSessions still maps back to the right in-memory key.
func sanitizeFilename(key string) string {
	return strings.ReplaceAll(key, ":", "_")
}

func (sm *SessionManager) Save(key string) error {
	if sm.storage == "" {
		return nil
	}

	filename := sanitizeFilename(key)

	// filepath.IsLocal rejects empty names, "..", absolute paths, and
	// OS-reserved device names (NUL, COM1 … on Windows).
	// The extra checks reject "." and any directory separators so that
	// the session file is always written directly inside sm.storage.
	if filename == "." || !filepath.IsLocal(filename) || strings.ContainsAny(filename, `/\`) {
		return os.ErrInvalid
	}

	// Snapshot under read lock, then perform slow file I/O after unlock.
	sm.mu.RLock()
	stored, ok := sm.sessions[key]
	if !ok {
		sm.mu.RUnlock()
		return nil
	}

	snapshot := Session{
		Key:     stored.Key,
		Summary: stored.Summary,
		Created: stored.Created,
		Updated: stored.Updated,
	}
	if len(stored.Messages) > 0 {
		snapshot.Messages = make([]providers.Message, len(stored.Messages))
		copy(snapshot.Messages, stored.Messages)
	} else {
		snapshot.Messages = []providers.Message{}
	}
	sm.mu.RUnlock()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(sm.storage, filename+".json")
	tmpFile, err := os.CreateTemp(sm.storage, "session-*.tmp")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(0644); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, sessionPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (sm *SessionManager) loadSessions() error {
	files, err := os.ReadDir(sm.storage)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		sessionPath := filepath.Join(sm.storage, file.Name())
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		// Prune sessions older than TTL to prevent unbounded disk growth
		if sm.ttl > 0 && time.Since(session.Updated) > sm.ttl {
			_ = os.Remove(sessionPath)
			continue
		}

		// Strip orphaned tool_call tails: an assistant message with tool_calls
		// that was saved without its tool results (e.g. crash mid-iteration)
		// will cause every subsequent LLM call to fail with a 400 error.
		session.Messages = sanitizeMessages(session.Messages)

		sm.sessions[session.Key] = &session
	}

	return nil
}

// sanitizeMessages removes any incomplete tool_call/tool_result pairs from the
// tail of a message list. Walking backward, if the last group is an assistant
// message with tool_calls that is NOT followed by a matching set of tool result
// messages, the whole group is dropped. This repairs sessions corrupted by
// crashes or rate-limit interruptions mid-iteration.
func sanitizeMessages(msgs []providers.Message) []providers.Message {
	original := msgs
	for {
		if len(msgs) == 0 {
			break
		}
		last := msgs[len(msgs)-1]
		// If the tail is a tool result, it's part of a completed pair — OK.
		if last.Role == "tool" {
			break
		}
		// If the tail is an assistant message WITH tool_calls, the results are
		// missing (they would appear after). Drop the assistant message.
		if last.Role == "assistant" && len(last.ToolCalls) > 0 {
			msgs = msgs[:len(msgs)-1]
			continue
		}
		// Any other tail (user message, assistant without tool_calls) is fine.
		break
	}
	// If elements were trimmed, copy to a new backing array so the dropped
	// Message values (including their string fields) can be GC'd.
	if len(msgs) < len(original) {
		compact := make([]providers.Message, len(msgs))
		copy(compact, msgs)
		return compact
	}
	return msgs
}

// SetHistory updates the messages of a session.
func (sm *SessionManager) SetHistory(key string, history []providers.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		// Create a deep copy to strictly isolate internal state
		// from the caller's slice.
		msgs := make([]providers.Message, len(history))
		copy(msgs, history)
		session.Messages = msgs
		session.Updated = time.Now()
	}
}
