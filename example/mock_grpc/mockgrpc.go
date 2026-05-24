package mockgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type ExistGrpcClient interface{}

type ClientDeps struct {
	Host         string
	IsMock       bool   // control mock by flag
	MocktoolHost string // when set, mocked methods route through mocktool instead of the real service
	FeatureName  string // sent as "x-feature" metadata to mocktool for scenario selection
}

type IGrpcClient interface {
	ExistFunction(ctx context.Context, req int64) error
	MockGrpcFunction(ctx context.Context, req int64) (int64, error)
}

type Client struct {
	existClient  ExistGrpcClient
	mocktoolConn *grpc.ClientConn
	featureName  string
	isMock       bool
}

func MustNewCoreUniMFClient(
	conf ClientDeps,
) IGrpcClient {
	conn, err := grpc.NewClient(
		conf.Host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	fmt.Println(conn)
	if err != nil {
		panic(fmt.Errorf("create grpc client host: %s : %w", conf.Host, err))
	}
	c := &Client{
		// existClient: pb.NewExistGrpcClient(conn),
		featureName: conf.FeatureName,
		isMock:      conf.IsMock,
	}
	if conf.MocktoolHost != "" {
		mc, err := grpc.NewClient(
			conf.MocktoolHost,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			panic(fmt.Errorf("create mocktool grpc client host: %s : %w", conf.MocktoolHost, err))
		}
		c.mocktoolConn = mc
	}
	return c
}

func (c *Client) ExistFunction(ctx context.Context, req int64) error {
	// _, err := c.existClient.GetUser(timeoutCtx /*&pb.GetUserRequest{}*/)
	// if err != nil {
	// 	return 0, err
	// }
	// return &entity.GetUserResponse{
	// 	Count: result.Count,
	// }, nil
	return nil
}

func (c *Client) MockGrpcFunction(ctx context.Context, req int64) (int64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// mock
	if c.isMock && c.mocktoolConn != nil {
		const fullMethod = "/UserService/GetUser"

		var req proto.Message
		accountId := int64(123456)
		m, err := mocktoolInvoke(timeoutCtx, c.mocktoolConn, accountId, c.featureName, fullMethod, req)
		if err != nil {
			return 0, err
		}
		count, _ := m["count"].(int64)
		return count, nil
	}
	// real
	// _, err := c.existClient.GetUser(timeoutCtx /*&pb.GetUserRequest{}*/)
	// if err != nil {
	// 	return 0, err
	// }
	// return &entity.GetUserResponse{
	// 	Count: result.Count,
	// }, nil
	return 0, nil
}

// mocktoolInvoke converts req to a structpb.Struct, calls fullMethod on the mocktool
// server, and returns the response as a plain map. featureName is forwarded as
// "x-feature" metadata so mocktool selects the right scenario.
func mocktoolInvoke(ctx context.Context, conn *grpc.ClientConn, accountId int64, featureName, fullMethod string, req proto.Message) (map[string]any, error) {
	reqJSON, err := protojson.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mocktool: marshal request: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(reqJSON, &m); err != nil {
		return nil, fmt.Errorf("mocktool: decode request JSON: %w", err)
	}
	reqStruct, err := structpb.NewStruct(m)
	if err != nil {
		return nil, fmt.Errorf("mocktool: build struct: %w", err)
	}
	if featureName != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-feature-name", featureName)
	}
	if accountId != 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-account-id", fmt.Sprint(accountId))
	}
	var respStruct structpb.Struct
	if err := conn.Invoke(ctx, fullMethod, reqStruct, &respStruct); err != nil {
		return nil, fmt.Errorf("mocktool: invoke %s: %w", fullMethod, err)
	}
	return respStruct.AsMap(), nil
}
