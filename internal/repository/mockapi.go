package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/repository/$GOFILE.mock.go -package=$GOPACKAGE
type IMockAPIRepository interface {
	ListAllActiveAPIs(ctx context.Context) ([]domain.MockAPI, error)
	ListActiveAPIsByScenario(ctx context.Context, scenarios []string) ([]domain.MockAPI, error)
	ListByScenarioNamePaginated(ctx context.Context, scenarioName string, params domain.PaginationParams) ([]domain.MockAPI, int64, error)
	SearchByScenarioAndNameOrPath(ctx context.Context, scenarioName, query string, params domain.PaginationParams) ([]domain.MockAPI, int64, error)
	Create(ctx context.Context, m *domain.MockAPI) error
	FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.MockAPI, error)
	FindByName(ctx context.Context, name string) (*domain.MockAPI, error)
	UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeletByObjectID(ctx context.Context, id primitive.ObjectID) error
	FindByFeatureScenarioPathMethodAndHash(ctx context.Context, featureName, scenarioName, path, method, hashInput string) (*domain.MockAPI, error)
}
type MockAPIRepository struct {
	repo IBaseRepository
}

func NewMockAPIRepository(db *mongo.Database) IMockAPIRepository {
	return &MockAPIRepository{
		repo: NewBaseRepository(db.Collection("mock_apis")),
	}
}

/* ---------- list ---------- */

func (_self *MockAPIRepository) ListActiveAPIsByScenario(ctx context.Context, scenarios []string) ([]domain.MockAPI, error) {
	var result []domain.MockAPI
	err := _self.repo.FindMany(ctx, bson.M{
		"scenario_name": bson.M{"$in": scenarios},
		"is_active":     true,
	}, &result)

	return result, err
}

func (_self *MockAPIRepository) ListAllActiveAPIs(ctx context.Context) ([]domain.MockAPI, error) {
	var result []domain.MockAPI
	err := _self.repo.FindMany(ctx, bson.M{
		"is_active": true,
	}, &result)

	return result, err
}

func (_self *MockAPIRepository) ListByScenario(
	ctx context.Context,
	scenarioID int64,
) ([]domain.MockAPI, error) {

	var result []domain.MockAPI
	err := _self.repo.FindMany(ctx, bson.M{
		"scenario_name": scenarioID,
	}, &result)

	return result, err
}

func (_self *MockAPIRepository) ListByScenarioName(
	ctx context.Context,
	scenarioName string,
) ([]domain.MockAPI, error) {

	var result []domain.MockAPI
	err := _self.repo.FindMany(ctx, bson.M{
		"scenario_name": scenarioName,
		// "is_active":     true,
	}, &result)

	return result, err
}

func (_self *MockAPIRepository) ListByScenarioNamePaginated(
	ctx context.Context,
	scenarioName string,
	params domain.PaginationParams,
) ([]domain.MockAPI, int64, error) {
	filter := bson.M{
		"scenario_name": scenarioName,
		// "is_active":     true,
	}

	total, err := _self.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.MockAPI
	err = _self.repo.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (_self *MockAPIRepository) SearchByScenarioAndNameOrPath(
	ctx context.Context,
	scenarioName string,
	query string,
	params domain.PaginationParams,
) ([]domain.MockAPI, int64, error) {
	// Build filter with scenario name and search query for name or path
	filter := bson.M{
		"scenario_name": scenarioName,
		"is_active":     true,
	}

	// Add search condition for name or path (case-insensitive)
	if query != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{
				"$regex":   query,
				"$options": "i", // case-insensitive
			}},
			{"path": bson.M{
				"$regex":   query,
				"$options": "i", // case-insensitive
			}},
		}
	}

	total, err := _self.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.MockAPI
	err = _self.repo.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

/* ---------- create ---------- */

func (_self *MockAPIRepository) Create(ctx context.Context, m *domain.MockAPI) error {
	m.ID = primitive.NewObjectID()
	return _self.repo.Insert(ctx, m)
}

/* ---------- update ---------- */

func (_self *MockAPIRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return _self.repo.UpdateByID(ctx, id, update)
}

/* ---------- execute ---------- */

func (_self *MockAPIRepository) FindByPathAndHash(
	ctx context.Context,
	path string,
	hash string,
) (*domain.MockAPI, error) {

	var result domain.MockAPI
	err := _self.repo.FindOne(ctx, bson.M{
		"path":      path,
		"hashcode":  hash,
		"is_active": true,
	}, &result)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (_self *MockAPIRepository) FindByFeatureScenarioPathMethodAndHash(
	ctx context.Context,
	featureName, scenarioName, path, method, hashInput string,
) (*domain.MockAPI, error) {

	var result domain.MockAPI
	filter := bson.M{
		"feature_name":  featureName,
		"scenario_name": scenarioName,
		"path":          path,
		"method":        method,
		"hash_input":    hashInput,
		"is_active":     true,
	}

	err := _self.repo.FindOne(ctx, filter, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (_self *MockAPIRepository) DeletByObjectID(ctx context.Context, id primitive.ObjectID) error {
	_, err := _self.repo.DeleteOne(ctx, id)
	return err
}

func (_self *MockAPIRepository) FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.MockAPI, error) {
	var output domain.MockAPI
	filter := bson.M{
		"_id": id,
	}
	_self.repo.FindOne(ctx, filter, &output)
	return &output, nil
}

func (_self *MockAPIRepository) FindByName(ctx context.Context, name string) (*domain.MockAPI, error) {
	var output domain.MockAPI
	filter := bson.M{
		"name": name,
	}
	_self.repo.FindOne(ctx, filter, &output)
	return &output, nil
}

func (_self *MockAPIRepository) UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return _self.repo.UpdateByObjectID(ctx, id, update)
}
