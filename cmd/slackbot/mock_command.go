package slackbot

import (
	"context"
	"fmt"
	"strings"
)

// isMockCommand reports whether text is a /mock contract generation command.
func isMockCommand(text string) bool {
	t := strings.TrimSpace(text)
	return strings.EqualFold(t, "/mock") ||
		strings.HasPrefix(strings.ToLower(t), "/mock ")
}

// parseMockArgs extracts optional inline flags from the /mock line:
//
//	/mock feature=referrals scenario=default <contract…>
//
// Unrecognised tokens are left in the returned contract string.
func parseMockArgs(text string) (feature, scenario, contract string) {
	// Strip the /mock prefix.
	after := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "/mock"))
	after = strings.TrimPrefix(after, "/mock") // idempotent for edge-cases

	var rest []string
	for _, tok := range strings.Fields(splitFirstLine(after)) {
		if strings.HasPrefix(tok, "feature=") {
			feature = strings.TrimPrefix(tok, "feature=")
		} else if strings.HasPrefix(tok, "scenario=") {
			scenario = strings.TrimPrefix(tok, "scenario=")
		} else {
			rest = append(rest, tok)
		}
	}

	// Everything from the first non-flag token onward is the contract body,
	// but we keep the full multi-line text after the first line.
	firstLine := splitFirstLine(after)
	contractStart := strings.Index(after, firstLine)
	bodyAfterFirstLine := strings.TrimSpace(after[contractStart+len(firstLine):])

	if len(rest) > 0 {
		contract = strings.Join(rest, " ") + "\n" + bodyAfterFirstLine
	} else {
		contract = bodyAfterFirstLine
	}
	contract = strings.TrimSpace(contract)
	return
}

func splitFirstLine(s string) string {
	if idx := strings.IndexAny(s, "\n\r"); idx >= 0 {
		return s[:idx]
	}
	return s
}

// buildMockPrompt constructs the enriched LLM prompt for the /mock command.
func buildMockPrompt(feature, scenario, contract string) string {
	var sb strings.Builder

	sb.WriteString("You are a mock-API generator. From the API contract below, create mock APIs that cover every example case.\n\n")

	sb.WriteString("## Instructions\n")
	sb.WriteString("1. **One feature** per API (use the provided feature name, or derive it from the API path — e.g. `referrals` from `/api-mf/private/referrals`).\n")
	sb.WriteString("2. Check whether the feature already exists (`list_features` / `search_mocks`). Create it only if missing.\n")
	sb.WriteString("3. **One scenario per logical case** (success variants, error cases). Name them descriptively:\n")
	sb.WriteString("   - Success cases: `success_<variant>` (e.g. `success_jobs`, `success_pty`)\n")
	sb.WriteString("   - Error cases: `error_<code_or_reason>` (e.g. `error_login_required`)\n")
	sb.WriteString("   - If the contract lists multiple responses for the same request, use the `responses` array with `from`/`to`/`status_code` entries.\n")
	sb.WriteString("4. **Per scenario** call `create_scenario` then `create_mock_api`:\n")
	sb.WriteString("   - Set `request_body` to the example request JSON (or omit if none).\n")
	sb.WriteString("   - Set `response` to the example response JSON.\n")
	sb.WriteString("   - Set `status_code` to the HTTP status: 200 for success, 400/401/403/404/500 for errors.\n")
	sb.WriteString("   - `method` from the contract (GET, POST, PUT, etc.).\n")
	sb.WriteString("   - `path` exactly as in the contract.\n")
	sb.WriteString("5. After all mock APIs are created, reply with a summary table: scenario name, path, method, status code.\n\n")

	if feature != "" {
		sb.WriteString(fmt.Sprintf("## Requested feature name\n`%s`\n\n", feature))
	}
	if scenario != "" {
		sb.WriteString(fmt.Sprintf("## Requested base scenario name\n`%s` (use as prefix or as-is for the primary case)\n\n", scenario))
	}

	sb.WriteString("## API Contract\n")
	sb.WriteString("```\n")
	sb.WriteString(contract)
	sb.WriteString("\n```\n\n")
	sb.WriteString("Now create the feature, scenarios, and mock APIs. Start by checking if the feature already exists.")

	return sb.String()
}

// handleMockCommand processes /mock <contract> by building a rich LLM prompt
// and running it through the normal tool-call loop.
func (d *Dispatcher) handleMockCommand(ctx context.Context, threadTS, _ /*channel*/ string, text string, reply Replier) error {
	feature, scenario, contract := parseMockArgs(text)

	if contract == "" {
		return reply(ctx, ":wave: Paste your API contract after `/mock`. Optionally add `feature=<name>` or `scenario=<name>`.\n\nExample:\n```\n/mock feature=referrals\nAPI-001: GET — /api/referrals\nRequest: {\"vertical\":\"JOBS\"}\nResponse: {\"data\":{...}}\nError: {\"error_code\":\"ERR-001\",...}\n```")
	}

	prompt := buildMockPrompt(feature, scenario, contract)

	// Route through the standard LLM + tool-call loop.
	history := d.memory.Get(threadTS)
	if len(history) == 0 {
		history = []ChatMessage{{Role: "system", Content: d.systemPrompt}}
	}
	history = append(history, ChatMessage{Role: "user", Content: prompt})

	_ = reply(ctx, ":hammer_and_wrench: Parsing contract and generating mock APIs… (this may take a few turns)")

	for i := 0; i < d.maxIter; i++ {
		resp, err := d.llm.Chat(ctx, ChatRequest{
			Model:    d.model,
			Messages: history,
			Tools:    d.reg.List(),
		})
		if err != nil {
			return fmt.Errorf("llm chat: %w", err)
		}

		history = append(history, ChatMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			final := resp.Content
			if final == "" {
				final = "(no response)"
			}
			d.memory.Append(threadTS, history)
			return reply(ctx, final)
		}

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
				// /mock never calls destructive tools — skip silently.
				history = append(history, ChatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    `{"error":"destructive tools are not allowed in /mock generation"}`,
				})
				continue
			}
			result, err := d.reg.Invoke(ctx, tc.Name, tc.Args)
			history = append(history, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    encodeToolResult(result, err),
			})
		}
	}

	d.memory.Append(threadTS, history)
	return reply(ctx, "Stopped: reached max iterations while generating mocks. The partial results have been saved — try `/mock` again with more specific names.")
}
