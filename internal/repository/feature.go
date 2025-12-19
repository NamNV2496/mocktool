package repository

import (
	"context"

	"github.com/namnv2496/mocktool/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type IFeatureRepository interface {
	ListAll(ctx context.Context) ([]domain.Feature, error)
	Create(ctx context.Context, f *domain.Feature) error
	Update(ctx context.Context, id int64, update bson.M) error
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

func (_self *FeatureRepository) ListAll(ctx context.Context) ([]domain.Feature, error) {
	var result []domain.Feature
	err := _self.FindMany(ctx, bson.M{}, &result)
	return result, err
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
