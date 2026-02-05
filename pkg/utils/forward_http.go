package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	HeaderAccountId  = "X-Account-Id"
	HeaderFeatureName = "X-Feature-Name"
	ForwardPathPrefix = "/forward"
)

/*
ForwardHTTP forwards an incoming HTTP request to the mock tool's /forward endpoint.
It copies the original method, path, headers, and body, then injects
X-Account-Id and X-Feature-Name headers before sending.
*/
func ForwardHTTP(ctx context.Context, req *http.Request, accountId, featureName, baseURL string) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("forward http - read body: %w", err)
	}
	defer req.Body.Close()

	targetURL := baseURL + ForwardPathPrefix + req.RequestURI
	forwardReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("forward http - create request: %w", err)
	}

	copyHeaders(forwardReq.Header, req.Header)
	forwardReq.Header.Set(HeaderAccountId, accountId)
	forwardReq.Header.Set(HeaderFeatureName, featureName)

	resp, err := http.DefaultClient.Do(forwardReq)
	if err != nil {
		return nil, fmt.Errorf("forward http - send request: %w", err)
	}
	return resp, nil
}

func copyHeaders(dst, src http.Header) {
	for k, vals := range src {
		dst[k] = vals
	}
}
