package repository

import (
	"context"

	"github.com/namnv2496/mocktool/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/repository/$GOFILE.mock.go -package=$GOPACKAGE
type IFeatureRepository interface {
	ListAllPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Feature, int64, error)
	SearchByName(ctx context.Context, query string, params domain.PaginationParams) ([]domain.Feature, int64, error)
	Create(ctx context.Context, f *domain.Feature) error
	UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteById(ctx context.Context, featureId primitive.ObjectID) error
	FindById(ctx context.Context, featureId primitive.ObjectID) (*domain.Feature, error)
	FindByName(ctx context.Context, name string) (*domain.Feature, error)
}
type FeatureRepository struct {
	repo IBaseRepository
}

func NewFeatureRepository(db *mongo.Database) IFeatureRepository {
	return &FeatureRepository{
		repo: NewBaseRepository(db.Collection("features")),
	}
}

/* ---------- queries ---------- */
func (_self *FeatureRepository) ListAllPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Feature, int64, error) {
	filter := bson.M{}

	total, err := _self.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.Feature
	err = _self.repo.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (_self *FeatureRepository) SearchByName(ctx context.Context, query string, params domain.PaginationParams) ([]domain.Feature, int64, error) {
	// Use MongoDB regex for case-insensitive search
	filter := bson.M{
		"name": bson.M{
			"$regex":   query,
			"$options": "i", // case-insensitive
		},
	}

	total, err := _self.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.Feature
	err = _self.repo.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (_self *FeatureRepository) ListActive(ctx context.Context) ([]domain.Feature, error) {
	var result []domain.Feature
	err := _self.repo.FindMany(ctx, bson.M{"is_active": true}, &result)
	return result, err
}

func (_self *FeatureRepository) Create(ctx context.Context, f *domain.Feature) error {
	f.ID = primitive.NewObjectID()
	return _self.repo.Insert(ctx, f)
}

func (_self *FeatureRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return _self.repo.UpdateByID(ctx, id, update)
}

func (_self *FeatureRepository) DeleteById(ctx context.Context, featureId primitive.ObjectID) error {
	_, err := _self.repo.DeleteOne(ctx, featureId)
	return err
}

func (_self *FeatureRepository) FindById(ctx context.Context, featureId primitive.ObjectID) (*domain.Feature, error) {
	var out domain.Feature
	filter := bson.M{
		"_id": featureId,
	}
	_self.repo.FindOne(ctx, filter, &out)
	return &out, nil
}

func (_self *FeatureRepository) UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return _self.repo.UpdateByObjectID(ctx, id, update)
}

func (_self *FeatureRepository) FindByName(ctx context.Context, name string) (*domain.Feature, error) {
	var out domain.Feature
	filter := bson.M{
		"name": name,
	}
	_self.repo.FindOne(ctx, filter, &out)
	return &out, nil
}
