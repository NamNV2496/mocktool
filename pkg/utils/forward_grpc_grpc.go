package utils

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

/*
GRPCCallOption wraps the target RPC invocation so ForwardGRPCToGRPC
can stay generic without depending on generated client code.
*/
type GRPCCallOption struct {
	// Invoke calls the upstream gRPC method and returns the trailer metadata on error.
	Invoke func(ctx context.Context) (metadata.MD, error)
}

/*
ForwardGRPCToGRPC executes a gRPC call via the provided GRPCCallOption and
propagates any trailer metadata back to the caller's context on failure.
This is the equivalent of the callGrpc pattern in example/grpc/main.go.
*/
func ForwardGRPCToGRPC(ctx context.Context, call GRPCCallOption) error {
	md, err := call.Invoke(ctx)
	if err != nil {
		if len(md) > 0 {
			grpc.SetTrailer(ctx, md)
		}
		return fmt.Errorf("forward grpc-grpc: %w", err)
	}
	return nil
}

/*
NewGRPCCallOption is a helper that builds a GRPCCallOption from a typed
unary RPC call. The caller provides the actual client method invocation
and grpc.Trailer is injected automatically.
*/
func NewGRPCCallOption(invoke func(ctx context.Context, trailer *metadata.MD) error) GRPCCallOption {
	return GRPCCallOption{
		Invoke: func(ctx context.Context) (metadata.MD, error) {
			var md metadata.MD
			err := invoke(ctx, &md)
			return md, err
		},
	}
}
