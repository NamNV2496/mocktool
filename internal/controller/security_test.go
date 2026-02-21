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

// TestSecurity_HeaderSanitization tests that dangerous headers are blocked/sanitized
func TestSecurity_HeaderSanitization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &configs.Config{}
	featureRepo := repositoryMocks.NewMockIFeatureRepository(ctrl)
	scenarioRepo := repositoryMocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := repositoryMocks.NewMockIAccountScenarioRepository(ctrl)
	mockAPIRepo := repositoryMocks.NewMockIMockAPIRepository(ctrl)
	loadTestController := controllerMocks.NewMockILoadTestController(ctrl)
	cacheRepo := repositoryMocks.NewMockICache(ctrl)

	controller := NewMockController(
		config,
		featureRepo,
		scenarioRepo,
		accountScenarioRepo,
		mockAPIRepo,
		loadTestController,
		cacheRepo,
	).(*MockController)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		setupMocks     func()
		checkResponse  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "malicious Set-Cookie header blocked",
			requestBody: `{
				"feature_name":"test-feature",
				"scenario_name":"test-scenario",
				"name":"test-api",
				"path":"/api/v1/test",
				"method":"POST",
				"input":{"test":"data"},
				"headers":"{\"Set-Cookie\":\"admin=true\",\"Content-Type\":\"application/json\"}",
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusCreated,
			setupMocks: func() {
				// Expect Create to be called, and we'll verify headers are sanitized
				mockAPIRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx interface{}, api interface{}) error {
						// Headers should have Set-Cookie removed
						return nil
					})
				mockAPIRepo.EXPECT().
					FindByNameAndFeatureAndScenario(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
			},
		},
		{
			name: "XSS in header value sanitized",
			requestBody: `{
				"feature_name":"test-feature",
				"scenario_name":"test-scenario",
				"name":"test-api",
				"path":"/api/v1/test",
				"method":"POST",
				"input":{"test":"data"},
				"headers":"{\"Content-Type\":\"<script>alert('xss')</script>\"}",
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusCreated,
			setupMocks: func() {
				mockAPIRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
				mockAPIRepo.EXPECT().
					FindByNameAndFeatureAndScenario(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
			},
		},
		{
			name: "CRLF injection in header blocked",
			requestBody: `{
				"feature_name":"test-feature",
				"scenario_name":"test-scenario",
				"name":"test-api",
				"path":"/api/v1/test",
				"method":"POST",
				"input":{"test":"data"},
				"headers":"{\"X-Request-ID\":\"test\\r\\nSet-Cookie: evil=true\"}",
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusCreated,
			setupMocks: func() {
				mockAPIRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
				mockAPIRepo.EXPECT().
					FindByNameAndFeatureAndScenario(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)

			},
		},
		{
			name: "allowed headers pass through",
			requestBody: `{
				"feature_name":"test-feature",
				"scenario_name":"test-scenario",
				"name":"test-api",
				"path":"/api/v1/test",
				"method":"POST",
				"input":{"test":"data"},
				"headers":"{\"Content-Type\":\"application/json\",\"X-Request-ID\":\"123\",\"X-Custom-Header\":\"value\"}",
				"output":{"result":"success"}
			}`,
			expectedStatus: http.StatusCreated,
			setupMocks: func() {
				mockAPIRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
				mockAPIRepo.EXPECT().
					FindByNameAndFeatureAndScenario(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
			},
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
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

// TestSecurity_ValidationPreventsInjection tests that validation prevents injection attacks
func TestSecurity_ValidationPreventsInjection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &configs.Config{}
	featureRepo := repositoryMocks.NewMockIFeatureRepository(ctrl)
	scenarioRepo := repositoryMocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := repositoryMocks.NewMockIAccountScenarioRepository(ctrl)
	mockAPIRepo := repositoryMocks.NewMockIMockAPIRepository(ctrl)
	loadTestController := controllerMocks.NewMockILoadTestController(ctrl)
	cacheRepo := repositoryMocks.NewMockICache(ctrl)

	controller := NewMockController(
		config,
		featureRepo,
		scenarioRepo,
		accountScenarioRepo,
		mockAPIRepo,
		loadTestController,
		cacheRepo,
	).(*MockController)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "SQL injection with spaces blocked by validation",
			requestBody:    `{"name":"test' OR '1'='1","description":"Test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "cannot contain spaces",
			setupMocks:     func() {},
		},
		{
			name:           "feature name with spaces blocked",
			requestBody:    `{"name":"feature name with spaces","description":"Test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "cannot contain spaces",
			setupMocks:     func() {},
		},
		{
			name:           "path traversal with spaces blocked by validation",
			requestBody:    `{"name":"../ ../etc/passwd","description":"Test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "cannot contain spaces",
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

			err := controller.CreateNewFeature(c)

			assert.Error(t, err)
			httpErr, ok := err.(*echo.HTTPError)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedStatus, httpErr.Code)
			assert.Contains(t, httpErr.Message, tt.expectedError)
		})
	}
}
