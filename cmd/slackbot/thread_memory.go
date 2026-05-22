package slackbot

import (
	"sync"
	"time"
)

// threadMemory keeps a bounded conversation history per Slack thread so the
// bot can resolve follow-ups like "and how about feature xyz?".
//
// Entries expire after ttl of inactivity. The data is intentionally
// process-local: this is dev tooling, and restarts losing context is fine.
type threadMemory struct {
	mu      sync.Mutex
	ttl     time.Duration
	threads map[string]*threadEntry
}

type threadEntry struct {
	messages []ChatMessage
	expires  time.Time
}

func newThreadMemory(ttl time.Duration) *threadMemory {
	return &threadMemory{ttl: ttl, threads: make(map[string]*threadEntry)}
}

// Get returns the saved history for a thread, plus a closure to append new
// messages back to the store. The returned slice is a copy; callers may mutate.
func (m *threadMemory) Get(threadTS string) []ChatMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked()
	e, ok := m.threads[threadTS]
	if !ok {
		return nil
	}
	out := make([]ChatMessage, len(e.messages))
	copy(out, e.messages)
	return out
}

// Append stores the updated message slice and refreshes the TTL.
func (m *threadMemory) Append(threadTS string, msgs []ChatMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threads[threadTS] = &threadEntry{
		messages: append([]ChatMessage(nil), msgs...),
		expires:  time.Now().Add(m.ttl),
	}
}

func (m *threadMemory) gcLocked() {
	now := time.Now()
	for k, e := range m.threads {
		if now.After(e.expires) {
			delete(m.threads, k)
		}
	}
}
