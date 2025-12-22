package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	testgrpc "github.com/namnv2496/mocktool/example/grpc/proto/generated"
	"github.com/namnv2496/mocktool/pkg/errorcustome"
	pb "github.com/namnv2496/mocktool/pkg/generated/github.com/namnv/mockTool/pkg/errorcustome"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type TestController struct {
	testgrpc.UnimplementedTestServiceServer
}

func NewTestController() testgrpc.TestServiceServer {
	return &TestController{}
}

func main() {
	// Initialize gRPC controller
	controller := NewTestController()
	listener, err := net.Listen("tcp", ":9091")
	if err != nil {
		return
	}
	defer listener.Close()
	var opts = []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			validator.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			validator.StreamServerInterceptor(),
		),
	}
	server := grpc.NewServer(opts...)
	reflection.Register(server)
	testgrpc.RegisterTestServiceServer(server, controller)
	conn, err := grpc.NewClient(":9091", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return
	}
	defer conn.Close()
	// add custom http response
	muxOptions := make([]runtime.ServeMuxOption, 0)
	muxOptions = append(muxOptions, runtime.WithErrorHandler(customHttpResponse))

	mux := runtime.NewServeMux(muxOptions...)
	testgrpc.RegisterTestServiceHandler(context.Background(), mux, conn)
	go func() {
		server.Serve(listener)
	}()

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic("failed to start HTTP server")
	}
}

func (_self *TestController) TestAPI(ctx context.Context, req *testgrpc.TestRequest) (*testgrpc.TestResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal request failed: %v", err)
	}

	// 2️⃣ Build target URL
	targetURL := "http://localhost:8081/forward/api/v1/test"
	forwardReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		targetURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create http request failed: %v", err)
	}
	// parse accountId from token
	accountId := "1"
	forwardReq.Header.Set("Content-Type", "application/json")
	forwardReq.Header.Set("X-Feature-Name", "test_feature")
	forwardReq.Header.Set("X-Account-Id", accountId)

	// 3️⃣ Send request
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(forwardReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 4️⃣ Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read response failed: %v", err)
	}

	// =================================================WAY 1===================================================
	// forward error message from server
	if resp.StatusCode >= 400 {
		var errResp errorcustome.ErrorResponse
		json.Unmarshal(respBody, &errResp)

		// Reconstruct gRPC status error with details
		st := status.New(errResp.GrpcCode, errResp.ErrorMessage)
		stWithDetails, _ := st.WithDetails(&pb.ErrorDetail{
			ErrorCode: errResp.ErrorCode,
			Metadata:  errResp.Details,
		})
		return nil, stWithDetails.Err()
		// return nil, errorcustome.WrapErrorResponse(errResp)
	}

	// =================================================WAY 2===================================================
	// on ly get message from server and create new error with that error message
	if resp.StatusCode >= 400 {
		metadata := make(map[string]string, 0)
		metadata["x-trace-id"] = "jk3k49-234kfd934-fdk239d3-dk93dk3-d"

		// Extract error message from response
		var errResp map[string]interface{}
		var errorMessage string
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			if msg, ok := errResp["message"]; ok {
				errorMessage = fmt.Sprintf("%v", msg)
			} else {
				errorMessage = string(respBody)
			}
		} else {
			errorMessage = string(respBody)
		}
		return nil, errorcustome.NewError(codes.Internal, "ERR.001", "Forward error: %s", metadata, errorMessage)
	}
	// ====================================================================================================

	// 5️⃣ Unmarshal → proto response
	var out testgrpc.TestResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal response failed: %v", err)
	}

	return &out, nil
}

func customHttpResponse(ctx context.Context, _ *runtime.ServeMux, marshaller runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	customErrResp := errorcustome.ConvertToHttpResponse(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(customErrResp.HttpStatus)
	marshaller.NewEncoder(w).Encode(customErrResp)
}
