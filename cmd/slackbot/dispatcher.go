package slackbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/namnv2496/mocktool/internal/tools"
)

// Replier is the side-effect surface the dispatcher needs from Slack. The bot
// supplies a closure that posts a message into the originating thread; tests
// inject a fake to assert what was said.
type Replier func(ctx context.Context, text string) error

// Dispatcher runs one user-turn through the LLM, dispatches any tool calls
// (subject to destructive-confirmation), and posts the final assistant text.
//
// It does not handle Slack-specific concerns (events, channels, threads) —
// the bot package translates Slack events into Dispatcher.Handle calls.
type Dispatcher struct {
	llm     LLMClient
	model   string
	reg     *tools.Registry
	memory  *threadMemory
	confirm *confirmStore
	maxIter int

	// systemPrompt is sent as the first message of every conversation.
	systemPrompt string
}

// NewDispatcher constructs a Dispatcher. maxIter caps the tool-call loop to
// prevent runaway LLM↔tool ping-pongs. confirmStore is stamped with the bot's
// global TTL.
func NewDispatcher(llm LLMClient, model string, reg *tools.Registry, mem *threadMemory, conf *confirmStore, maxIter int) *Dispatcher {
	if maxIter <= 0 {
		maxIter = 5
	}
	return &Dispatcher{
		llm:     llm,
		model:   model,
		reg:     reg,
		memory:  mem,
		confirm: conf,
		maxIter: maxIter,
		systemPrompt: `You help users manage mocktool, a mock-API service.
Use the provided tools to answer questions and perform actions.
Be concise. When a destructive action is requested, the system will prompt the user for confirmation — you do not need to do that yourself; just call the tool.
If the user is ambiguous (e.g. asks about "the feature" without naming it), ask for clarification before calling tools.`,
	}
}

// HandleText processes one user message in a thread. If the text is a
// "confirm <token>" / "cancel <token>" reply to a pending destructive
// action, that branch is taken instead of consulting the LLM.
// If the text is a "/mock <contract>" command, the mock-generation flow runs.
func (d *Dispatcher) HandleText(ctx context.Context, threadTS, channel, text string, reply Replier) error {
	// Confirmation/cancellation short-circuit (no LLM call).
	if action, token, ok := parseConfirmCommand(text); ok {
		return d.handleConfirm(ctx, action, token, reply)
	}

	// /mock contract generation — dedicated flow.
	if isMockCommand(text) {
		return d.handleMockCommand(ctx, threadTS, channel, text, reply)
	}

	history := d.memory.Get(threadTS)
	if len(history) == 0 {
		history = []ChatMessage{{Role: "system", Content: d.systemPrompt}}
	}
	history = append(history, ChatMessage{Role: "user", Content: text})

	for i := 0; i < d.maxIter; i++ {
		resp, err := d.llm.Chat(ctx, ChatRequest{
			Model:    d.model,
			Messages: history,
			Tools:    d.reg.List(),
		})
		if err != nil {
			return fmt.Errorf("llm chat: %w", err)
		}

		// Record the assistant's turn before acting on it.
		history = append(history, ChatMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			// Plain answer — post it and persist history.
			final := resp.Content
			if final == "" {
				final = "(no response)"
			}
			d.memory.Append(threadTS, history)
			return reply(ctx, final)
		}

		// Process every tool call in this assistant turn.
		var needConfirmation []ToolCall
		for _, tc := range resp.ToolCalls {
			tool, ok := d.reg.Get(tc.Name)
			if !ok {
				history = append(history, ChatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf(`{"error":"unknown tool %q"}`, tc.Name),
				})
				continue
			}
			if tool.Destructive {
				needConfirmation = append(needConfirmation, tc)
				continue
			}
			result, err := d.reg.Invoke(ctx, tc.Name, tc.Args)
			payload := encodeToolResult(result, err)
			history = append(history, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    payload,
			})
		}

		// If any destructive tool was requested, pause the loop and ask the
		// user. We do NOT feed a tool_result back for those — the LLM should
		// not get to keep planning past a confirmation gate.
		if len(needConfirmation) > 0 {
			d.memory.Append(threadTS, history)
			return d.promptConfirm(ctx, needConfirmation, threadTS, channel, reply)
		}
	}

	d.memory.Append(threadTS, history)
	return reply(ctx, "Stopped: reached the max tool-call iterations. Try rephrasing.")
}

func (d *Dispatcher) handleConfirm(ctx context.Context, action, token string, reply Replier) error {
	p, ok := d.confirm.Claim(token)
	if !ok {
		return reply(ctx, "Confirmation expired or token unknown. Re-issue the original command.")
	}
	if action == "cancel" {
		return reply(ctx, fmt.Sprintf("Cancelled %s.", p.ToolName))
	}
	result, err := d.reg.Invoke(ctx, p.ToolName, p.Args)
	if err != nil {
		return reply(ctx, fmt.Sprintf("`%s` failed: %v", p.ToolName, err))
	}
	pretty, _ := json.MarshalIndent(result, "", "  ")
	return reply(ctx, fmt.Sprintf("`%s` done.\n```\n%s\n```", p.ToolName, string(pretty)))
}

func (d *Dispatcher) promptConfirm(ctx context.Context, calls []ToolCall, threadTS, channel string, reply Replier) error {
	if len(calls) == 1 {
		c := calls[0]
		p := d.confirm.Stash(c.Name, c.Args, threadTS, channel)
		args := compactJSON(c.Args)
		return reply(ctx, fmt.Sprintf(
			":warning: About to run *destructive* tool `%s` with args:\n```%s```\nReply `confirm %s` to proceed or `cancel %s`.",
			c.Name, args, p.Token, p.Token,
		))
	}
	// Multi-tool destructive call: stash each, ask once.
	var lines []string
	for _, c := range calls {
		p := d.confirm.Stash(c.Name, c.Args, threadTS, channel)
		lines = append(lines,
			fmt.Sprintf("- `%s` %s — `confirm %s` / `cancel %s`",
				c.Name, compactJSON(c.Args), p.Token, p.Token),
		)
	}
	return reply(ctx, ":warning: Multiple destructive tools queued. Confirm each:\n"+joinLines(lines))
}

func encodeToolResult(v any, err error) string {
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(b)
	}
	b, mErr := json.Marshal(v)
	if mErr != nil {
		return `{"error":"non-serializable tool result"}`
	}
	return string(b)
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func joinLines(s []string) string {
	if len(s) == 0 {
		return ""
	}
	out := s[0]
	for _, x := range s[1:] {
		out += "\n" + x
	}
	return out
}

// ErrEmptyMessage marks empty inputs the dispatcher should ignore.
var ErrEmptyMessage = errors.New("empty message")
