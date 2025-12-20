package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

type IScenarioRepository interface {
	Create(ctx context.Context, s *domain.Scenario) error
	UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error
	UpdateByFilter(ctx context.Context, filter bson.M, update bson.M) error
	GetByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.Scenario, error)
	ListByFeatureName(ctx context.Context, featureName string) ([]domain.Scenario, error)
	GetActiveScenarioByFeatureName(ctx context.Context, featureName string) (*domain.Scenario, error)
	GetActiveScenarios(ctx context.Context) ([]string, error)
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

func (r *ScenarioRepository) ListByFeatureName(
	ctx context.Context,
	featureName string,
) ([]domain.Scenario, error) {
	var result []domain.Scenario
	err := r.FindMany(ctx, bson.M{
		"feature_name": featureName,
	}, &result)
	return result, err
}

func (r *ScenarioRepository) Create(ctx context.Context, s *domain.Scenario) error {
	s.ID = primitive.NewObjectID()
	return r.Insert(ctx, s)
}

func (r *ScenarioRepository) UpdateByObjectID(
	ctx context.Context,
	id primitive.ObjectID,
	update bson.M,
) error {
	return r.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
}

func (r *ScenarioRepository) UpdateByFilter(
	ctx context.Context,
	filter bson.M,
	update bson.M,
) error {
	return r.UpdateMany(ctx, filter, bson.M{"$set": update})
}

func (r *ScenarioRepository) GetByObjectID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.Scenario, error) {
	var result domain.Scenario
	err := r.FindOne(ctx, bson.M{"_id": id}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *ScenarioRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return r.UpdateByID(ctx, id, update)
}

func (r *ScenarioRepository) GetActiveScenarioByFeatureName(ctx context.Context, featureName string) (*domain.Scenario, error) {
	var result domain.Scenario
	err := r.FindOne(ctx, bson.M{
		"feature_name": featureName,
		"is_active":    true,
	}, &result)

	return &result, err
}

func (r *ScenarioRepository) GetActiveScenarios(ctx context.Context) ([]string, error) {
	var results []domain.Scenario
	err := r.FindMany(ctx, bson.M{
		"is_active": true,
	}, &results)

	if err != nil {
		return nil, err
	}

	scenarios := make([]string, len(results))
	for i, scenario := range results {
		scenarios[i] = scenario.Name
	}

	return scenarios, nil
}
