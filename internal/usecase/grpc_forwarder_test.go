package usecase

// import (
// 	"context"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/bson/primitive"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.uber.org/mock/gomock"
// 	"google.golang.org/grpc/codes"

// 	"github.com/namnv2496/mocktool/internal/domain"
// 	mockrepo "github.com/namnv2496/mocktool/mocks/repository"
// )

// func newGRPCForwardUC(t *testing.T) (
// 	IGRPCForwardUC,
// 	*mockrepo.MockIGRPCMockAPIRepository,
// 	*mockrepo.MockIScenarioRepository,
// 	*mockrepo.MockIAccountScenarioRepository,
// ) {
// 	ctrl := gomock.NewController(t)
// 	grpcRepo := mockrepo.NewMockIGRPCMockAPIRepository(ctrl)
// 	scenarioRepo := mockrepo.NewMockIScenarioRepository(ctrl)
// 	accountScenarioRepo := mockrepo.NewMockIAccountScenarioRepository(ctrl)
// 	uc := NewGRPCForwardUC(grpcRepo, scenarioRepo, accountScenarioRepo)
// 	return uc, grpcRepo, scenarioRepo, accountScenarioRepo
// }

// var (
// 	testScenarioID  = primitive.NewObjectID()
// 	testAccountID   = "acc-1"
// 	testFeatureName = "my-feature"
// 	testScenarioObj = &domain.Scenario{Name: "scenario-1"}
// )

// func setupScenarioMocks(
// 	accountScenarioRepo *mockrepo.MockIAccountScenarioRepository,
// 	scenarioRepo *mockrepo.MockIScenarioRepository,
// ) {
// 	accountScenarioRepo.EXPECT().
// 		GetActiveScenario(gomock.Any(), testFeatureName, gomock.Any()).
// 		Return(&domain.AccountScenario{ScenarioID: testScenarioID}, nil)
// 	scenarioRepo.EXPECT().
// 		GetByObjectID(gomock.Any(), testScenarioID).
// 		Return(testScenarioObj, nil)
// }

// func outputBSON(t *testing.T, m map[string]any) bson.Raw {
// 	t.Helper()
// 	raw, err := bson.Marshal(m)
// 	require.NoError(t, err)
// 	return raw
// }

// func TestHandleCall_HitExactHash(t *testing.T) {
// 	uc, grpcRepo, scenarioRepo, accountScenarioRepo := newGRPCForwardUC(t)
// 	setupScenarioMocks(accountScenarioRepo, scenarioRepo)

// 	reqBytes := []byte(`{"id":1}`)
// 	expectedHash := hashBytes(reqBytes)
// 	mock := &domain.GRPCMockAPI{
// 		ServiceName:  "com.example.UserService",
// 		MethodName:   "GetUser",
// 		Output:       outputBSON(t, map[string]any{"name": "Alice"}),
// 		StatusCode:   0,
// 		Latency:      0,
// 	}
// 	grpcRepo.EXPECT().
// 		FindByFeatureScenarioServiceMethodAndHash(gomock.Any(), testFeatureName, "scenario-1", "com.example.UserService", "GetUser", expectedHash).
// 		Return(mock, nil)

// 	aid := testAccountID
// 	result, code, err := uc.HandleCall(context.Background(), "/com.example.UserService/GetUser", reqBytes, testFeatureName, &aid)

// 	require.NoError(t, err)
// 	assert.Equal(t, codes.OK, code)
// 	assert.NotNil(t, result)
// 	assert.Equal(t, "Alice", result.Fields["name"].GetStringValue())
// }

// func TestHandleCall_FallbackToEmptyHash(t *testing.T) {
// 	uc, grpcRepo, scenarioRepo, accountScenarioRepo := newGRPCForwardUC(t)
// 	setupScenarioMocks(accountScenarioRepo, scenarioRepo)

// 	reqBytes := []byte(`{"id":99}`)
// 	computedHash := hashBytes(reqBytes)
// 	mock := &domain.GRPCMockAPI{
// 		ServiceName: "svc",
// 		MethodName:  "Method",
// 		Output:      outputBSON(t, map[string]any{"ok": true}),
// 	}
// 	grpcRepo.EXPECT().
// 		FindByFeatureScenarioServiceMethodAndHash(gomock.Any(), testFeatureName, "scenario-1", "svc", "Method", computedHash).
// 		Return(nil, mongo.ErrNoDocuments)
// 	grpcRepo.EXPECT().
// 		FindByFeatureScenarioServiceMethodAndHash(gomock.Any(), testFeatureName, "scenario-1", "svc", "Method", "").
// 		Return(mock, nil)

// 	aid := testAccountID
// 	result, code, err := uc.HandleCall(context.Background(), "/svc/Method", reqBytes, testFeatureName, &aid)

// 	require.NoError(t, err)
// 	assert.Equal(t, codes.OK, code)
// 	assert.True(t, result.Fields["ok"].GetBoolValue())
// }

// func TestHandleCall_NotFound(t *testing.T) {
// 	uc, grpcRepo, scenarioRepo, accountScenarioRepo := newGRPCForwardUC(t)
// 	setupScenarioMocks(accountScenarioRepo, scenarioRepo)

// 	grpcRepo.EXPECT().
// 		FindByFeatureScenarioServiceMethodAndHash(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
// 		Return(nil, mongo.ErrNoDocuments).Times(2)

// 	aid := testAccountID
// 	_, code, err := uc.HandleCall(context.Background(), "/svc/Missing", []byte(`{}`), testFeatureName, &aid)

// 	assert.Error(t, err)
// 	assert.Equal(t, codes.NotFound, code)
// }

// func TestHandleCall_StatusCodeOverride(t *testing.T) {
// 	uc, grpcRepo, scenarioRepo, accountScenarioRepo := newGRPCForwardUC(t)
// 	setupScenarioMocks(accountScenarioRepo, scenarioRepo)

// 	mock := &domain.GRPCMockAPI{
// 		ServiceName: "svc",
// 		MethodName:  "Fail",
// 		Output:      outputBSON(t, map[string]any{}),
// 		StatusCode:  int32(codes.PermissionDenied),
// 	}
// 	grpcRepo.EXPECT().
// 		FindByFeatureScenarioServiceMethodAndHash(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
// 		Return(mock, nil)

// 	aid := testAccountID
// 	_, code, err := uc.HandleCall(context.Background(), "/svc/Fail", nil, testFeatureName, &aid)

// 	assert.Error(t, err)
// 	assert.Equal(t, codes.PermissionDenied, code)
// }

// func TestSplitFullMethod(t *testing.T) {
// 	tests := []struct {
// 		input   string
// 		svc     string
// 		method  string
// 	}{
// 		{"/com.example.UserService/GetUser", "com.example.UserService", "GetUser"},
// 		{"/svc/Method", "svc", "Method"},
// 		{"NoSlashAtAll", "NoSlashAtAll", ""},
// 	}
// 	for _, tt := range tests {
// 		svc, method := splitFullMethod(tt.input)
// 		assert.Equal(t, tt.svc, svc, "input: %s", tt.input)
// 		assert.Equal(t, tt.method, method, "input: %s", tt.input)
// 	}
// }
