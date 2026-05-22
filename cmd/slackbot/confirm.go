package slackbot

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// pendingConfirm represents a destructive tool call awaiting user confirmation.
type pendingConfirm struct {
	Token    string
	ToolName string
	Args     json.RawMessage
	ThreadTS string
	Channel  string
	Expires  time.Time
}

// confirmStore holds pending destructive actions keyed by their token. Tokens
// are short random hex strings shown to the user; on receipt of "confirm <token>"
// the bot looks up the entry and executes it. Entries auto-expire.
type confirmStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	pending map[string]pendingConfirm
}

func newConfirmStore(ttl time.Duration) *confirmStore {
	return &confirmStore{ttl: ttl, pending: make(map[string]pendingConfirm)}
}

// Stash records a pending action and returns its token.
func (s *confirmStore) Stash(toolName string, args json.RawMessage, threadTS, channel string) pendingConfirm {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	token := genToken()
	p := pendingConfirm{
		Token:    token,
		ToolName: toolName,
		Args:     args,
		ThreadTS: threadTS,
		Channel:  channel,
		Expires:  time.Now().Add(s.ttl),
	}
	s.pending[token] = p
	return p
}

// Claim atomically removes and returns the pending action for the given token.
// Returns ok=false if the token is unknown or expired.
func (s *confirmStore) Claim(token string) (pendingConfirm, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	p, ok := s.pending[token]
	if !ok {
		return pendingConfirm{}, false
	}
	delete(s.pending, token)
	return p, true
}

func (s *confirmStore) gcLocked() {
	now := time.Now()
	for k, p := range s.pending {
		if now.After(p.Expires) {
			delete(s.pending, k)
		}
	}
}

// parseConfirmCommand recognizes "confirm <token>" or "cancel <token>" in a
// trimmed message, ignoring case. Returns the action and token, or ok=false.
func parseConfirmCommand(text string) (action, token string, ok bool) {
	s := strings.TrimSpace(text)
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return "", "", false
	}
	switch strings.ToLower(parts[0]) {
	case "confirm", "cancel":
		return strings.ToLower(parts[0]), parts[1], true
	}
	return "", "", false
}

func genToken() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
