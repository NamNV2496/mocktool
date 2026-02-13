package repository

import (
	"testing"

	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestAccountScenarioRepository_Create(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	accountID := "user123"
	accountScenario := &domain.AccountScenario{
		FeatureName: "test-feature",
		ScenarioID:  primitive.NewObjectID(),
		AccountId:   &accountID,
	}

	err := repo.Create(ctx, accountScenario)
	require.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, accountScenario.ID)
}

func TestAccountScenarioRepository_GetActiveScenario(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	scenarioID1 := primitive.NewObjectID()
	scenarioID2 := primitive.NewObjectID()
	globalScenarioID := primitive.NewObjectID()

	accountID1 := "user123"
	accountID2 := "user456"

	// Create account-specific scenarios
	accountScenario1 := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  scenarioID1,
		AccountId:   &accountID1,
	}
	accountScenario2 := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  scenarioID2,
		AccountId:   &accountID2,
	}
	// Create global scenario (accountID = nil)
	globalScenario := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  globalScenarioID,
		AccountId:   nil,
	}

	err := repo.Create(ctx, accountScenario1)
	require.NoError(t, err)
	err = repo.Create(ctx, accountScenario2)
	require.NoError(t, err)
	err = repo.Create(ctx, globalScenario)
	require.NoError(t, err)

	tests := []struct {
		name               string
		featureName        string
		accountID          *string
		expectedScenarioID primitive.ObjectID
	}{
		{
			name:               "get account-specific scenario for user123",
			featureName:        "feature-1",
			accountID:          &accountID1,
			expectedScenarioID: scenarioID1,
		},
		{
			name:               "get account-specific scenario for user456",
			featureName:        "feature-1",
			accountID:          &accountID2,
			expectedScenarioID: scenarioID2,
		},
		{
			name:               "fallback to global scenario for unknown user",
			featureName:        "feature-1",
			accountID:          stringPtr("unknown-user"),
			expectedScenarioID: globalScenarioID,
		},
		{
			name:               "get global scenario when accountID is nil",
			featureName:        "feature-1",
			accountID:          nil,
			expectedScenarioID: globalScenarioID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetActiveScenario(ctx, tt.featureName, tt.accountID)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedScenarioID, result.ScenarioID)
		})
	}

	// Test non-existent feature
	_, err = repo.GetActiveScenario(ctx, "non-existent-feature", nil)
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)
}

func TestAccountScenarioRepository_DeactivateByFeatureAndAccount(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	accountID := "user123"
	scenarioID := primitive.NewObjectID()

	// Create account scenario
	accountScenario := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  scenarioID,
		AccountId:   &accountID,
	}
	err := repo.Create(ctx, accountScenario)
	require.NoError(t, err)

	// Verify it exists
	_, err = repo.GetActiveScenario(ctx, "feature-1", &accountID)
	require.NoError(t, err)

	// Deactivate it
	err = repo.DeactivateByFeatureAndAccount(ctx, "feature-1", &accountID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetActiveScenario(ctx, "feature-1", &accountID)
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)
}

func TestAccountScenarioRepository_DeactivateAllAccountSpecificMappings(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	scenarioID := primitive.NewObjectID()
	globalScenarioID := primitive.NewObjectID()

	accountID1 := "user123"
	accountID2 := "user456"

	// Create account-specific scenarios
	accountScenario1 := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  scenarioID,
		AccountId:   &accountID1,
	}
	accountScenario2 := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  scenarioID,
		AccountId:   &accountID2,
	}
	// Create global scenario
	globalScenario := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  globalScenarioID,
		AccountId:   nil,
	}

	err := repo.Create(ctx, accountScenario1)
	require.NoError(t, err)
	err = repo.Create(ctx, accountScenario2)
	require.NoError(t, err)
	err = repo.Create(ctx, globalScenario)
	require.NoError(t, err)

	// Deactivate all account-specific mappings
	err = repo.DeactivateAllAccountSpecificMappings(ctx, "feature-1")
	require.NoError(t, err)

	// Verify account-specific mappings are deleted
	_, err = repo.GetActiveScenario(ctx, "feature-1", &accountID1)
	// Should fallback to global scenario
	require.NoError(t, err)

	// Verify global scenario still exists
	result, err := repo.GetActiveScenario(ctx, "feature-1", nil)
	require.NoError(t, err)
	assert.Equal(t, globalScenarioID, result.ScenarioID)
}

func TestAccountScenarioRepository_GetByObjectID(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	accountID := "user123"
	accountScenario := &domain.AccountScenario{
		FeatureName: "test-feature",
		ScenarioID:  primitive.NewObjectID(),
		AccountId:   &accountID,
	}

	err := repo.Create(ctx, accountScenario)
	require.NoError(t, err)

	// Get by ObjectID
	result, err := repo.GetByObjectID(ctx, accountScenario.ID)
	require.NoError(t, err)
	assert.Equal(t, accountScenario.ID, result.ID)
	assert.Equal(t, accountScenario.FeatureName, result.FeatureName)
	assert.Equal(t, *accountScenario.AccountId, *result.AccountId)

	// Test with non-existent ID
	_, err = repo.GetByObjectID(ctx, primitive.NewObjectID())
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)
}

func TestAccountScenarioRepository_Delete(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	accountID := "user123"
	accountScenario := &domain.AccountScenario{
		FeatureName: "test-feature",
		ScenarioID:  primitive.NewObjectID(),
		AccountId:   &accountID,
	}

	err := repo.Create(ctx, accountScenario)
	require.NoError(t, err)

	// Delete the account scenario
	err = repo.Delete(ctx, accountScenario.ID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetByObjectID(ctx, accountScenario.ID)
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)
}

func TestAccountScenarioRepository_ListByFeature(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	accountID1 := "user123"
	accountID2 := "user456"

	// Create scenarios for feature-1
	scenarios := []*domain.AccountScenario{
		{
			FeatureName: "feature-1",
			ScenarioID:  primitive.NewObjectID(),
			AccountId:   &accountID1,
		},
		{
			FeatureName: "feature-1",
			ScenarioID:  primitive.NewObjectID(),
			AccountId:   &accountID2,
		},
		{
			FeatureName: "feature-1",
			ScenarioID:  primitive.NewObjectID(),
			AccountId:   nil, // global
		},
		{
			FeatureName: "feature-2",
			ScenarioID:  primitive.NewObjectID(),
			AccountId:   &accountID1,
		},
	}

	for _, s := range scenarios {
		err := repo.Create(ctx, s)
		require.NoError(t, err)
	}

	// List scenarios for feature-1
	result, err := repo.ListByFeature(ctx, "feature-1")
	require.NoError(t, err)
	assert.Equal(t, 3, len(result))

	// List scenarios for feature-2
	result, err = repo.ListByFeature(ctx, "feature-2")
	require.NoError(t, err)
	assert.Equal(t, 1, len(result))

	// List scenarios for non-existent feature
	result, err = repo.ListByFeature(ctx, "non-existent")
	require.NoError(t, err)
	assert.Equal(t, 0, len(result))
}

func TestAccountScenarioRepository_GlobalScenarioFallback(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewAccountScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	globalScenarioID := primitive.NewObjectID()

	// Create only global scenario (no account-specific ones)
	globalScenario := &domain.AccountScenario{
		FeatureName: "feature-1",
		ScenarioID:  globalScenarioID,
		AccountId:   nil,
	}

	err := repo.Create(ctx, globalScenario)
	require.NoError(t, err)

	// Test various account IDs should all get the global scenario
	accountIDs := []string{"user1", "user2", "user3"}
	for _, accID := range accountIDs {
		t.Run("accountID: "+accID, func(t *testing.T) {
			result, err := repo.GetActiveScenario(ctx, "feature-1", &accID)
			require.NoError(t, err)
			assert.Equal(t, globalScenarioID, result.ScenarioID)
			assert.Nil(t, result.AccountId) // Should be global scenario
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
