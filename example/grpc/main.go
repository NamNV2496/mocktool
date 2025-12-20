package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	schedulerv1 "github.com/namnv2496/mocktool/example/grpc/proto/generated/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type SchedulerEventController struct {
	schedulerv1.UnimplementedSchedulerEventServiceServer
}

func NewSchedulerEventController() schedulerv1.SchedulerEventServiceServer {
	return &SchedulerEventController{}
}

func main() {
	// Initialize gRPC controller
	controller := NewSchedulerEventController()
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
	schedulerv1.RegisterSchedulerEventServiceServer(server, controller)
	conn, err := grpc.NewClient(":9091", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return
	}
	defer conn.Close()
	mux := runtime.NewServeMux()
	schedulerv1.RegisterSchedulerEventServiceHandler(context.Background(), mux, conn)
	go func() {
		server.Serve(listener)
	}()
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic("failed to start HTTP server")
	}
}

type Request struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

func (_self *SchedulerEventController) GetSchedulerEvents(
	ctx context.Context,
	req *schedulerv1.GetSchedulerEventsRequest,
) (*schedulerv1.GetSchedulerEventsResponse, error) {

	// 1️⃣ Marshal proto request → JSON
	requestBody := Request{
		Id:   123,
		Name: "test",
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal request failed: %v", err)
	}

	// 2️⃣ Build target URL
	targetURL := "http://localhost:8081/forward/api/v1/test?feature_name=test_feature"
	forwardReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		targetURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create http request failed: %v", err)
	}
	forwardReq.Header.Set("Content-Type", "application/json")

	// 3️⃣ Send request
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(forwardReq)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "forward request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, status.Errorf(
			codes.Internal,
			"forward service error: %s",
			string(b),
		)
	}

	// 4️⃣ Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read response failed: %v", err)
	}

	// 5️⃣ Unmarshal → proto response
	var out schedulerv1.GetSchedulerEventsResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal response failed: %v", err)
	}

	return &out, nil
}
