package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

type IMockAPIRepository interface {
	ListAllActiveAPIs(ctx context.Context) ([]domain.MockAPI, error)
	ListByScenarioName(ctx context.Context, scenarioName string) ([]domain.MockAPI, error)
	Create(ctx context.Context, m *domain.MockAPI) error
	UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error
}
type MockAPIRepository struct {
	*BaseRepository
}

func NewMockAPIRepository(db *mongo.Database) *MockAPIRepository {
	return &MockAPIRepository{
		BaseRepository: NewBaseRepository(db.Collection("mock_apis")),
	}
}

/* ---------- list ---------- */

func (_self *MockAPIRepository) ListActiveAPIsByScenario(ctx context.Context, scenarios []string) ([]domain.MockAPI, error) {
	var result []domain.MockAPI
	err := _self.FindMany(ctx, bson.M{
		"scenario_name": bson.M{"$in": scenarios},
		"is_active":     true,
	}, &result)

	return result, err
}

func (_self *MockAPIRepository) ListAllActiveAPIs(ctx context.Context) ([]domain.MockAPI, error) {
	var result []domain.MockAPI
	err := _self.FindMany(ctx, bson.M{
		"is_active": true,
	}, &result)

	return result, err
}

func (_self *MockAPIRepository) ListByScenario(
	ctx context.Context,
	scenarioID int64,
) ([]domain.MockAPI, error) {

	var result []domain.MockAPI
	err := _self.FindMany(ctx, bson.M{
		"scenario_name": scenarioID,
	}, &result)

	return result, err
}

func (_self *MockAPIRepository) ListByScenarioName(
	ctx context.Context,
	scenarioName string,
) ([]domain.MockAPI, error) {

	var result []domain.MockAPI
	err := _self.FindMany(ctx, bson.M{
		"scenario_name": scenarioName,
		"is_active":     true,
	}, &result)

	return result, err
}

/* ---------- create ---------- */

func (_self *MockAPIRepository) Create(ctx context.Context, m *domain.MockAPI) error {
	m.ID = primitive.NewObjectID()
	return _self.Insert(ctx, m)
}

/* ---------- update ---------- */

func (_self *MockAPIRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return _self.UpdateByID(ctx, id, update)
}

/* ---------- execute ---------- */

func (_self *MockAPIRepository) FindByPathAndHash(
	ctx context.Context,
	path string,
	hash string,
) (*domain.MockAPI, error) {

	var result domain.MockAPI
	err := _self.col.FindOne(ctx, bson.M{
		"path":      path,
		"hashcode":  hash,
		"is_active": true,
	}).Decode(&result)

	if err != nil {
		return nil, err
	}

	return &result, nil
}
