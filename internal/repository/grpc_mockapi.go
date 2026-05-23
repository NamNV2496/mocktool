package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/repository/$GOFILE.mock.go -package=$GOPACKAGE
type IGRPCMockAPIRepository interface {
	Create(ctx context.Context, m *domain.GRPCMockAPI) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*domain.GRPCMockAPI, error)
	FindByFeatureScenarioServiceMethodAndHash(ctx context.Context, featureName, scenarioName, serviceName, methodName, hashInput string) (*domain.GRPCMockAPI, error)
	ListByFeatureAndScenario(ctx context.Context, featureName, scenarioName string) ([]domain.GRPCMockAPI, error)
	ListAll(ctx context.Context) ([]domain.GRPCMockAPI, error)
	UpdateByID(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteByID(ctx context.Context, id primitive.ObjectID) error
}

type GRPCMockAPIRepository struct {
	repo IBaseRepository
}

func NewGRPCMockAPIRepository(db *mongo.Database) IGRPCMockAPIRepository {
	return &GRPCMockAPIRepository{
		repo: NewBaseRepository(db.Collection("grpc_mock_apis")),
	}
}

func (_self *GRPCMockAPIRepository) Create(ctx context.Context, m *domain.GRPCMockAPI) error {
	m.ID = primitive.NewObjectID()
	m.CreatedAt = time.Now().UTC()
	m.UpdatedAt = m.CreatedAt
	m.IsActive = true
	return _self.repo.Insert(ctx, m)
}

func (_self *GRPCMockAPIRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.GRPCMockAPI, error) {
	var result domain.GRPCMockAPI
	err := _self.repo.FindOne(ctx, bson.M{"_id": id}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (_self *GRPCMockAPIRepository) FindByFeatureScenarioServiceMethodAndHash(
	ctx context.Context,
	featureName, scenarioName, serviceName, methodName, hashInput string,
) (*domain.GRPCMockAPI, error) {
	var result domain.GRPCMockAPI
	err := _self.repo.FindOne(ctx, bson.M{
		"feature_name":  featureName,
		"scenario_name": scenarioName,
		"service_name":  serviceName,
		"method_name":   methodName,
		"hash_input":    hashInput,
		"is_active":     true,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (_self *GRPCMockAPIRepository) ListByFeatureAndScenario(ctx context.Context, featureName, scenarioName string) ([]domain.GRPCMockAPI, error) {
	filter := bson.M{"feature_name": featureName}
	if scenarioName != "" {
		filter["scenario_name"] = scenarioName
	}
	var result []domain.GRPCMockAPI
	err := _self.repo.FindMany(ctx, filter, &result)
	return result, err
}

func (_self *GRPCMockAPIRepository) ListAll(ctx context.Context) ([]domain.GRPCMockAPI, error) {
	var result []domain.GRPCMockAPI
	err := _self.repo.FindMany(ctx, bson.M{"is_active": true}, &result)
	return result, err
}

func (_self *GRPCMockAPIRepository) UpdateByID(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return _self.repo.UpdateByObjectID(ctx, id, update)
}

func (_self *GRPCMockAPIRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	_, err := _self.repo.DeleteOne(ctx, id)
	return err
}
