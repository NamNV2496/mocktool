package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

type ILoadTestScenarioRepository interface {
	Create(ctx context.Context, s *domain.LoadTestScenario) error
	Update(ctx context.Context, id primitive.ObjectID, update bson.M) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LoadTestScenario, error)
	GetByName(ctx context.Context, name string) (*domain.LoadTestScenario, error)
	ListPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.LoadTestScenario, int64, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type LoadTestScenarioRepository struct {
	*BaseRepository
}

func NewLoadTestScenarioRepository(db *mongo.Database) *LoadTestScenarioRepository {
	return &LoadTestScenarioRepository{
		BaseRepository: NewBaseRepository(db.Collection("loadtest_scenarios")),
	}
}

func (r *LoadTestScenarioRepository) Create(ctx context.Context, s *domain.LoadTestScenario) error {
	s.ID = primitive.NewObjectID()
	s.CreatedAt = time.Now().UTC()
	s.UpdatedAt = time.Now().UTC()
	return r.Insert(ctx, s)
}

func (r *LoadTestScenarioRepository) Update(
	ctx context.Context,
	id primitive.ObjectID,
	update bson.M,
) error {
	update["updated_at"] = time.Now().UTC()
	return r.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
}

func (r *LoadTestScenarioRepository) GetByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.LoadTestScenario, error) {
	var result domain.LoadTestScenario
	err := r.FindOne(ctx, bson.M{"_id": id}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *LoadTestScenarioRepository) GetByName(
	ctx context.Context,
	name string,
) (*domain.LoadTestScenario, error) {
	var result domain.LoadTestScenario
	err := r.FindOne(ctx, bson.M{"name": name}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *LoadTestScenarioRepository) ListPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.LoadTestScenario, int64, error) {
	filter := bson.M{}

	total, err := r.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var result []domain.LoadTestScenario
	err = r.FindManyWithPagination(ctx, filter, params.Skip(), params.Limit(), &result)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *LoadTestScenarioRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
