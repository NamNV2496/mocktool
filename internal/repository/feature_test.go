package repository

import (
	"context"
	"testing"

	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestFeatureRepository_Create(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB)
	ctx := helper.GetContext()

	feature := &domain.Feature{
		Name:        "test-feature",
		Description: "Test feature description",
		IsActive:    true,
	}

	err := repo.Create(ctx, feature)
	require.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, feature.ID)
}

func TestFeatureRepository_ListAllPaginated(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB)
	ctx := helper.GetContext()

	// Create test features
	features := []*domain.Feature{
		{Name: "feature-1", Description: "First feature", IsActive: true},
		{Name: "feature-2", Description: "Second feature", IsActive: true},
		{Name: "feature-3", Description: "Third feature", IsActive: false},
	}

	for _, f := range features {
		err := repo.Create(ctx, f)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		params        domain.PaginationParams
		expectedCount int
		expectedTotal int64
	}{
		{
			name:          "first page with default size",
			params:        domain.PaginationParams{Page: 1, PageSize: 10},
			expectedCount: 3,
			expectedTotal: 3,
		},
		{
			name:          "paginated - first page size 2",
			params:        domain.PaginationParams{Page: 1, PageSize: 2},
			expectedCount: 2,
			expectedTotal: 3,
		},
		{
			name:          "paginated - second page size 2",
			params:        domain.PaginationParams{Page: 2, PageSize: 2},
			expectedCount: 1,
			expectedTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, total, err := repo.ListAllPaginated(ctx, tt.params)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, tt.expectedTotal, total)
		})
	}
}

func TestFeatureRepository_SearchByName(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB)
	ctx := helper.GetContext()

	// Create test features
	features := []*domain.Feature{
		{Name: "user-authentication", Description: "User auth", IsActive: true},
		{Name: "user-profile", Description: "User profile", IsActive: true},
		{Name: "payment-processing", Description: "Payments", IsActive: true},
	}

	for _, f := range features {
		err := repo.Create(ctx, f)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "search for 'user' - should find 2",
			query:         "user",
			expectedCount: 2,
		},
		{
			name:          "search for 'payment' - should find 1",
			query:         "payment",
			expectedCount: 1,
		},
		{
			name:          "search for 'auth' - should find 1",
			query:         "auth",
			expectedCount: 1,
		},
		{
			name:          "search for non-existent - should find 0",
			query:         "nonexistent",
			expectedCount: 0,
		},
		{
			name:          "case insensitive search 'USER'",
			query:         "USER",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := domain.PaginationParams{Page: 1, PageSize: 10}
			result, total, err := repo.SearchByName(ctx, tt.query, params)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, int64(tt.expectedCount), total)
		})
	}
}

func TestFeatureRepository_UpdateByObjectID(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB)
	ctx := helper.GetContext()

	// Create a feature
	feature := &domain.Feature{
		Name:        "original-name",
		Description: "Original description",
		IsActive:    true,
	}
	err := repo.Create(ctx, feature)
	require.NoError(t, err)

	// Update the feature
	update := bson.M{
		"name":        "updated-name",
		"description": "Updated description",
		"is_active":   false,
	}

	err = repo.UpdateByObjectID(ctx, feature.ID, update)
	require.NoError(t, err)

	// Verify the update
	baseRepo := repo.(*FeatureRepository).BaseRepository
	var updated domain.Feature
	err = baseRepo.FindOne(ctx, bson.M{"_id": feature.ID}, &updated)
	require.NoError(t, err)

	assert.Equal(t, "updated-name", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.False(t, updated.IsActive)
}

func TestFeatureRepository_ListActive(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB).(*FeatureRepository)
	ctx := helper.GetContext()

	// Create test features with different active states
	features := []*domain.Feature{
		{Name: "active-1", Description: "Active feature 1", IsActive: true},
		{Name: "active-2", Description: "Active feature 2", IsActive: true},
		{Name: "inactive-1", Description: "Inactive feature", IsActive: false},
	}

	for _, f := range features {
		err := repo.Create(ctx, f)
		require.NoError(t, err)
	}

	// List only active features
	result, err := repo.ListActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(result))

	// Verify all returned features are active
	for _, f := range result {
		assert.True(t, f.IsActive)
	}
}

func TestFeatureRepository_Update(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB).(*FeatureRepository)
	ctx := helper.GetContext()

	// Create a feature
	feature := &domain.Feature{
		Name:        "test-feature",
		Description: "Test description",
		IsActive:    true,
	}
	err := repo.Create(ctx, feature)
	require.NoError(t, err)

	// Note: This tests the Update method which uses int64 ID
	// Since we're using ObjectID, this test demonstrates the method exists
	// but may not be practically used
	update := bson.M{
		"description": "Updated via Update method",
	}

	// This will fail because we're using ObjectID, not int64 ID
	// But it tests that the method signature is correct
	err = repo.Update(ctx, 123, update)
	// We expect this to not panic, even if it doesn't find the document
	assert.Error(t, err) // Should error because no document with int64 ID exists
}

func TestFeatureRepository_EmptyDatabase(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB)
	ctx := helper.GetContext()

	// Test listing from empty database
	params := domain.PaginationParams{Page: 1, PageSize: 10}
	result, total, err := repo.ListAllPaginated(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, int64(0), total)

	// Test search on empty database
	result, total, err = repo.SearchByName(ctx, "test", params)
	require.NoError(t, err)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, int64(0), total)
}

func TestFeatureRepository_ConcurrentCreates(t *testing.T) {
	helper := SetupTestDB(t)
	defer helper.Cleanup(t)

	repo := NewFeatureRepository(helper.DB)
	ctx := context.Background()

	// Create features concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(index int) {
			feature := &domain.Feature{
				Name:        "concurrent-feature-" + string(rune(index)),
				Description: "Created concurrently",
				IsActive:    true,
			}
			err := repo.Create(ctx, feature)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all were created
	params := domain.PaginationParams{Page: 1, PageSize: 10}
	result, total, err := repo.ListAllPaginated(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Equal(t, 5, len(result))
}
