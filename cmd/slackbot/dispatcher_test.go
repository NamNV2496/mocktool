package slackbot

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/namnv2496/mocktool/internal/tools"
)

// fakeLLM serves a scripted sequence of responses.
type fakeLLM struct {
	mu        sync.Mutex
	idx       int
	responses []ChatResponse
	requests  []ChatRequest
}

func (f *fakeLLM) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if f.idx >= len(f.responses) {
		return ChatResponse{Content: "(out of script)"}, nil
	}
	out := f.responses[f.idx]
	f.idx++
	return out, nil
}

// captureReplier records every reply for assertions.
type capture struct {
	mu   sync.Mutex
	msgs []string
}

func (c *capture) reply() Replier {
	return func(_ context.Context, text string) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.msgs = append(c.msgs, text)
		return nil
	}
}

func buildDispatcher(t *testing.T, llm LLMClient, reg *tools.Registry) (*Dispatcher, *threadMemory, *confirmStore) {
	t.Helper()
	mem := newThreadMemory(time.Minute)
	conf := newConfirmStore(time.Minute)
	return NewDispatcher(llm, "gpt-test", reg, mem, conf, 5), mem, conf
}

func TestDispatcher_TextResponsePostedAndStored(t *testing.T) {
	reg := tools.NewRegistry()
	llm := &fakeLLM{responses: []ChatResponse{{Content: "hello"}}}
	d, mem, _ := buildDispatcher(t, llm, reg)
	cap := &capture{}

	err := d.HandleText(context.Background(), "TS1", "C", "hi", cap.reply())
	require.NoError(t, err)
	assert.Equal(t, []string{"hello"}, cap.msgs)
	assert.NotEmpty(t, mem.Get("TS1"))
}

func TestDispatcher_NonDestructiveToolCalledAndResultFedBack(t *testing.T) {
	called := false
	reg := tools.NewRegistry(tools.Tool{
		Name: "list_features",
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			called = true
			return map[string]any{"features": []string{"a"}}, nil
		},
	})
	llm := &fakeLLM{responses: []ChatResponse{
		{ToolCalls: []ToolCall{{ID: "tc1", Name: "list_features", Args: json.RawMessage(`{}`)}}},
		{Content: "you have 1 feature: a"},
	}}
	d, _, _ := buildDispatcher(t, llm, reg)
	cap := &capture{}

	err := d.HandleText(context.Background(), "TS1", "C", "list features", cap.reply())
	require.NoError(t, err)
	assert.True(t, called)
	require.Len(t, cap.msgs, 1)
	assert.Equal(t, "you have 1 feature: a", cap.msgs[0])

	// Second LLM turn must have seen the tool result message.
	require.Len(t, llm.requests, 2)
	last := llm.requests[1].Messages
	var sawToolMsg bool
	for _, m := range last {
		if m.Role == "tool" && m.ToolCallID == "tc1" {
			sawToolMsg = true
			assert.Contains(t, m.Content, "features")
		}
	}
	assert.True(t, sawToolMsg, "tool result should be fed back to LLM")
}

func TestDispatcher_DestructiveToolPausesForConfirmation(t *testing.T) {
	invoked := false
	reg := tools.NewRegistry(tools.Tool{
		Name:        "delete_feature",
		Destructive: true,
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			invoked = true
			return "ok", nil
		},
	})
	llm := &fakeLLM{responses: []ChatResponse{
		{ToolCalls: []ToolCall{{ID: "tc1", Name: "delete_feature", Args: json.RawMessage(`{"feature":"x"}`)}}},
	}}
	d, _, conf := buildDispatcher(t, llm, reg)
	cap := &capture{}

	err := d.HandleText(context.Background(), "TS1", "C", "delete feature x", cap.reply())
	require.NoError(t, err)
	assert.False(t, invoked, "destructive tool must not run before confirmation")
	require.Len(t, cap.msgs, 1)
	assert.Contains(t, cap.msgs[0], "destructive")

	// There should be exactly one pending confirmation.
	conf.mu.Lock()
	require.Len(t, conf.pending, 1)
	var token string
	for k := range conf.pending {
		token = k
	}
	conf.mu.Unlock()

	// Confirm runs the handler.
	err = d.HandleText(context.Background(), "TS1", "C", "confirm "+token, cap.reply())
	require.NoError(t, err)
	assert.True(t, invoked)
}

func TestDispatcher_CancelDoesNotInvoke(t *testing.T) {
	invoked := false
	reg := tools.NewRegistry(tools.Tool{
		Name:        "delete_feature",
		Destructive: true,
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			invoked = true
			return nil, nil
		},
	})
	llm := &fakeLLM{responses: []ChatResponse{
		{ToolCalls: []ToolCall{{ID: "tc1", Name: "delete_feature", Args: json.RawMessage(`{"feature":"x"}`)}}},
	}}
	d, _, conf := buildDispatcher(t, llm, reg)
	cap := &capture{}

	err := d.HandleText(context.Background(), "TS1", "C", "delete x", cap.reply())
	require.NoError(t, err)

	var token string
	conf.mu.Lock()
	for k := range conf.pending {
		token = k
	}
	conf.mu.Unlock()

	err = d.HandleText(context.Background(), "TS1", "C", "cancel "+token, cap.reply())
	require.NoError(t, err)
	assert.False(t, invoked)
	assert.Contains(t, cap.msgs[len(cap.msgs)-1], "Cancelled")
}

func TestDispatcher_UnknownTokenReportedAsExpired(t *testing.T) {
	reg := tools.NewRegistry()
	llm := &fakeLLM{}
	d, _, _ := buildDispatcher(t, llm, reg)
	cap := &capture{}

	err := d.HandleText(context.Background(), "TS1", "C", "confirm beefdead", cap.reply())
	require.NoError(t, err)
	require.Len(t, cap.msgs, 1)
	assert.Contains(t, cap.msgs[0], "expired")
}

func TestDispatcher_MaxIterationsCap(t *testing.T) {
	// LLM keeps asking for the same tool forever — the dispatcher must stop.
	reg := tools.NewRegistry(tools.Tool{
		Name: "noop",
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			return "ok", nil
		},
	})
	llm := &fakeLLM{
		responses: []ChatResponse{
			{ToolCalls: []ToolCall{{ID: "1", Name: "noop", Args: json.RawMessage(`{}`)}}},
			{ToolCalls: []ToolCall{{ID: "2", Name: "noop", Args: json.RawMessage(`{}`)}}},
			{ToolCalls: []ToolCall{{ID: "3", Name: "noop", Args: json.RawMessage(`{}`)}}},
			{ToolCalls: []ToolCall{{ID: "4", Name: "noop", Args: json.RawMessage(`{}`)}}},
			{ToolCalls: []ToolCall{{ID: "5", Name: "noop", Args: json.RawMessage(`{}`)}}},
		},
	}
	d, _, _ := buildDispatcher(t, llm, reg)
	cap := &capture{}

	err := d.HandleText(context.Background(), "TS1", "C", "loop", cap.reply())
	require.NoError(t, err)
	require.Len(t, cap.msgs, 1)
	assert.Contains(t, cap.msgs[0], "max tool-call iterations")
}

func TestDispatcher_UnknownToolFedBackAsError(t *testing.T) {
	reg := tools.NewRegistry()
	llm := &fakeLLM{responses: []ChatResponse{
		{ToolCalls: []ToolCall{{ID: "tc1", Name: "missing_tool", Args: json.RawMessage(`{}`)}}},
		{Content: "I cannot do that."},
	}}
	d, _, _ := buildDispatcher(t, llm, reg)
	cap := &capture{}

	err := d.HandleText(context.Background(), "TS1", "C", "x", cap.reply())
	require.NoError(t, err)
	assert.Equal(t, []string{"I cannot do that."}, cap.msgs)
}

func TestParseConfirmCommand(t *testing.T) {
	cases := []struct {
		in     string
		action string
		token  string
		ok     bool
	}{
		{"confirm abc123", "confirm", "abc123", true},
		{"CANCEL deadbeef", "cancel", "deadbeef", true},
		{"  confirm  xyz  ", "confirm", "xyz", true},
		{"confirm", "", "", false},
		{"hello world", "", "", false},
		{"confirm a b", "", "", false},
	}
	for _, c := range cases {
		a, tok, ok := parseConfirmCommand(c.in)
		assert.Equal(t, c.ok, ok, c.in)
		assert.Equal(t, c.action, a, c.in)
		assert.Equal(t, c.token, tok, c.in)
	}
}
