package repository

import (
	"context"

	"github.com/namnv2496/mocktool/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type IFeatureRepository interface {
	ListAllPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Feature, int64, error)
	SearchByName(ctx context.Context, query string, params domain.PaginationParams) ([]domain.Feature, int64, error)
	Create(ctx context.Context, f *domain.Feature) error
	UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error
}
type FeatureRepository struct {
	*BaseRepository
}

func NewFeatureRepository(db *mongo.Database) IFeatureRepository {
	return &FeatureRepository{
		BaseRepository: NewBaseRepository(db.Collection("features")),
	}
}

/* ---------- queries ---------- */
func (_self *FeatureRepository) ListAllPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Feature, int64, error) {
	filter := bson.M{}

	total, err := _self.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.Feature
	err = _self.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
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

	total, err := _self.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.Feature
	err = _self.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (_self *FeatureRepository) ListActive(ctx context.Context) ([]domain.Feature, error) {
	var result []domain.Feature
	err := _self.FindMany(ctx, bson.M{"is_active": true}, &result)
	return result, err
}

func (_self *FeatureRepository) Create(ctx context.Context, f *domain.Feature) error {
	f.ID = primitive.NewObjectID()
	return _self.Insert(ctx, f)
}

func (_self *FeatureRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return _self.UpdateByID(ctx, id, update)
}
