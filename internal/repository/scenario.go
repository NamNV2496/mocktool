package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/repository/$GOFILE.mock.go -package=$GOPACKAGE
type IScenarioRepository interface {
	Create(ctx context.Context, s *domain.Scenario) error
	UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error
	GetByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.Scenario, error)
	DeleteByObjectID(ctx context.Context, id primitive.ObjectID) error
	ListByFeatureNamePaginated(ctx context.Context, featureName string, params domain.PaginationParams) ([]domain.Scenario, int64, error)
	SearchByFeatureAndName(ctx context.Context, featureName, query string, params domain.PaginationParams) ([]domain.Scenario, int64, error)
	FindByFeatureNameAndName(ctx context.Context, featureName, name string) (*domain.Scenario, error)
}
type ScenarioRepository struct {
	repo IBaseRepository
}

func NewScenarioRepository(db *mongo.Database) *ScenarioRepository {
	return &ScenarioRepository{
		repo: NewBaseRepository(db.Collection("scenarios")),
	}
}

func (r *ScenarioRepository) ListByFeatureNamePaginated(
	ctx context.Context,
	featureName string,
	params domain.PaginationParams,
) ([]domain.Scenario, int64, error) {
	filter := bson.M{"feature_name": featureName}

	total, err := r.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.Scenario
	err = r.repo.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *ScenarioRepository) SearchByFeatureAndName(
	ctx context.Context,
	featureName string,
	query string,
	params domain.PaginationParams,
) ([]domain.Scenario, int64, error) {
	// Build filter
	filter := bson.M{
		"name": bson.M{
			"$regex":   query,
			"$options": "i", // case-insensitive
		},
	}

	// Add feature filter if provided
	if featureName != "" {
		filter["feature_name"] = featureName
	}

	total, err := r.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.Scenario
	err = r.repo.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *ScenarioRepository) Create(ctx context.Context, s *domain.Scenario) error {
	s.ID = primitive.NewObjectID()
	return r.repo.Insert(ctx, s)
}

func (r *ScenarioRepository) UpdateByObjectID(
	ctx context.Context,
	id primitive.ObjectID,
	update bson.M,
) error {
	return r.repo.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
}

func (r *ScenarioRepository) GetByObjectID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.Scenario, error) {
	var result domain.Scenario
	err := r.repo.FindOne(ctx, bson.M{"_id": id}, &result)
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
	err := r.repo.FindOne(ctx, bson.M{
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
	_, err := r.repo.DeleteOne(ctx, id)
	return err
}

func (r *ScenarioRepository) DeleteByObjectID(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.repo.DeleteOne(ctx, id)
	return err
}

func (r *ScenarioRepository) FindByFeatureNameAndName(ctx context.Context, featureName, name string) (*domain.Scenario, error) {
	var out domain.Scenario
	filter := bson.M{
		"feature_name": featureName,
		"name":         name,
	}
	r.repo.FindOne(ctx, filter, &out)
	return &out, nil
}
