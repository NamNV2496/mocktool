package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// We use structpb.Value / structpb.Struct as real proto messages for testing
// without pulling in a generated proto from this repo.

func TestForwardGRPCToHTTP(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/forward/api/v1/test", r.URL.Path)
			assert.Equal(t, "acc-42", r.Header.Get(HeaderAccountId))
			assert.Equal(t, "svc_a", r.Header.Get(HeaderFeatureName))

			w.Header().Set("Content-Type", "application/json")
			// protojson parses a plain JSON object directly into structpb.Struct
			w.Write([]byte(`{"key":"val"}`))
		}))
		defer server.Close()

		input, err := structpb.NewValue("hello")
		require.NoError(t, err)

		out := &structpb.Struct{}
		err = ForwardGRPCToHTTP(ctx, input, out, "/api/v1/test", "acc-42", "svc_a", server.URL)
		require.NoError(t, err)
		assert.Contains(t, out.GetFields(), "key")
	})

	t.Run("UpstreamHTTPError", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"grpc_code":     float64(codes.InvalidArgument),
				"error_code":    "ERR.VALIDATION",
				"error_message": "field X is required",
				"details":       map[string]string{"field": "X"},
			})
		}))
		defer server.Close()

		input, _ := structpb.NewValue("data")
		out := &structpb.Struct{}
		err := ForwardGRPCToHTTP(ctx, input, out, "/api/v1/test", "acc-1", "feat", server.URL)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "field X is required")
	})

	t.Run("UpstreamHTTPErrorMalformedBody", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("plain text error"))
		}))
		defer server.Close()

		input, _ := structpb.NewValue("x")
		out := &structpb.Struct{}
		err := ForwardGRPCToHTTP(ctx, input, out, "/api/v1/test", "acc-1", "feat", server.URL)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "plain text error")
	})

	t.Run("UnmarshalResponseError", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not-json"))
		}))
		defer server.Close()

		input, _ := structpb.NewValue("x")
		out := &structpb.Struct{}
		err := ForwardGRPCToHTTP(ctx, input, out, "/api/v1/test", "acc-1", "feat", server.URL)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "unmarshal response")
	})

	t.Run("UpstreamUnreachable", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		addr := server.URL
		server.Close()

		input, _ := structpb.NewValue("x")
		out := &structpb.Struct{}
		err := ForwardGRPCToHTTP(ctx, input, out, "/api/v1/test", "acc-1", "feat", addr)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unavailable, st.Code())
	})
}
