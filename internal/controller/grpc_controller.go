package controller

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	grpc_reflection_v1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/internal/usecase"
)

func init() {
	// Override the default "proto" codec so UnknownServiceHandler can receive
	// and send raw bytes without a concrete proto.Message type.
	encoding.RegisterCodec(rawBytesCodec{})
}

// rawBytesCodec handles []byte in addition to proto.Message.
// Registered as "proto" so it replaces the default codec on this server.
type rawBytesCodec struct{}

func (rawBytesCodec) Name() string { return "proto" }

func (rawBytesCodec) Marshal(v any) ([]byte, error) {
	switch m := v.(type) {
	case []byte:
		return m, nil
	case proto.Message:
		return proto.Marshal(m)
	default:
		return nil, fmt.Errorf("rawBytesCodec: unsupported type %T", v)
	}
}

func (rawBytesCodec) Unmarshal(data []byte, v any) error {
	switch m := v.(type) {
	case *[]byte:
		*m = data
		return nil
	case proto.Message:
		return proto.Unmarshal(data, m)
	default:
		return fmt.Errorf("rawBytesCodec: unsupported type %T", v)
	}
}

type IGRPCController interface {
	StartGRPCServer() error
}

type GRPCController struct {
	config       *configs.Config
	grpcForward  usecase.IGRPCForwardUC
	grpcMockRepo repository.IGRPCMockAPIRepository
}

func NewGRPCController(
	config *configs.Config,
	grpcForward usecase.IGRPCForwardUC,
	grpcMockRepo repository.IGRPCMockAPIRepository,
) IGRPCController {
	return &GRPCController{
		config:       config,
		grpcForward:  grpcForward,
		grpcMockRepo: grpcMockRepo,
	}
}

func (_self *GRPCController) StartGRPCServer() error {
	addr := _self.config.AppConfig.GRPCPort
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := grpc.NewServer(
		grpc.UnknownServiceHandler(_self.unknownServiceHandler),
	)

	// Register dynamic reflection so Postman / grpcurl can discover services.
	dynSvcs := &dynamicServices{repo: _self.grpcMockRepo}
	reflSrv := reflection.NewServer(reflection.ServerOptions{
		Services:           dynSvcs,
		DescriptorResolver: dynSvcs,
	})
	grpc_reflection_v1alpha.RegisterServerReflectionServer(srv, reflSrv)

	slog.Info("gRPC mock server listening", "addr", addr)
	return srv.Serve(lis)
}

func (_self *GRPCController) unknownServiceHandler(_ any, stream grpc.ServerStream) error {
	ctx := stream.Context()
	fullMethod, _ := grpc.Method(ctx)

	md, _ := metadata.FromIncomingContext(ctx)

	featureName := firstMDValue(md, "x-feature-name")
	if featureName == "" {
		return status.Error(codes.InvalidArgument, "x-feature-name metadata is required")
	}

	var accountID *string
	if vals := md.Get("x-account-id"); len(vals) > 0 {
		v := vals[0]
		accountID = &v
	}

	var reqBytes []byte
	if err := stream.RecvMsg(&reqBytes); err != nil {
		return status.Errorf(codes.Internal, "receive request: %v", err)
	}

	result, grpcCode, err := _self.grpcForward.HandleCall(
		context.WithoutCancel(ctx),
		fullMethod,
		reqBytes,
		featureName,
		accountID,
	)
	if err != nil {
		return err
	}
	if grpcCode != codes.OK {
		return status.Error(grpcCode, "mock status")
	}

	respBytes, err := proto.Marshal(result)
	if err != nil {
		return status.Errorf(codes.Internal, "marshal response: %v", err)
	}

	return stream.SendMsg(respBytes)
}

func firstMDValue(md metadata.MD, key string) string {
	if vals := md.Get(key); len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// dynamicServices implements ServiceInfoProvider and protodesc.Resolver so that
// gRPC reflection advertises services and methods from the mock database.
// Postman and grpcurl can then discover services without a .proto file.
type dynamicServices struct {
	repo repository.IGRPCMockAPIRepository
}

// GetServiceInfo returns the unique service names stored in the mock database.
func (d *dynamicServices) GetServiceInfo() map[string]grpc.ServiceInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mocks, err := d.repo.ListAll(ctx)
	if err != nil {
		return map[string]grpc.ServiceInfo{}
	}
	info := make(map[string]grpc.ServiceInfo)
	for _, m := range mocks {
		info[m.ServiceName] = grpc.ServiceInfo{}
	}
	return info
}

// FindFileByPath satisfies protodesc.Resolver. It delegates well-known types
// to the global registry and synthesises one file per service name.
func (d *dynamicServices) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if fd, err := protoregistry.GlobalFiles.FindFileByPath(path); err == nil {
		return fd, nil
	}
	const prefix, suffix = "mocktool/", ".proto"
	if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
		svcName := path[len(prefix) : len(path)-len(suffix)]
		return d.buildServiceFile(svcName)
	}
	return nil, protoregistry.NotFound
}

// FindDescriptorByName satisfies protodesc.Resolver. It delegates well-known
// types to the global registry and falls back to our synthetic service files.
func (d *dynamicServices) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if desc, err := protoregistry.GlobalFiles.FindDescriptorByName(name); err == nil {
		return desc, nil
	}
	fd, err := d.buildServiceFile(string(name))
	if err != nil {
		return nil, protoregistry.NotFound
	}
	svcs := fd.Services()
	if svcs.Len() == 0 {
		return nil, protoregistry.NotFound
	}
	return svcs.Get(0), nil
}

// buildServiceFile synthesises a FileDescriptor for one service from the
// mock database. All methods use google.protobuf.Struct for input/output so
// that Postman / grpcurl can call them without a hand-written .proto file.
func (d *dynamicServices) buildServiceFile(fullServiceName string) (protoreflect.FileDescriptor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mocks, err := d.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var methods []*descriptorpb.MethodDescriptorProto
	for _, m := range mocks {
		if m.ServiceName == fullServiceName && !seen[m.MethodName] {
			seen[m.MethodName] = true
			mn := m.MethodName
			methods = append(methods, &descriptorpb.MethodDescriptorProto{
				Name:       &mn,
				InputType:  strPtr(".google.protobuf.Struct"),
				OutputType: strPtr(".google.protobuf.Struct"),
			})
		}
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("service %q not found in mock database", fullServiceName)
	}

	pkg, short := "", fullServiceName
	if idx := strings.LastIndex(fullServiceName, "."); idx >= 0 {
		pkg = fullServiceName[:idx]
		short = fullServiceName[idx+1:]
	}

	fdp := &descriptorpb.FileDescriptorProto{
		Name:       strPtr(fmt.Sprintf("mocktool/%s.proto", fullServiceName)),
		Package:    &pkg,
		Dependency: []string{"google/protobuf/struct.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name:   &short,
			Method: methods,
		}},
		Syntax: strPtr("proto3"),
	}

	return protodesc.NewFile(fdp, d)
}

func strPtr(s string) *string { return &s }
