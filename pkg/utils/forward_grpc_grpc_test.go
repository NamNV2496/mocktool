package utils

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestForwardGRPCToGRPC(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		call := NewGRPCCallOption(func(_ context.Context, _ *metadata.MD) error {
			return nil
		})

		err := ForwardGRPCToGRPC(ctx, call)
		require.NoError(t, err)
	})

	t.Run("UpstreamErrorWithTrailer", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		expectedMD := metadata.New(map[string]string{
			"x-trace-id": "trace-abc",
		})

		call := NewGRPCCallOption(func(_ context.Context, md *metadata.MD) error {
			*md = expectedMD
			return errors.New("upstream rpc failed")
		})

		err := ForwardGRPCToGRPC(ctx, call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream rpc failed")
	})

	t.Run("UpstreamErrorNoTrailer", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		call := NewGRPCCallOption(func(_ context.Context, _ *metadata.MD) error {
			return errors.New("no trailer error")
		})

		err := ForwardGRPCToGRPC(ctx, call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no trailer error")
	})
}

func TestNewGRPCCallOption(t *testing.T) {
	t.Parallel()

	t.Run("WiresTrailerPointer", func(t *testing.T) {
		t.Parallel()

		sentMD := metadata.New(map[string]string{"key": "val"})
		opt := NewGRPCCallOption(func(_ context.Context, md *metadata.MD) error {
			*md = sentMD
			return nil
		})

		md, err := opt.Invoke(t.Context())
		require.NoError(t, err)
		assert.Equal(t, []string{"val"}, md.Get("key"))
	})
}
