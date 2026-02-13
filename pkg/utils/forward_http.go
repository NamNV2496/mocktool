package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	HeaderAccountId   = "X-Account-Id"
	HeaderFeatureName = "X-Feature-Name"
	ForwardPathPrefix = "/forward"
)

func ForwardHTTP(ctx context.Context, req *http.Request, path, accountId, featureName, baseURL, method string) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("forward http - read body: %w", err)
	}
	defer req.Body.Close()

	// targetURL := baseURL + ForwardPathPrefix + req.RequestURI
	targetURL := baseURL + ForwardPathPrefix + path
	// forwardReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, bytes.NewReader(body))
	forwardReq, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(body))
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
