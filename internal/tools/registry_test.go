package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_ListPreservesInsertionOrder(t *testing.T) {
	r := NewRegistry(
		Tool{Name: "a", Handler: noopHandler},
		Tool{Name: "c", Handler: noopHandler},
		Tool{Name: "b", Handler: noopHandler},
	)

	got := r.List()
	require.Len(t, got, 3)
	assert.Equal(t, "a", got[0].Name)
	assert.Equal(t, "c", got[1].Name)
	assert.Equal(t, "b", got[2].Name)
}

func TestRegistry_DuplicateNamePanics(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover(), "expected panic on duplicate tool name")
	}()
	NewRegistry(
		Tool{Name: "x", Handler: noopHandler},
		Tool{Name: "x", Handler: noopHandler},
	)
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry(Tool{Name: "x", Description: "hi", Handler: noopHandler})

	got, ok := r.Get("x")
	require.True(t, ok)
	assert.Equal(t, "hi", got.Description)

	_, ok = r.Get("missing")
	assert.False(t, ok)
}

func TestRegistry_InvokeUnknownReturnsErrUnknownTool(t *testing.T) {
	r := NewRegistry()
	_, err := r.Invoke(context.Background(), "missing", nil)
	assert.ErrorIs(t, err, ErrUnknownTool)
}

func TestRegistry_InvokeCallsHandler(t *testing.T) {
	called := false
	r := NewRegistry(Tool{
		Name: "echo",
		Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
			called = true
			return map[string]string{"got": string(args)}, nil
		},
	})

	res, err := r.Invoke(context.Background(), "echo", json.RawMessage(`"hi"`))
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, map[string]string{"got": `"hi"`}, res)
}

func TestRegistry_InvokePropagatesHandlerError(t *testing.T) {
	want := errors.New("boom")
	r := NewRegistry(Tool{
		Name: "bad",
		Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
			return nil, want
		},
	})

	_, err := r.Invoke(context.Background(), "bad", nil)
	assert.ErrorIs(t, err, want)
}

func TestRegistry_MissingHandlerErrors(t *testing.T) {
	r := NewRegistry(Tool{Name: "no-handler"})
	_, err := r.Invoke(context.Background(), "no-handler", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no handler")
}

func noopHandler(ctx context.Context, args json.RawMessage) (any, error) { return nil, nil }
