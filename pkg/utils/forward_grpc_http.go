package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/namnv2496/mocktool/pkg/errorcustome"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func ForwardGRPCToHTTP(ctx context.Context, msg proto.Message, out proto.Message, path, accountId, featureName, baseURL, method string) error {
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	body, err := marshaler.Marshal(msg)
	if err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - marshal request: %v", err)
	}

	targetURL := baseURL + ForwardPathPrefix + path
	var req *http.Request

	// For GET and DELETE, convert body to query parameters
	if method == http.MethodGet || method == http.MethodDelete {
		// Parse JSON body into map
		var params map[string]interface{}
		if err := json.Unmarshal(body, &params); err != nil {
			return status.Errorf(codes.Internal, "forward grpc-http - unmarshal params: %v", err)
		}

		// Build query string
		queryValues := url.Values{}
		for key, value := range params {
			queryValues.Add(key, fmt.Sprintf("%v", value))
		}

		if len(queryValues) > 0 {
			targetURL = targetURL + "?" + queryValues.Encode()
		}

		req, err = http.NewRequestWithContext(ctx, method, targetURL, nil)
		if err != nil {
			return status.Errorf(codes.Internal, "forward grpc-http - create request: %v", err)
		}
	} else {
		// For POST, PUT, PATCH - keep body
		req, err = http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(body))
		if err != nil {
			return status.Errorf(codes.Internal, "forward grpc-http - create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set(HeaderAccountId, accountId)
	req.Header.Set(HeaderFeatureName, featureName)

	// Print curl command for Postman testing
	fmt.Printf("curl -X %s '%s' \\\n", method, targetURL)
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("  -H '%s: %s' \\\n", key, value)
		}
	}
	if method != http.MethodGet && method != http.MethodDelete {
		fmt.Printf("  -d '%s'\n\n", string(body))
	} else {
		fmt.Println()
	}

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

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(respBody, out); err != nil {
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

// example

/*
func (c *LocationController) LocationMapping(ctx context.Context, req *pb.LocationMappingRequest) (*pb.LocationMappingResponse, error) {
	var output pb.LocationMappingResponse
	err := ForwardGRPCToHTTP(ctx, req, &output, "/v1/public/locations/mapping", "1", "location_v3", "http://localhost:8082", http.MethodGet)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

func ForwardGRPCToHTTP(ctx context.Context, msg proto.Message, out proto.Message, path, accountId, featureName, baseURL, method string) error {
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	body, err := marshaler.Marshal(msg)
	if err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - marshal request: %v", err)
	}

	targetURL := baseURL + ForwardPathPrefix + path
	var req *http.Request

	// For GET and DELETE, convert body to query parameters
	if method == http.MethodGet || method == http.MethodDelete {
		// Parse JSON body into map
		var params map[string]interface{}
		if err := json.Unmarshal(body, &params); err != nil {
			return status.Errorf(codes.Internal, "forward grpc-http - unmarshal params: %v", err)
		}

		// Build query string
		queryValues := url.Values{}
		for key, value := range params {
			queryValues.Add(key, fmt.Sprintf("%v", value))
		}

		if len(queryValues) > 0 {
			targetURL = targetURL + "?" + queryValues.Encode()
		}

		req, err = http.NewRequestWithContext(ctx, method, targetURL, nil)
		if err != nil {
			return status.Errorf(codes.Internal, "forward grpc-http - create request: %v", err)
		}
	} else {
		// For POST, PUT, PATCH - keep body
		req, err = http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(body))
		if err != nil {
			return status.Errorf(codes.Internal, "forward grpc-http - create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set(HeaderAccountId, accountId)
	req.Header.Set(HeaderFeatureName, featureName)

	// Print curl command for Postman testing
	fmt.Printf("curl -X %s '%s' \\\n", method, targetURL)
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("  -H '%s: %s' \\\n", key, value)
		}
	}
	if method != http.MethodGet && method != http.MethodDelete {
		fmt.Printf("  -d '%s'\n\n", string(body))
	} else {
		fmt.Println()
	}

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

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(respBody, out); err != nil {
		return status.Errorf(codes.Internal, "forward grpc-http - unmarshal response: %v", err)
	}
	return nil
}
*/
