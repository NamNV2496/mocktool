package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

type IScenarioRepository interface {
	Create(ctx context.Context, s *domain.Scenario) error
	Update(ctx context.Context, id int64, update bson.M) error
	ListByFeature(ctx context.Context, featureID int64) ([]domain.Scenario, error)
}
type ScenarioRepository struct {
	*BaseRepository
}

func NewScenarioRepository(db *mongo.Database) *ScenarioRepository {
	return &ScenarioRepository{
		BaseRepository: NewBaseRepository(db.Collection("scenarios")),
	}
}

func (r *ScenarioRepository) ListByFeature(
	ctx context.Context,
	featureID int64,
) ([]domain.Scenario, error) {

	var result []domain.Scenario
	err := r.FindMany(ctx, bson.M{
		"feature_id": featureID,
		"is_active":  true,
	}, &result)

	return result, err
}

func (r *ScenarioRepository) Create(ctx context.Context, s *domain.Scenario) error {
	return r.Insert(ctx, s)
}

func (r *ScenarioRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return r.UpdateByID(ctx, id, update)
}
