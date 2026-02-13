package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/namnv2496/mocktool/internal/domain"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/repository/$GOFILE.mock.go -package=$GOPACKAGE
type IAccountScenarioRepository interface {
	Create(ctx context.Context, as *domain.AccountScenario) error
	GetActiveScenario(ctx context.Context, featureName string, accountId *string) (*domain.AccountScenario, error)
	DeactivateByFeatureAndAccount(ctx context.Context, featureName string, accountId *string) error
	DeactivateAllAccountSpecificMappings(ctx context.Context, featureName string) error
}

type AccountScenarioRepository struct {
	*BaseRepository
}

func NewAccountScenarioRepository(db *mongo.Database) *AccountScenarioRepository {
	return &AccountScenarioRepository{
		BaseRepository: NewBaseRepository(db.Collection("account_scenarios")),
	}
}

func (r *AccountScenarioRepository) Create(ctx context.Context, as *domain.AccountScenario) error {
	as.ID = primitive.NewObjectID()
	return r.Insert(ctx, as)
}

func (r *AccountScenarioRepository) UpdateByObjectID(
	ctx context.Context,
	id primitive.ObjectID,
	update bson.M,
) error {
	return r.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
}

func (r *AccountScenarioRepository) GetByObjectID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.AccountScenario, error) {
	var result domain.AccountScenario
	err := r.FindOne(ctx, bson.M{"_id": id}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetActiveScenario returns the active scenario for a feature and accountId
// If accountId is provided, it first looks for account-specific mapping, then falls back to global
func (r *AccountScenarioRepository) GetActiveScenario(
	ctx context.Context,
	featureName string,
	accountId *string,
) (*domain.AccountScenario, error) {
	var result domain.AccountScenario

	// If accountId is provided, try to find account-specific mapping
	if accountId != nil {
		err := r.FindOne(ctx, bson.M{
			"feature_name": featureName,
			"account_id":   *accountId,
		}, &result)

		// If found, return it
		if err == nil {
			return &result, nil
		}
	}

	// Fallback to global mapping (account_id is null)
	err := r.FindOne(ctx, bson.M{
		"feature_name": featureName,
		"account_id":   nil,
	}, &result)

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeactivateByFeatureAndAccount deletes the active scenario mapping for a feature and account
func (r *AccountScenarioRepository) DeactivateByFeatureAndAccount(
	ctx context.Context,
	featureName string,
	accountId *string,
) error {
	filter := bson.M{"feature_name": featureName}
	if accountId != nil {
		filter["account_id"] = *accountId
	} else {
		filter["account_id"] = nil
	}

	_, err := r.BaseRepository.col.DeleteMany(ctx, filter)
	return err
}

// DeactivateAllAccountSpecificMappings deletes all account-specific mappings for a feature
// (keeps only the global mapping with account_id = nil)
func (r *AccountScenarioRepository) DeactivateAllAccountSpecificMappings(
	ctx context.Context,
	featureName string,
) error {
	filter := bson.M{
		"feature_name": featureName,
		"account_id":   bson.M{"$ne": nil}, // Delete all where account_id is NOT nil
	}

	_, err := r.BaseRepository.col.DeleteMany(ctx, filter)
	return err
}

func (r *AccountScenarioRepository) Delete(
	ctx context.Context,
	id primitive.ObjectID,
) error {
	_, err := r.BaseRepository.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *AccountScenarioRepository) ListByFeature(
	ctx context.Context,
	featureName string,
) ([]domain.AccountScenario, error) {
	var results []domain.AccountScenario
	err := r.FindMany(ctx, bson.M{"feature_name": featureName}, &results)
	return results, err
}
