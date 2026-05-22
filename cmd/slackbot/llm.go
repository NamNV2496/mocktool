package slackbot

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"

	"github.com/namnv2496/mocktool/internal/tools"
)

// LLMClient is the minimal surface the dispatcher needs from a chat-completion
// provider with tool calling. The OpenAI implementation lives in openai_client.go;
// tests inject a fake.
type LLMClient interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// ChatRequest is the protocol-agnostic shape of a chat-completion turn.
type ChatRequest struct {
	Model    string
	Messages []ChatMessage
	Tools    []tools.Tool
}

// ChatMessage matches OpenAI's role/content/tool_calls shape, but kept independent
// so we can swap providers without touching the dispatcher.
type ChatMessage struct {
	Role       string     // "system" | "user" | "assistant" | "tool"
	Content    string     // textual content (may be empty when assistant only emits tool_calls)
	Name       string     // optional speaker name
	ToolCallID string     // set when Role=="tool"
	ToolCalls  []ToolCall // set when assistant invokes tools
}

// ToolCall is one function-call request from the assistant.
type ToolCall struct {
	ID   string
	Name string
	Args json.RawMessage
}

// ChatResponse is what the dispatcher acts on.
type ChatResponse struct {
	Content   string
	ToolCalls []ToolCall
	// FinishReason mirrors OpenAI's: "stop", "tool_calls", "length", etc.
	FinishReason string
}

// openAIClient is the production LLMClient backed by sashabaranov/go-openai.
type openAIClient struct {
	c *openai.Client
}

// NewOpenAIClient constructs a real client; pass empty baseURL for default.
func NewOpenAIClient(apiKey, baseURL string) LLMClient {
	if baseURL == "" {
		return &openAIClient{c: openai.NewClient(apiKey)}
	}
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return &openAIClient{c: openai.NewClientWithConfig(cfg)}
}

func (o *openAIClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	msgs := make([]openai.ChatCompletionMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		om := openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
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
		msgs = append(msgs, om)
	}

	toolSchemas := make([]openai.Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		toolSchemas = append(toolSchemas, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(t.InputSchema),
			},
		})
	}

	model := req.Model
	if model == "" {
		model = openai.GPT4o
	}

	resp, err := o.c.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    model,
		Messages: msgs,
		Tools:    toolSchemas,
	})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("openai: %w", err)
	}
	if len(resp.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("openai: no choices in response")
	}
	choice := resp.Choices[0]
	out := ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
	}
	for _, tc := range choice.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: json.RawMessage(tc.Function.Arguments),
		})
	}
	return out, nil
}
