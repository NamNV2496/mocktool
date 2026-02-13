package repository

import (
	"testing"

	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestScenarioRepository_Create(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	scenario := &domain.Scenario{
		FeatureName: "test-feature",
		Name:        "test-scenario",
		Description: "Test scenario description",
	}

	err := repo.Create(ctx, scenario)
	require.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, scenario.ID)
}

func TestScenarioRepository_GetByObjectID(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	// Create a scenario
	scenario := &domain.Scenario{
		FeatureName: "test-feature",
		Name:        "test-scenario",
		Description: "Test description",
	}
	err := repo.Create(ctx, scenario)
	require.NoError(t, err)

	// Retrieve by ObjectID
	result, err := repo.GetByObjectID(ctx, scenario.ID)
	require.NoError(t, err)
	assert.Equal(t, scenario.ID, result.ID)
	assert.Equal(t, scenario.Name, result.Name)
	assert.Equal(t, scenario.FeatureName, result.FeatureName)

	// Test with non-existent ID
	nonExistentID := primitive.NewObjectID()
	_, err = repo.GetByObjectID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)
}

func TestScenarioRepository_GetByName(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	// Create scenarios
	scenario1 := &domain.Scenario{
		FeatureName: "feature-1",
		Name:        "scenario-a",
		Description: "Scenario A",
	}
	scenario2 := &domain.Scenario{
		FeatureName: "feature-2",
		Name:        "scenario-a",
		Description: "Scenario A for feature 2",
	}

	err := repo.Create(ctx, scenario1)
	require.NoError(t, err)
	err = repo.Create(ctx, scenario2)
	require.NoError(t, err)

	// Get by feature name and scenario name
	result, err := repo.GetByName(ctx, "feature-1", "scenario-a")
	require.NoError(t, err)
	assert.Equal(t, scenario1.ID, result.ID)
	assert.Equal(t, "feature-1", result.FeatureName)

	result, err = repo.GetByName(ctx, "feature-2", "scenario-a")
	require.NoError(t, err)
	assert.Equal(t, scenario2.ID, result.ID)
	assert.Equal(t, "feature-2", result.FeatureName)

	// Test with non-existent scenario
	_, err = repo.GetByName(ctx, "feature-1", "non-existent")
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)
}

func TestScenarioRepository_ListByFeatureNamePaginated(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	// Create scenarios for different features
	scenarios := []*domain.Scenario{
		{FeatureName: "feature-1", Name: "scenario-1", Description: "Scenario 1"},
		{FeatureName: "feature-1", Name: "scenario-2", Description: "Scenario 2"},
		{FeatureName: "feature-1", Name: "scenario-3", Description: "Scenario 3"},
		{FeatureName: "feature-2", Name: "scenario-1", Description: "Scenario 1 for feature 2"},
	}

	for _, s := range scenarios {
		err := repo.Create(ctx, s)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		featureName   string
		params        domain.PaginationParams
		expectedCount int
		expectedTotal int64
	}{
		{
			name:          "feature-1 all scenarios",
			featureName:   "feature-1",
			params:        domain.PaginationParams{Page: 1, PageSize: 10},
			expectedCount: 3,
			expectedTotal: 3,
		},
		{
			name:          "feature-1 paginated - page 1",
			featureName:   "feature-1",
			params:        domain.PaginationParams{Page: 1, PageSize: 2},
			expectedCount: 2,
			expectedTotal: 3,
		},
		{
			name:          "feature-1 paginated - page 2",
			featureName:   "feature-1",
			params:        domain.PaginationParams{Page: 2, PageSize: 2},
			expectedCount: 1,
			expectedTotal: 3,
		},
		{
			name:          "feature-2 scenarios",
			featureName:   "feature-2",
			params:        domain.PaginationParams{Page: 1, PageSize: 10},
			expectedCount: 1,
			expectedTotal: 1,
		},
		{
			name:          "non-existent feature",
			featureName:   "feature-999",
			params:        domain.PaginationParams{Page: 1, PageSize: 10},
			expectedCount: 0,
			expectedTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, total, err := repo.ListByFeatureNamePaginated(ctx, tt.featureName, tt.params)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, tt.expectedTotal, total)
		})
	}
}

func TestScenarioRepository_SearchByFeatureAndName(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	// Create test scenarios
	scenarios := []*domain.Scenario{
		{FeatureName: "auth", Name: "login-flow", Description: "Login flow"},
		{FeatureName: "auth", Name: "logout-flow", Description: "Logout flow"},
		{FeatureName: "auth", Name: "signup-flow", Description: "Signup flow"},
		{FeatureName: "payment", Name: "checkout-flow", Description: "Checkout"},
	}

	for _, s := range scenarios {
		err := repo.Create(ctx, s)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		featureName   string
		query         string
		expectedCount int
	}{
		{
			name:          "search 'login' in auth feature",
			featureName:   "auth",
			query:         "login",
			expectedCount: 1,
		},
		{
			name:          "search 'flow' in auth feature",
			featureName:   "auth",
			query:         "flow",
			expectedCount: 3,
		},
		{
			name:          "search 'checkout' in payment feature",
			featureName:   "payment",
			query:         "checkout",
			expectedCount: 1,
		},
		{
			name:          "search across all features (empty featureName)",
			featureName:   "",
			query:         "flow",
			expectedCount: 4,
		},
		{
			name:          "case insensitive search",
			featureName:   "auth",
			query:         "LOGIN",
			expectedCount: 1,
		},
		{
			name:          "non-existent query",
			featureName:   "auth",
			query:         "nonexistent",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := domain.PaginationParams{Page: 1, PageSize: 10}
			result, total, err := repo.SearchByFeatureAndName(ctx, tt.featureName, tt.query, params)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, int64(tt.expectedCount), total)
		})
	}
}

func TestScenarioRepository_UpdateByObjectID(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	// Create a scenario
	scenario := &domain.Scenario{
		FeatureName: "test-feature",
		Name:        "original-name",
		Description: "Original description",
	}
	err := repo.Create(ctx, scenario)
	require.NoError(t, err)

	// Update the scenario
	update := bson.M{
		"name":        "updated-name",
		"description": "Updated description",
	}

	err = repo.UpdateByObjectID(ctx, scenario.ID, update)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.GetByObjectID(ctx, scenario.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, "test-feature", updated.FeatureName) // Should remain unchanged
}

func TestScenarioRepository_Delete(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	// Create a scenario
	scenario := &domain.Scenario{
		FeatureName: "test-feature",
		Name:        "scenario-to-delete",
		Description: "Will be deleted",
	}
	err := repo.Create(ctx, scenario)
	require.NoError(t, err)

	// Verify it exists
	_, err = repo.GetByObjectID(ctx, scenario.ID)
	require.NoError(t, err)

	// Delete the scenario
	err = repo.Delete(ctx, scenario.ID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetByObjectID(ctx, scenario.ID)
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)

	// Delete non-existent scenario should not error
	err = repo.Delete(ctx, primitive.NewObjectID())
	assert.NoError(t, err)
}

func TestScenarioRepository_EmptyDatabase(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	params := domain.PaginationParams{Page: 1, PageSize: 10}

	// Test listing from empty database
	result, total, err := repo.ListByFeatureNamePaginated(ctx, "feature-1", params)
	require.NoError(t, err)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, int64(0), total)

	// Test search on empty database
	result, total, err = repo.SearchByFeatureAndName(ctx, "feature-1", "test", params)
	require.NoError(t, err)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, int64(0), total)
}

func TestScenarioRepository_MultipleUpdates(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewScenarioRepository(helper.DB)
	ctx := helper.GetContext()

	// Create a scenario
	scenario := &domain.Scenario{
		FeatureName: "test-feature",
		Name:        "scenario",
		Description: "Version 1",
	}
	err := repo.Create(ctx, scenario)
	require.NoError(t, err)

	// Perform multiple updates
	updates := []string{"Version 2", "Version 3", "Version 4"}
	for _, desc := range updates {
		update := bson.M{"description": desc}
		err = repo.UpdateByObjectID(ctx, scenario.ID, update)
		require.NoError(t, err)
	}

	// Verify final state
	result, err := repo.GetByObjectID(ctx, scenario.ID)
	require.NoError(t, err)
	assert.Equal(t, "Version 4", result.Description)
}
