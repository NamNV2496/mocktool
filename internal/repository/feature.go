package repository

import (
	"context"

	"github.com/namnv2496/mocktool/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
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

func (r *FeatureRepository) ListAll(ctx context.Context) ([]domain.Feature, error) {
	var result []domain.Feature
	err := r.FindMany(ctx, bson.M{}, &result)
	return result, err
}

func (r *FeatureRepository) ListActive(ctx context.Context) ([]domain.Feature, error) {
	var result []domain.Feature
	err := r.FindMany(ctx, bson.M{"is_active": true}, &result)
	return result, err
}

func (r *FeatureRepository) Create(ctx context.Context, f *domain.Feature) error {
	return r.Insert(ctx, f)
}

func (r *FeatureRepository) Update(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	return r.UpdateByID(ctx, id, update)
}
