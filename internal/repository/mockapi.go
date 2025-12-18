package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

type IMockAPIRepository interface {
	ListByScenario(ctx context.Context, scenarioID int64) ([]domain.MockAPI, error)
	Create(ctx context.Context, m *domain.MockAPI) error
	FindByPathAndHash(ctx context.Context, path string, hash string) (*domain.MockAPI, error)
	Update(ctx context.Context, id int64, update bson.M) error
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

func (r *MockAPIRepository) ListByScenario(
	ctx context.Context,
	scenarioID int64,
) ([]domain.MockAPI, error) {

	var result []domain.MockAPI
	err := r.FindMany(ctx, bson.M{
		"scenario_id": scenarioID,
		"is_active":   true,
	}, &result)

	return result, err
}

/* ---------- create ---------- */

func (r *MockAPIRepository) Create(ctx context.Context, m *domain.MockAPI) error {
	return r.Insert(ctx, m)
}

/* ---------- update ---------- */

func (r *MockAPIRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return r.UpdateByID(ctx, id, update)
}

/* ---------- execute ---------- */

func (r *MockAPIRepository) FindByPathAndHash(
	ctx context.Context,
	path string,
	hash string,
) (*domain.MockAPI, error) {

	var result domain.MockAPI
	err := r.col.FindOne(ctx, bson.M{
		"path":      path,
		"hashcode":  hash,
		"is_active": true,
	}).Decode(&result)

	if err != nil {
		return nil, err
	}

	return &result, nil
}
