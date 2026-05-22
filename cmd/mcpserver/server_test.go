package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/namnv2496/mocktool/internal/tools"
)

func TestBuild_NilRegistryErrors(t *testing.T) {
	_, err := Build(nil, Config{})
	require.Error(t, err)
}

func TestBuild_RegistersEveryTool(t *testing.T) {
	reg := tools.NewRegistry(
		tools.Tool{
			Name:        "echo",
			Description: "echo args",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"v":{"type":"string"}}}`),
			Handler: func(_ context.Context, args json.RawMessage) (any, error) {
				return map[string]string{"got": string(args)}, nil
			},
		},
		tools.Tool{
			Name:        "boom",
			Description: "errors out",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
				return nil, errors.New("nope")
			},
		},
	)
	sse, err := Build(reg, Config{Name: "test"})
	require.NoError(t, err)
	assert.NotNil(t, sse)
}
