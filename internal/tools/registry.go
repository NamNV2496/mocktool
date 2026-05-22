// Package tools defines a protocol-agnostic registry of callable tools that
// expose mocktool admin operations. Both the MCP server (cmd/mcpserver) and the
// Slack bot (cmd/slackbot) consume the same registry — this is the single
// source of truth for what the LLM and other MCP clients can do.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

// HandlerFunc receives raw JSON arguments and returns any JSON-serializable
// result. Returning an error signals a tool-level failure; the caller decides
// how to surface it (MCP error result, Slack message, etc.).
type HandlerFunc func(ctx context.Context, args json.RawMessage) (any, error)

// Tool is the protocol-agnostic description of a callable tool.
//
// InputSchema is a JSON Schema describing the args object. It is shipped
// verbatim to MCP clients and adapted to OpenAI function-calling on the Slack
// side.
//
// Destructive marks tools whose effect cannot be undone (deletes, deactivates).
// Slack-side dispatch holds destructive calls behind an explicit confirmation
// step; MCP clients receive a hint via the standard `destructiveHint`
// annotation.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	Destructive bool
	Handler     HandlerFunc
}

// ErrUnknownTool is returned by Registry.Invoke when no tool matches the name.
var ErrUnknownTool = fmt.Errorf("unknown tool")

// Registry holds tools indexed by name with stable insertion order.
type Registry struct {
	byName map[string]Tool
	order  []string
}

// NewRegistry constructs a Registry from the given tools. Duplicate names
// panic at construction — the registry is wired at process start, so a
// duplicate is a programming error worth failing loudly.
func NewRegistry(ts ...Tool) *Registry {
	r := &Registry{byName: make(map[string]Tool, len(ts))}
	for _, t := range ts {
		if _, dup := r.byName[t.Name]; dup {
			panic(fmt.Sprintf("tools: duplicate tool name %q", t.Name))
		}
		r.byName[t.Name] = t
		r.order = append(r.order, t.Name)
	}
	return r
}

// List returns tools in insertion order.
func (r *Registry) List() []Tool {
	out := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.byName[name])
	}
	return out
}

// Names returns the names of all registered tools, sorted for stable output.
func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	sort.Strings(out)
	return out
}

// Get looks up a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.byName[name]
	return t, ok
}

// Invoke dispatches to the named tool's handler.
func (r *Registry) Invoke(ctx context.Context, name string, args json.RawMessage) (any, error) {
	t, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTool, name)
	}
	if t.Handler == nil {
		return nil, fmt.Errorf("tools: %q has no handler", name)
	}
	return t.Handler(ctx, args)
}
