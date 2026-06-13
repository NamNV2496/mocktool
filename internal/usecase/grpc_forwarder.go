package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/namnv2496/mocktool/internal/repository"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/usecase/$GOFILE.mock.go -package=$GOPACKAGE
type IGRPCForwardUC interface {
	HandleCall(ctx context.Context, fullMethod string, reqBytes []byte, featureName string, scenario string) (*structpb.Struct, codes.Code, error)
}

type GRPCForwardUC struct {
	grpcMockRepo        repository.IGRPCMockAPIRepository
	scenarioRepo        repository.IScenarioRepository
	accountScenarioRepo repository.IAccountScenarioRepository
}

func NewGRPCForwardUC(
	grpcMockRepo repository.IGRPCMockAPIRepository,
	scenarioRepo repository.IScenarioRepository,
	accountScenarioRepo repository.IAccountScenarioRepository,
) IGRPCForwardUC {
	return &GRPCForwardUC{
		grpcMockRepo:        grpcMockRepo,
		scenarioRepo:        scenarioRepo,
		accountScenarioRepo: accountScenarioRepo,
	}
}

func (_self *GRPCForwardUC) HandleCall(
	ctx context.Context,
	fullMethod string,
	reqBytes []byte,
	featureName string,
	scenario string,
) (*structpb.Struct, codes.Code, error) {
	serviceName, methodName := splitFullMethod(fullMethod)

	// Hash request for body-based matching.
	// Decode proto bytes as Struct → canonical JSON → sha256 so the hash
	// matches what the admin UI stores via utils.GenerateHashFromInput.
	hashInput := hashStructProto(reqBytes)

	// Exact match first; fall back to match-all (empty hash) if needed
	mock, err := _self.grpcMockRepo.FindByFeatureScenarioServiceMethodAndHash(
		ctx, featureName, scenario, serviceName, methodName, hashInput,
	)
	if err == mongo.ErrNoDocuments && hashInput != "" {
		mock, err = _self.grpcMockRepo.FindByFeatureScenarioServiceMethodAndHash(
			ctx, featureName, scenario, serviceName, methodName, "",
		)
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, codes.NotFound, status.Errorf(codes.NotFound, "mock not found for %s/%s", serviceName, methodName)
		}
		return nil, codes.Internal, status.Errorf(codes.Internal, "lookup: %v", err)
	}

	if mock.Latency > 0 {
		time.Sleep(time.Duration(mock.Latency) * time.Millisecond)
	}

	// Transcode bson.Raw → map → structpb.Struct
	result, err := bsonRawToStruct(mock.Output)
	if err != nil {
		return nil, codes.Internal, status.Errorf(codes.Internal, "transcode output: %v", err)
	}

	grpcCode := codes.Code(mock.StatusCode)
	if grpcCode != codes.OK {
		return nil, grpcCode, status.Error(grpcCode, "mock status")
	}

	return result, codes.OK, nil
}

func splitFullMethod(fullMethod string) (serviceName, methodName string) {
	// fullMethod format: /package.ServiceName/MethodName
	trimmed := strings.TrimPrefix(fullMethod, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx < 0 {
		return trimmed, ""
	}
	return trimmed[:idx], trimmed[idx+1:]
}

// hashStructProto decodes proto bytes as google.protobuf.Struct, serialises the
// resulting map to canonical JSON (sorted keys), and returns the SHA-256 hex
// digest. This matches utils.GenerateHashFromInput used on the admin write path,
// so body-based mock lookup works end-to-end.
func hashStructProto(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var s structpb.Struct
	if err := proto.Unmarshal(b, &s); err != nil {
		sum := sha256.Sum256(b)
		return hex.EncodeToString(sum[:])
	}
	m := s.AsMap()
	if len(m) == 0 {
		return ""
	}
	canonical, err := json.Marshal(m)
	if err != nil {
		sum := sha256.Sum256(b)
		return hex.EncodeToString(sum[:])
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

func bsonRawToStruct(raw bson.Raw) (*structpb.Struct, error) {
	jsonBytes, err := bson.MarshalExtJSON(raw, false, false)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err2 := json.Unmarshal(jsonBytes, &m); err2 != nil {
		return nil, err2
	}

	return structpb.NewStruct(m)
}
