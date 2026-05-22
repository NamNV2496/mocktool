package chat

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"

	"github.com/namnv2496/mocktool/internal/tools"
)

// ChatMessage is the protocol-agnostic conversation turn used by the HTTP API.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}

type toolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// Handler is a stateless HTTP-oriented AI chat handler. The caller supplies
// the full conversation history on every request; the handler executes the
// LLM+tool loop and returns the final text reply.
type Handler struct {
	reg    *tools.Registry
	client *openai.Client
	model  string
}

const systemPrompt = `You help users manage mocktool, a mock-API service.
Use the provided tools to answer questions and perform actions.
Be concise and helpful. When listing results, format them clearly.`

// New constructs a Handler. If apiKey is empty the handler will return an
// error on every Chat call with a hint to set OPENAI_API_KEY.
func New(reg *tools.Registry, apiKey, model string) *Handler {
	return NewWithEndpoint(reg, apiKey, model, "https://openrouter.ai/api/v1")
}

// NewWithEndpoint creates a Handler with a custom API endpoint (for OpenRouter, etc).
func NewWithEndpoint(reg *tools.Registry, apiKey, model, endpoint string) *Handler {
	if model == "" {
		model = "openrouter/auto"
	}
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = endpoint
	return &Handler{
		reg:    reg,
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

// Chat runs one user message through the LLM+tool dispatch loop and returns
// the final assistant reply. history is the prior conversation (excluding the
// new message) so the handler remains stateless — the client owns memory.
func (h *Handler) Chat(ctx context.Context, history []ChatMessage, text string) (string, error) {
	msgs := []ChatMessage{{Role: "system", Content: systemPrompt}}
	msgs = append(msgs, history...)
	msgs = append(msgs, ChatMessage{Role: "user", Content: text})

	for range 5 {
		resp, err := h.callLLM(ctx, msgs)
		if err != nil {
			return "", err
		}
		msgs = append(msgs, ChatMessage{
			Role:      "assistant",
			Content:   resp.content,
			ToolCalls: resp.toolCalls,
		})
		if len(resp.toolCalls) == 0 {
			if resp.content == "" {
				return "(no response)", nil
			}
			return resp.content, nil
		}
		for _, tc := range resp.toolCalls {
			result, err := h.reg.Invoke(ctx, tc.Name, tc.Args)
			msgs = append(msgs, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    encodeResult(result, err),
			})
		}
	}
	return "Stopped: reached the max tool-call iterations. Try rephrasing.", nil
}

type llmResp struct {
	content   string
	toolCalls []toolCall
}

func (h *Handler) callLLM(ctx context.Context, msgs []ChatMessage) (llmResp, error) {
	oaiMsgs := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, m := range msgs {
		om := openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, openai.ToolCall{
				ID:   tc.ID,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      tc.Name,
					Arguments: string(tc.Args),
				},
			})
		}
		oaiMsgs = append(oaiMsgs, om)
	}

	toolSchemas := make([]openai.Tool, 0, len(h.reg.List()))
	for _, t := range h.reg.List() {
		toolSchemas = append(toolSchemas, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(t.InputSchema),
			},
		})
	}

	r, err := h.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    h.model,
		Messages: oaiMsgs,
		Tools:    toolSchemas,
	})
	if err != nil {
		return llmResp{}, fmt.Errorf("openai: %w", err)
	}
	if len(r.Choices) == 0 {
		return llmResp{}, fmt.Errorf("openai: no choices in response")
	}
	choice := r.Choices[0]
	out := llmResp{content: choice.Message.Content}
	for _, tc := range choice.Message.ToolCalls {
		out.toolCalls = append(out.toolCalls, toolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: json.RawMessage(tc.Function.Arguments),
		})
	}
	return out, nil
}

func encodeResult(v any, err error) string {
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(b)
	}
	b, mErr := json.Marshal(v)
	if mErr != nil {
		return `{"error":"non-serializable result"}`
	}
	return string(b)
}
