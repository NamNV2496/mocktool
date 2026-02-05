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
	GetByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.Scenario, error)
	ListByFeatureNamePaginated(ctx context.Context, featureName string, params domain.PaginationParams) ([]domain.Scenario, int64, error)
}
type ScenarioRepository struct {
	*BaseRepository
}

func NewScenarioRepository(db *mongo.Database) *ScenarioRepository {
	return &ScenarioRepository{
		BaseRepository: NewBaseRepository(db.Collection("scenarios")),
	}
}

func (r *ScenarioRepository) ListByFeatureNamePaginated(
	ctx context.Context,
	featureName string,
	params domain.PaginationParams,
) ([]domain.Scenario, int64, error) {
	filter := bson.M{"feature_name": featureName}

	total, err := r.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.Scenario
	err = r.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
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

func (r *ScenarioRepository) GetByName(
	ctx context.Context,
	featureName string,
	scenarioName string,
) (*domain.Scenario, error) {
	var result domain.Scenario
	err := r.FindOne(ctx, bson.M{
		"feature_name": featureName,
		"name":         scenarioName,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *ScenarioRepository) Delete(
	ctx context.Context,
	id primitive.ObjectID,
) error {
	_, err := r.BaseRepository.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
