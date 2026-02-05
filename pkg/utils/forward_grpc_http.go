package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/namnv2496/mocktool/pkg/errorcustome"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

/*
ForwardGRPCToHTTP marshals a proto message to JSON, forwards it to the mock
tool's HTTP /forward endpoint, and unmarshals the JSON response back into out.
On HTTP >= 400 it returns an errorcustome error that preserves grpc code and details.
*/
func ForwardGRPCToHTTP(ctx context.Context, msg proto.Message, out proto.Message, path, accountId, featureName, baseURL string) error {
	body, err := protojson.Marshal(msg)
	if err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - marshal request: %v", err)
	}

	targetURL := baseURL + ForwardPathPrefix + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderAccountId, accountId)
	req.Header.Set(HeaderFeatureName, featureName)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return status.Errorf(codes.Unavailable, "forward grpc-http - send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - read response: %v", err)
	}

	if resp.StatusCode >= 400 {
		return parseHTTPError(respBody)
	}

	if err := protojson.Unmarshal(respBody, out); err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - unmarshal response: %v", err)
	}
	return nil
}

func parseHTTPError(body []byte) error {
	var errResp errorcustome.ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - upstream error: %s", string(body))
	}
	return errorcustome.NewError(
		errResp.GrpcCode,
		errResp.ErrorCode,
		errResp.ErrorMessage,
		errResp.Details,
	)
}
