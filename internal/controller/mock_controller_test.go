package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/domain"
	controllerMocks "github.com/namnv2496/mocktool/mocks/controller"
	repositoryMocks "github.com/namnv2496/mocktool/mocks/repository"
	customValidator "github.com/namnv2496/mocktool/pkg/validator"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

func setupTestController(t *testing.T) (*MockController, *gomock.Controller, *repositoryMocks.MockIFeatureRepository, *repositoryMocks.MockIScenarioRepository, *repositoryMocks.MockIAccountScenarioRepository, *repositoryMocks.MockIMockAPIRepository) {
	ctrl := gomock.NewController(t)

	config := &configs.Config{
		AppConfig: configs.AppConfig{
			HTTPPort: ":8081",
		},
	}

	featureRepo := repositoryMocks.NewMockIFeatureRepository(ctrl)
	scenarioRepo := repositoryMocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := repositoryMocks.NewMockIAccountScenarioRepository(ctrl)
	mockAPIRepo := repositoryMocks.NewMockIMockAPIRepository(ctrl)
	loadTestController := controllerMocks.NewMockILoadTestController(ctrl)

	// Setup expectation for RegisterRoutes which is called in StartHttpServer
	loadTestController.EXPECT().RegisterRoutes(gomock.Any()).AnyTimes()

	controller := NewMockController(
		config,
		featureRepo,
		scenarioRepo,
		accountScenarioRepo,
		mockAPIRepo,
		loadTestController,
	).(*MockController)

	return controller, ctrl, featureRepo, scenarioRepo, accountScenarioRepo, mockAPIRepo
}

func TestMockController_HealthCheck(t *testing.T) {
	controller, ctrl, _, _, _, _ := setupTestController(t)
	defer ctrl.Finish()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.HealthCheck(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "mocktool", response["service"])
}

func TestMockController_ReadinessCheck(t *testing.T) {
	controller, ctrl, featureRepo, _, _, _ := setupTestController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		setupMocks     func()
		expectedStatus int
		expectedReady  bool
	}{
		{
			name: "database connected - ready",
			setupMocks: func() {
				featureRepo.EXPECT().
					ListAllPaginated(gomock.Any(), gomock.Any()).
					Return([]domain.Feature{}, int64(0), nil)
			},
			expectedStatus: http.StatusOK,
			expectedReady:  true,
		},
		{
			name: "database error - not ready",
			setupMocks: func() {
				featureRepo.EXPECT().
					ListAllPaginated(gomock.Any(), gomock.Any()).
					Return(nil, int64(0), assert.AnError)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedReady:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			err := controller.ReadinessCheck(c)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &response)

			if tt.expectedReady {
				assert.Equal(t, "ready", response["status"])
				assert.Equal(t, "connected", response["database"])
			} else {
				assert.Equal(t, "not ready", response["status"])
			}
		})
	}
}

func TestMockController_GetFeatures(t *testing.T) {
	controller, ctrl, featureRepo, _, _, _ := setupTestController(t)
	defer ctrl.Finish()

	features := []domain.Feature{
		{
			ID:          primitive.NewObjectID(),
			Name:        "feature1",
			Description: "Feature 1",
			IsActive:    true,
		},
		{
			ID:          primitive.NewObjectID(),
			Name:        "feature2",
			Description: "Feature 2",
			IsActive:    true,
		},
	}

	featureRepo.EXPECT().
		ListAllPaginated(gomock.Any(), gomock.Any()).
		Return(features, int64(2), nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mocktool/features", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.GetFeatures(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response domain.PaginatedResponse[domain.Feature]
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, int64(2), response.Total)
	assert.Len(t, response.Data, 2)
}

func TestMockController_SearchFeaturesByName(t *testing.T) {
	controller, ctrl, featureRepo, _, _, _ := setupTestController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		query          string
		setupMocks     func()
		expectedStatus int
		wantErr        bool
	}{
		{
			name:  "successful search",
			query: "test",
			setupMocks: func() {
				features := []domain.Feature{
					{Name: "test-feature"},
				}
				featureRepo.EXPECT().
					SearchByName(gomock.Any(), "test", gomock.Any()).
					Return(features, int64(1), nil)
			},
			expectedStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "missing query parameter",
			query:          "",
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			url := "/api/v1/mocktool/features/search"
			if tt.query != "" {
				url += "?q=" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			err := controller.SearchFeaturesByName(c)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestMockController_CreateNewFeature(t *testing.T) {
	controller, ctrl, featureRepo, _, _, _ := setupTestController(t)
	defer ctrl.Finish()

	featureRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil)

	e := echo.New()
	e.Validator = customValidator.NewValidator()

	featureJSON := `{"name":"test-feature","description":"test description"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mocktool/features", strings.NewReader(featureJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.CreateNewFeature(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response domain.Feature
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "test-feature", response.Name)
	assert.Equal(t, "test description", response.Description)
	assert.True(t, response.IsActive)
}

func TestMockController_ListScenarioByFeature(t *testing.T) {
	controller, ctrl, _, scenarioRepo, accountScenarioRepo, _ := setupTestController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		featureName    string
		setupMocks     func()
		expectedStatus int
		wantErr        bool
	}{
		{
			name:        "successful list",
			featureName: "test-feature",
			setupMocks: func() {
				scenarioID := primitive.NewObjectID()
				scenarios := []domain.Scenario{
					{Name: "scenario1"},
				}
				accountScenario := &domain.AccountScenario{
					ScenarioID: scenarioID,
				}
				scenario := &domain.Scenario{
					ID:          scenarioID,
					FeatureName: "test-feature",
					Name:        "test-scenario",
				}
				scenarioRepo.EXPECT().
					GetByObjectID(gomock.Any(), scenarioID).
					Return(scenario, nil)
				scenarioRepo.EXPECT().
					ListByFeatureNamePaginated(gomock.Any(), "test-feature", gomock.Any()).
					Return(scenarios, int64(1), nil)
				accountScenarioRepo.EXPECT().
					GetActiveScenario(gomock.Any(), "test-feature", gomock.Any()).
					Return(accountScenario, nil)

			},
			expectedStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "missing feature name",
			featureName:    "",
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			url := "/api/v1/mocktool/scenarios"
			if tt.featureName != "" {
				url += "?feature_name=" + tt.featureName
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			err := controller.ListScenariosByFeature(c)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestMockController_CreateNewScenarioByFeature(t *testing.T) {
	controller, ctrl, _, scenarioRepo, _, _ := setupTestController(t)
	defer ctrl.Finish()

	scenarioRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil)

	e := echo.New()
	e.Validator = customValidator.NewValidator()

	scenarioJSON := `{"feature_name":"test-feature","name":"test-scenario","description":"test desc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mocktool/scenarios", strings.NewReader(scenarioJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.CreateNewScenariosByFeature(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestMockController_ActivateScenario(t *testing.T) {
	controller, ctrl, _, scenarioRepo, accountScenarioRepo, _ := setupTestController(t)
	defer ctrl.Finish()

	scenarioID := primitive.NewObjectID()
	scenario := &domain.Scenario{
		ID:          scenarioID,
		FeatureName: "test-feature",
		Name:        "test-scenario",
	}

	scenarioRepo.EXPECT().
		GetByObjectID(gomock.Any(), scenarioID).
		Return(scenario, nil)

	// When activating globally (no account_id param), it deactivates all account-specific mappings first
	accountScenarioRepo.EXPECT().
		DeactivateAllAccountSpecificMappings(gomock.Any(), "test-feature").
		Return(nil)

	accountScenarioRepo.EXPECT().
		DeactivateByFeatureAndAccount(gomock.Any(), "test-feature", gomock.Any()).
		Return(nil)

	accountScenarioRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mocktool/scenarios/"+scenarioID.Hex()+"/activate", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("scenario_id")
	c.SetParamValues(scenarioID.Hex())

	err := controller.ActivateScenario(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMockController_ListActiveScenarioByFeature(t *testing.T) {
	controller, ctrl, _, scenarioRepo, accountScenarioRepo, _ := setupTestController(t)
	defer ctrl.Finish()

	scenarioID := primitive.NewObjectID()
	accountScenario := &domain.AccountScenario{
		ScenarioID: scenarioID,
	}
	scenario := &domain.Scenario{
		ID:   scenarioID,
		Name: "active-scenario",
	}

	accountScenarioRepo.EXPECT().
		GetActiveScenario(gomock.Any(), "test-feature", gomock.Any()).
		Return(accountScenario, nil)

	scenarioRepo.EXPECT().
		GetByObjectID(gomock.Any(), scenarioID).
		Return(scenario, nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mocktool/scenarios/active?feature_name=test-feature", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.ListActiveScenariosByFeature(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response domain.Scenario
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "active-scenario", response.Name)
}

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name         string
		queryParams  string
		expectedPage int
		expectedSize int
	}{
		{
			name:         "default params",
			queryParams:  "",
			expectedPage: domain.DefaultPage,
			expectedSize: domain.DefaultPageSize,
		},
		{
			name:         "custom params",
			queryParams:  "?page=2&page_size=20",
			expectedPage: 2,
			expectedSize: 20,
		},
		{
			name:         "invalid params default",
			queryParams:  "?page=-1&page_size=0",
			expectedPage: domain.DefaultPage,
			expectedSize: domain.DefaultPageSize,
		},
		{
			name:         "exceeds max page size",
			queryParams:  "?page=1&page_size=200",
			expectedPage: 1,
			expectedSize: domain.MaxPageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			params := parsePaginationParams(c)

			assert.Equal(t, tt.expectedPage, params.Page)
			assert.Equal(t, tt.expectedSize, params.PageSize)
		})
	}
}

func TestNewMockController(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &configs.Config{}
	featureRepo := repositoryMocks.NewMockIFeatureRepository(ctrl)
	scenarioRepo := repositoryMocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := repositoryMocks.NewMockIAccountScenarioRepository(ctrl)
	mockAPIRepo := repositoryMocks.NewMockIMockAPIRepository(ctrl)
	loadTestController := controllerMocks.NewMockILoadTestController(ctrl)

	controller := NewMockController(
		config,
		featureRepo,
		scenarioRepo,
		accountScenarioRepo,
		mockAPIRepo,
		loadTestController,
	)

	assert.NotNil(t, controller)
	assert.Implements(t, (*IMockController)(nil), controller)
}
