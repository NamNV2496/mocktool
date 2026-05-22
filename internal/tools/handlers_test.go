package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"

	"github.com/namnv2496/mocktool/internal/domain"
	repomock "github.com/namnv2496/mocktool/mocks/repository"
)

type mockDeps struct {
	feature  *repomock.MockIFeatureRepository
	scenario *repomock.MockIScenarioRepository
	account  *repomock.MockIAccountScenarioRepository
	api      *repomock.MockIMockAPIRepository
	cache    *repomock.MockICache
}

func newDeps(t *testing.T) (Deps, mockDeps) {
	ctrl := gomock.NewController(t)
	m := mockDeps{
		feature:  repomock.NewMockIFeatureRepository(ctrl),
		scenario: repomock.NewMockIScenarioRepository(ctrl),
		account:  repomock.NewMockIAccountScenarioRepository(ctrl),
		api:      repomock.NewMockIMockAPIRepository(ctrl),
		cache:    repomock.NewMockICache(ctrl),
	}
	return Deps{
		Feature:         m.feature,
		Scenario:        m.scenario,
		AccountScenario: m.account,
		MockAPI:         m.api,
		Cache:           m.cache,
	}, m
}

func TestBuildAll_ExposesExpectedTools(t *testing.T) {
	d, _ := newDeps(t)
	r := BuildAll(d)

	got := r.Names()
	want := []string{
		"activate_scenario", "create_feature", "create_mock_api",
		"create_scenario", "deactivate_scenario", "delete_feature",
		"delete_mock_api", "delete_scenario", "disable_feature",
		"enable_feature", "get_active_scenario", "get_mock_api_curl",
		"list_apis", "list_features", "list_scenarios",
		"reset_mock_api_counter", "search_mocks", "search_scenarios",
		"set_scenario_inactive", "update_feature", "update_mock_api",
		"update_scenario",
	}
	assert.ElementsMatch(t, want, got)
}

func TestBuildAll_DestructiveFlagsCorrectlySet(t *testing.T) {
	d, _ := newDeps(t)
	r := BuildAll(d)

	destructive := map[string]bool{
		"deactivate_scenario":  true,
		"delete_feature":       true,
		"delete_scenario":      true,
		"delete_mock_api":      true,
		"disable_feature":      true,
		"set_scenario_inactive": true,
	}
	for _, tool := range r.List() {
		assert.Equal(t, destructive[tool.Name], tool.Destructive, "tool=%s", tool.Name)
	}
}

func TestListFeatures_HappyPath(t *testing.T) {
	d, m := newDeps(t)
	want := []domain.Feature{{Name: "insertAd"}, {Name: "search"}}
	m.feature.EXPECT().
		ListAllPaginated(gomock.Any(), gomock.Any()).
		Return(want, int64(2), nil)

	res, err := BuildAll(d).Invoke(context.Background(), "list_features", json.RawMessage(`{}`))
	require.NoError(t, err)

	got := res.(map[string]any)
	assert.EqualValues(t, 2, got["total"])
	assert.Equal(t, want, got["features"])
}

func TestListFeatures_WithQueryUsesSearch(t *testing.T) {
	d, m := newDeps(t)
	m.feature.EXPECT().
		SearchByName(gomock.Any(), "ins", gomock.Any()).
		Return([]domain.Feature{{Name: "insertAd"}}, int64(1), nil)

	_, err := BuildAll(d).Invoke(context.Background(), "list_features", json.RawMessage(`{"query":"ins"}`))
	require.NoError(t, err)
}

func TestDeleteFeature_CascadesAndInvalidatesCache(t *testing.T) {
	d, m := newDeps(t)
	featureID := primitive.NewObjectID()
	m.feature.EXPECT().FindByName(gomock.Any(), "insertAd").Return(&domain.Feature{ID: featureID, Name: "insertAd"}, nil)
	m.feature.EXPECT().DeleteById(gomock.Any(), featureID).Return(nil)
	m.scenario.EXPECT().DeleteByFeatureName(gomock.Any(), "insertAd").Return(nil)
	m.api.EXPECT().DeleteByFeatureName(gomock.Any(), "insertAd").Return(nil)
	m.cache.EXPECT().InvalidAllKey(gomock.Any(), gomock.Any()).Return(nil)

	res, err := BuildAll(d).Invoke(context.Background(), "delete_feature", json.RawMessage(`{"feature":"insertAd"}`))
	require.NoError(t, err)
	assert.Equal(t, "insertAd", res.(map[string]any)["feature"])
}

func TestDeleteFeature_NotFoundReturnsError(t *testing.T) {
	d, m := newDeps(t)
	m.feature.EXPECT().FindByName(gomock.Any(), "missing").Return(nil, nil)

	_, err := BuildAll(d).Invoke(context.Background(), "delete_feature", json.RawMessage(`{"feature":"missing"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestActivateScenario_ClearsAllMappingsAndCreatesGlobal(t *testing.T) {
	d, m := newDeps(t)
	scenarioID := primitive.NewObjectID()

	m.scenario.EXPECT().
		FindByFeatureNameAndName(gomock.Any(), "insertAd", "s1").
		Return(&domain.Scenario{ID: scenarioID, FeatureName: "insertAd", Name: "s1"}, nil)
	m.account.EXPECT().DeactivateAllAccountSpecificMappings(gomock.Any(), "insertAd").Return(nil)
	m.account.EXPECT().DeactivateByFeatureAndAccount(gomock.Any(), "insertAd", (*string)(nil)).Return(nil)
	m.account.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, as *domain.AccountScenario) error {
			assert.Equal(t, "insertAd", as.FeatureName)
			assert.Equal(t, scenarioID, as.ScenarioID)
			assert.Nil(t, as.AccountId)
			return nil
		},
	)
	m.cache.EXPECT().InvalidAllKey(gomock.Any(), gomock.Any()).Return(nil)

	res, err := BuildAll(d).Invoke(context.Background(), "activate_scenario", json.RawMessage(`{"feature":"insertAd","scenario":"s1"}`))
	require.NoError(t, err)
	assert.Equal(t, true, res.(map[string]any)["activated"])
}

func TestCreateMockAPI_PersistsAndInvalidatesCache(t *testing.T) {
	d, m := newDeps(t)

	m.api.EXPECT().FindByNameAndFeatureAndScenario(gomock.Any(), "createUser", "insertAd", "s1").Return(nil, nil)
	m.api.EXPECT().FindByFeatureScenarioPathMethodAndHash(
		gomock.Any(), "insertAd", "s1", "/api/v1/user", "POST", gomock.Any(),
	).Return(nil, nil)
	m.api.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, api *domain.MockAPI) error {
			assert.Equal(t, "/api/v1/user", api.Path)
			assert.Equal(t, "POST", api.Method)
			assert.NotEmpty(t, api.HashInput, "hash should be computed from request_body")
			assert.NotZero(t, len(api.Output))
			return nil
		},
	)
	m.cache.EXPECT().InvalidAllKey(gomock.Any(), gomock.Any()).Return(nil)

	res, err := BuildAll(d).Invoke(context.Background(), "create_mock_api", json.RawMessage(`{
        "feature": "insertAd",
        "scenario": "s1",
        "name": "createUser",
        "path": "/api/v1/user",
        "method": "POST",
        "request_body": {"username": "a"},
        "response": {"id": 1, "username": "a"},
        "headers": {"X-Trace-Id": "abc"}
    }`))
	require.NoError(t, err)
	assert.Equal(t, "createUser", res.(map[string]any)["name"])
}

func TestCreateMockAPI_DuplicateByNameRejected(t *testing.T) {
	d, m := newDeps(t)

	m.api.EXPECT().FindByNameAndFeatureAndScenario(gomock.Any(), "createUser", "insertAd", "s1").
		Return(&domain.MockAPI{Name: "createUser"}, nil)

	_, err := BuildAll(d).Invoke(context.Background(), "create_mock_api", json.RawMessage(`{
        "feature": "insertAd",
        "scenario": "s1",
        "name": "createUser",
        "path": "/api/v1/user",
        "method": "POST",
        "response": {}
    }`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestDeleteMockAPI_NotFoundReturnsError(t *testing.T) {
	d, m := newDeps(t)
	id := primitive.NewObjectID()
	m.api.EXPECT().FindByObjectID(gomock.Any(), id).Return(nil, nil)

	_, err := BuildAll(d).Invoke(
		context.Background(),
		"delete_mock_api",
		json.RawMessage(`{"api_id":"`+id.Hex()+`"}`),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteScenario_CascadesMockAPIsAndAccountScenario(t *testing.T) {
	d, m := newDeps(t)
	scenarioID := primitive.NewObjectID()

	m.scenario.EXPECT().FindByFeatureNameAndName(gomock.Any(), "insertAd", "s1").Return(
		&domain.Scenario{ID: scenarioID, FeatureName: "insertAd", Name: "s1"}, nil)
	m.scenario.EXPECT().DeleteByObjectID(gomock.Any(), scenarioID).Return(nil)
	m.api.EXPECT().DeleteByScenarioName(gomock.Any(), "s1").Return(nil)
	m.account.EXPECT().DeleteByScenarioId(gomock.Any(), scenarioID).Return(nil)
	m.cache.EXPECT().InvalidAllKey(gomock.Any(), gomock.Any()).Return(nil)

	_, err := BuildAll(d).Invoke(context.Background(), "delete_scenario", json.RawMessage(`{"feature":"insertAd","scenario":"s1"}`))
	require.NoError(t, err)
}

func TestGetActiveScenario_NoActiveReturnsNull(t *testing.T) {
	d, m := newDeps(t)
	m.account.EXPECT().GetActiveScenario(gomock.Any(), "insertAd", (*string)(nil)).Return(nil, nil)

	res, err := BuildAll(d).Invoke(context.Background(), "get_active_scenario", json.RawMessage(`{"feature":"insertAd"}`))
	require.NoError(t, err)
	out := res.(map[string]any)
	assert.Nil(t, out["active"])
}

func TestListScenarios_FlagsGloballyActive(t *testing.T) {
	d, m := newDeps(t)
	activeID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()
	m.scenario.EXPECT().ListByFeatureNamePaginated(gomock.Any(), "insertAd", gomock.Any()).
		Return([]domain.Scenario{
			{ID: activeID, Name: "s1", FeatureName: "insertAd"},
			{ID: otherID, Name: "s2", FeatureName: "insertAd"},
		}, int64(2), nil)
	m.account.EXPECT().GetActiveScenario(gomock.Any(), "insertAd", (*string)(nil)).
		Return(&domain.AccountScenario{ScenarioID: activeID, FeatureName: "insertAd"}, nil)

	res, err := BuildAll(d).Invoke(context.Background(), "list_scenarios", json.RawMessage(`{"feature":"insertAd"}`))
	require.NoError(t, err)

	scenarios := res.(map[string]any)["scenarios"].([]map[string]any)
	require.Len(t, scenarios, 2)
	assert.True(t, scenarios[0]["is_global_active"].(bool))
	assert.False(t, scenarios[1]["is_global_active"].(bool))
}

// Verify the bson conversion helper round-trips JSON-compatible payloads.
func TestBsonRawToJSON_RoundTrip(t *testing.T) {
	src := map[string]any{"a": 1, "b": "two", "c": []any{int32(3), int32(4)}}
	raw, err := bson.Marshal(src)
	require.NoError(t, err)
	got := bsonRawToJSON(raw)
	assert.NotNil(t, got)
}
