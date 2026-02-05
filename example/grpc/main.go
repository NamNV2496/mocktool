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

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	testgrpc "github.com/namnv2496/mocktool/example/grpc/proto/generated"
	"github.com/namnv2496/mocktool/pkg/errorcustome"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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
	// =================================================WAY 1===================================================
	return callHttp(ctx, req)
	// =================================================WAY 2===================================================
	return callGrpc(ctx, req)
}

func callHttp(ctx context.Context, req *testgrpc.TestRequest) (*testgrpc.TestResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal request failed: %v", err)
	}

	// 2️⃣ Build target URL
	targetURL := "http://localhost:8082/forward/api/v1/test"
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
	// throw new error with forwarding error from server
	if resp.StatusCode >= 400 {
		var errResp errorcustome.ErrorResponse
		json.Unmarshal(respBody, &errResp)

		return nil, errorcustome.NewError(
			errResp.GrpcCode,
			errResp.ErrorCode,
			errResp.ErrorMessage,
			errResp.Details,
			nil,
		)
	}

	// =================================================WAY 2===================================================
	// throw new error with new trace-id
	if resp.StatusCode >= 400 {
		metadata := make(map[string]string, 0)
		metadata["x-trace-id"] = uuid.NewString()

		// Extract error message from response
		var errResp map[string]any
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

	// =================================================WAY 3===================================================
	if resp.StatusCode >= 400 {
		// set trace-id because it is http. If call grpc it can get from ctx
		var errResp errorcustome.ErrorResponse
		json.Unmarshal(respBody, &errResp)

		// Set context
		md := metadata.New(map[string]string{
			"x-trace-id":  errResp.TraceId,
			"error_code":  errResp.ErrorCode,
			"http_status": fmt.Sprint(errResp.HttpStatus),
		})
		grpc.SetTrailer(ctx, md)
		return nil, fmt.Errorf("error: %s", errResp.ErrorMessage)
	}
	// ====================================================================================================
	// 5️⃣ Unmarshal → proto response
	var out testgrpc.TestResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal response failed: %v", err)
	}
	return &out, nil
}

func callGrpc(ctx context.Context, req *testgrpc.TestRequest) (*testgrpc.TestResponse, error) {
	// for test only
	conn, err := grpc.NewClient("anotherService:9092", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	anotherServiceClient := testgrpc.NewTestServiceClient(conn)
	var md metadata.MD
	// read metadata from grpc trailer
	_, err = anotherServiceClient.AnotherServiceFunc(ctx, nil, grpc.Trailer(&md))
	// fake server response
	// ======================================for test ONLY======================================
	md, err = fakeGrpcServerRespErr(ctx)
	// ======================================end for test ONLY======================================

	if err != nil {
		// forward grpc trailer to current context
		grpc.SetTrailer(ctx, md)
		if st, ok := status.FromError(err); ok {
			if val := md.Get("x-trace-id"); len(val) > 0 {
				grpc.SetTrailer(ctx, md)
			}
			return nil, fmt.Errorf("%s", st.Message())
		}
		return nil, err
	}
	return nil, nil
}

func fakeGrpcServerRespErr(ctx context.Context) (metadata.MD, error) {
	md := metadata.New(map[string]string{
		"x-trace-id":    uuid.NewString(),
		"error_code":    "ERR.001",
		"error_message": "internal server error",
	})
	// set trailer for response context
	grpc.SetTrailer(ctx, md)
	return md, fmt.Errorf("fakeGrpcServerRespErr")
}

func customHttpResponse(ctx context.Context, _ *runtime.ServeMux, marshaller runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	customErrResp := errorcustome.WrapErrorResponse(ctx, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(customErrResp.HttpStatus)
	marshaller.NewEncoder(w).Encode(customErrResp)
}
