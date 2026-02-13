package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/configs"
	controllerMocks "github.com/namnv2496/mocktool/mocks/controller"
	repositoryMocks "github.com/namnv2496/mocktool/mocks/repository"
	customValidator "github.com/namnv2496/mocktool/pkg/validator"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestValidation_CreateNewFeature tests that validation is properly enforced
func TestValidation_CreateNewFeature(t *testing.T) {
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
	).(*MockController)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "valid feature",
			requestBody:    `{"name":"test-feature","description":"Test description"}`,
			expectedStatus: http.StatusCreated,
			setupMocks: func() {
				featureRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:           "missing required name",
			requestBody:    `{"description":"Test description"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name is required",
			setupMocks:     func() {},
		},
		{
			name:           "name with spaces",
			requestBody:    `{"name":"test feature","description":"Test description"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name cannot contain spaces",
			setupMocks:     func() {},
		},
		{
			name:           "empty name",
			requestBody:    `{"name":"","description":"Test description"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name is required",
			setupMocks:     func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Validator = customValidator.NewValidator()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/mocktool/features", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			err := controller.CreateNewFeature(c)

			if tt.expectedStatus == http.StatusCreated {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				assert.Contains(t, httpErr.Message, tt.expectedError)
			}
		})
	}
}

// TestValidation_CreateNewScenario tests scenario validation
func TestValidation_CreateNewScenario(t *testing.T) {
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
	).(*MockController)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "valid scenario",
			requestBody:    `{"feature_name":"test-feature","name":"test-scenario","description":"Test"}`,
			expectedStatus: http.StatusCreated,
			setupMocks: func() {
				scenarioRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:           "missing feature_name",
			requestBody:    `{"name":"test-scenario","description":"Test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "FeatureName is required",
			setupMocks:     func() {},
		},
		{
			name:           "feature_name with spaces",
			requestBody:    `{"feature_name":"test feature","name":"test-scenario"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "FeatureName cannot contain spaces",
			setupMocks:     func() {},
		},
		{
			name:           "scenario name with spaces",
			requestBody:    `{"feature_name":"test-feature","name":"test scenario"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name cannot contain spaces",
			setupMocks:     func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Validator = customValidator.NewValidator()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/mocktool/scenarios", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			err := controller.CreateNewScenariosByFeature(c)

			if tt.expectedStatus == http.StatusCreated {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				assert.Contains(t, httpErr.Message, tt.expectedError)
			}
		})
	}
}

// TestValidation_CreateMockAPI tests mock API validation
func TestValidation_CreateMockAPI(t *testing.T) {
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
	).(*MockController)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name: "valid mock API",
			requestBody: `{
				"feature_name":"test-feature",
				"scenario_name":"test-scenario",
				"name":"test-api",
				"path":"/api/v1/test",
				"method":"POST",
				"input":{"username":"test"},
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusCreated,
			setupMocks: func() {
				mockAPIRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "missing required fields",
			requestBody: `{
				"description":"Test API",
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "is required",
			setupMocks:     func() {},
		},
		{
			name: "path with spaces",
			requestBody: `{
				"feature_name":"test-feature",
				"scenario_name":"test-scenario",
				"name":"test-api",
				"path":"/api/v1/test path",
				"method":"POST",
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Path cannot contain spaces",
			setupMocks:     func() {},
		},
		{
			name: "method with spaces",
			requestBody: `{
				"feature_name":"test-feature",
				"scenario_name":"test-scenario",
				"name":"test-api",
				"path":"/api/v1/test",
				"method":"POST METHOD",
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Method cannot contain spaces",
			setupMocks:     func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Validator = customValidator.NewValidator()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/mocktool/mockapis", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			err := controller.CreateMockAPIByScenario(c)

			if tt.expectedStatus == http.StatusCreated {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				assert.Contains(t, httpErr.Message, tt.expectedError)
			}
		})
	}
}
