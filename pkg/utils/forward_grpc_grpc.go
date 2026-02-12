package utils

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type GRPCCallOption struct {
	Invoke func(ctx context.Context) (metadata.MD, error)
}

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

func NewGRPCCallOption(invoke func(ctx context.Context, trailer *metadata.MD) error) GRPCCallOption {
	return GRPCCallOption{
		Invoke: func(ctx context.Context) (metadata.MD, error) {
			var md metadata.MD
			err := invoke(ctx, &md)
			return md, err
		},
	}
}
