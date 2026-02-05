package utils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForwardHTTP(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/forward/api/v1/users", r.URL.Path)
			assert.Equal(t, "acc-123", r.Header.Get(HeaderAccountId))
			assert.Equal(t, "my_feature", r.Header.Get(HeaderFeatureName))
			assert.Equal(t, "keep-this", r.Header.Get("X-Custom"))

			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, `{"name":"test"}`, string(body))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"1"}`))
		}))
		defer server.Close()

		incomingReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"test"}`))
		require.NoError(t, err)
		incomingReq.Header.Set("X-Custom", "keep-this")
		incomingReq.RequestURI = "/api/v1/users"

		resp, err := ForwardHTTP(ctx, incomingReq, "acc-123", "my_feature", server.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, `{"id":"1"}`, string(respBody))
	})

	t.Run("ReadBodyError", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		incomingReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/items", nil)
		require.NoError(t, err)
		incomingReq.Body = io.NopCloser(errorReader{})
		incomingReq.RequestURI = "/api/v1/items"

		_, err = ForwardHTTP(ctx, incomingReq, "acc-1", "feat", "http://localhost")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read body")
	})

	t.Run("InvalidBaseURL", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		incomingReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "/ping", strings.NewReader(""))
		require.NoError(t, err)
		incomingReq.RequestURI = "/ping"

		_, err = ForwardHTTP(ctx, incomingReq, "acc-1", "feat", "://bad-url")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create request")
	})

	t.Run("UpstreamUnreachable", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		incomingReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "/ping", strings.NewReader(""))
		require.NoError(t, err)
		incomingReq.RequestURI = "/ping"

		// Point to a closed server so the TCP dial fails
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		addr := server.URL
		server.Close()

		_, err = ForwardHTTP(ctx, incomingReq, "acc-1", "feat", addr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "send request")
	})
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
