package usecase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/domain"
	mocks "github.com/namnv2496/mocktool/mocks/repository"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/mock/gomock"
)

func TestForwardUC_ResponseMockData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIRepo := mocks.NewMockIMockAPIRepository(ctrl)
	scenarioRepo := mocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := mocks.NewMockIAccountScenarioRepository(ctrl)

	uc := NewForwardUC(mockAPIRepo, scenarioRepo, accountScenarioRepo)

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		setupMocks     func()
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
		wantErr        bool
	}{
		{
			name: "successful mock response",
			setupRequest: func() *http.Request {
				body := map[string]interface{}{
					"field1": "value1",
					"field2": "value2",
				}
				bodyBytes, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/forward/api/v1/test", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Account-Id", "test-account")
				req.Header.Set("X-Feature-Name", "test-feature")
				return req
			},
			setupMocks: func() {
				scenarioID := primitive.NewObjectID()
				accountScenario := &domain.AccountScenario{
					ScenarioID: scenarioID,
				}
				scenario := &domain.Scenario{
					ID:   scenarioID,
					Name: "test-scenario",
				}

				output := map[string]interface{}{
					"success": true,
					"data":    "test response",
				}
				outputBson, _ := bson.Marshal(output)

				mockAPI := &domain.MockAPI{
					Output: outputBson,
				}

				accountIdPtr := "test-account"
				accountScenarioRepo.EXPECT().
					GetActiveScenario(gomock.Any(), "test-feature", &accountIdPtr).
					Return(accountScenario, nil)

				scenarioRepo.EXPECT().
					GetByObjectID(gomock.Any(), scenarioID).
					Return(scenario, nil)

				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(
						gomock.Any(),
						"test-feature",
						"test-scenario",
						gomock.Any(),
						"POST",
						gomock.Any(),
					).
					Return(mockAPI, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response["success"].(bool))
			},
			wantErr: false,
		},
		{
			name: "missing X-Account-Id header",
			setupRequest: func() *http.Request {
				body := map[string]interface{}{"field1": "value1"}
				bodyBytes, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/forward/api/v1/test", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Feature-Name", "test-feature")
				return req
			},
			setupMocks: func() {
				// No mocks needed - should fail before repository calls
			},
			expectedStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "missing X-Feature-Name header",
			setupRequest: func() *http.Request {
				body := map[string]interface{}{"field1": "value1"}
				bodyBytes, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/forward/api/v1/test", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Account-Id", "test-account")
				return req
			},
			setupMocks: func() {
				// No mocks needed
			},
			expectedStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "mock API not found",
			setupRequest: func() *http.Request {
				body := map[string]interface{}{"field1": "value1"}
				bodyBytes, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/forward/api/v1/test", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Account-Id", "test-account")
				req.Header.Set("X-Feature-Name", "test-feature")
				return req
			},
			setupMocks: func() {
				scenarioID := primitive.NewObjectID()
				accountScenario := &domain.AccountScenario{
					ScenarioID: scenarioID,
				}
				scenario := &domain.Scenario{
					ID:   scenarioID,
					Name: "test-scenario",
				}

				accountIdPtr := "test-account"
				accountScenarioRepo.EXPECT().
					GetActiveScenario(gomock.Any(), "test-feature", &accountIdPtr).
					Return(accountScenario, nil)

				scenarioRepo.EXPECT().
					GetByObjectID(gomock.Any(), scenarioID).
					Return(scenario, nil)

				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(nil, mongo.ErrNoDocuments)
			},
			expectedStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "empty request body",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/forward/api/v1/test", nil)
				req.Header.Set("X-Account-Id", "test-account")
				req.Header.Set("X-Feature-Name", "test-feature")
				return req
			},
			setupMocks: func() {
				scenarioID := primitive.NewObjectID()
				accountScenario := &domain.AccountScenario{
					ScenarioID: scenarioID,
				}
				scenario := &domain.Scenario{
					ID:   scenarioID,
					Name: "test-scenario",
				}

				output := map[string]interface{}{"result": "ok"}
				outputBson, _ := bson.Marshal(output)
				mockAPI := &domain.MockAPI{
					Output: outputBson,
				}

				accountIdPtr := "test-account"
				accountScenarioRepo.EXPECT().
					GetActiveScenario(gomock.Any(), "test-feature", &accountIdPtr).
					Return(accountScenario, nil)

				scenarioRepo.EXPECT().
					GetByObjectID(gomock.Any(), scenarioID).
					Return(scenario, nil)

				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						"GET",
						"", // Empty hash for empty body
					).
					Return(mockAPI, nil)
			},
			expectedStatus: http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := tt.setupRequest()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			// Execute
			err := uc.ResponseMockData(c)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestForwardUC_ResponsePublicMockData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIRepo := mocks.NewMockIMockAPIRepository(ctrl)
	scenarioRepo := mocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := mocks.NewMockIAccountScenarioRepository(ctrl)

	uc := NewForwardUC(mockAPIRepo, scenarioRepo, accountScenarioRepo)

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		setupMocks     func()
		expectedStatus int
		wantErr        bool
	}{
		{
			name: "successful public mock response",
			setupRequest: func() *http.Request {
				body := map[string]interface{}{"field1": "value1"}
				bodyBytes, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/public/forward/api/v1/test", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Feature-Name", "test-feature")
				return req
			},
			setupMocks: func() {
				scenarioID := primitive.NewObjectID()
				accountScenario := &domain.AccountScenario{
					ScenarioID: scenarioID,
				}
				scenario := &domain.Scenario{
					ID:   scenarioID,
					Name: "test-scenario",
				}

				output := map[string]interface{}{"public": "response"}
				outputBson, _ := bson.Marshal(output)
				mockAPI := &domain.MockAPI{
					Output: outputBson,
				}

				// For public API, accountId should be nil
				var nilAccountId *string = nil
				accountScenarioRepo.EXPECT().
					GetActiveScenario(gomock.Any(), "test-feature", nilAccountId).
					Return(accountScenario, nil)

				scenarioRepo.EXPECT().
					GetByObjectID(gomock.Any(), scenarioID).
					Return(scenario, nil)

				mockAPIRepo.EXPECT().
					FindByFeatureScenarioPathMethodAndHash(
						gomock.Any(),
						"test-feature",
						"test-scenario",
						gomock.Any(),
						"POST",
						gomock.Any(),
					).
					Return(mockAPI, nil)
			},
			expectedStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "missing feature name for public API",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/public/forward/api/v1/test", nil)
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := tt.setupRequest()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupMocks()

			err := uc.ResponsePublicMockData(c)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestForwardUC_WithQueryParameters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIRepo := mocks.NewMockIMockAPIRepository(ctrl)
	scenarioRepo := mocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := mocks.NewMockIAccountScenarioRepository(ctrl)

	uc := NewForwardUC(mockAPIRepo, scenarioRepo, accountScenarioRepo)

	t.Run("request with query parameters", func(t *testing.T) {
		body := map[string]interface{}{"field1": "value1"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/forward/api/v1/test?param1=value1&param2=value2", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Account-Id", "test-account")
		req.Header.Set("X-Feature-Name", "test-feature")

		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		scenarioID := primitive.NewObjectID()
		accountScenario := &domain.AccountScenario{
			ScenarioID: scenarioID,
		}
		scenario := &domain.Scenario{
			ID:   scenarioID,
			Name: "test-scenario",
		}

		output := map[string]interface{}{"result": "with query params"}
		outputBson, _ := bson.Marshal(output)
		mockAPI := &domain.MockAPI{
			Output: outputBson,
		}

		accountIdPtr := "test-account"
		accountScenarioRepo.EXPECT().
			GetActiveScenario(gomock.Any(), "test-feature", &accountIdPtr).
			Return(accountScenario, nil)

		scenarioRepo.EXPECT().
			GetByObjectID(gomock.Any(), scenarioID).
			Return(scenario, nil)

		// The path should include sorted query parameters
		mockAPIRepo.EXPECT().
			FindByFeatureScenarioPathMethodAndHash(
				gomock.Any(),
				"test-feature",
				"test-scenario",
				gomock.Any(), // Path will include query params
				"POST",
				gomock.Any(),
			).
			Return(mockAPI, nil)

		err := uc.ResponseMockData(c)
		assert.NoError(t, err)
	})
}

func TestForwardUC_WithCustomHeaders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIRepo := mocks.NewMockIMockAPIRepository(ctrl)
	scenarioRepo := mocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := mocks.NewMockIAccountScenarioRepository(ctrl)

	uc := NewForwardUC(mockAPIRepo, scenarioRepo, accountScenarioRepo)

	t.Run("response with custom headers", func(t *testing.T) {
		body := map[string]interface{}{"field1": "value1"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/forward/api/v1/test", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Account-Id", "test-account")
		req.Header.Set("X-Feature-Name", "test-feature")

		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		scenarioID := primitive.NewObjectID()
		accountScenario := &domain.AccountScenario{
			ScenarioID: scenarioID,
		}
		scenario := &domain.Scenario{
			ID:   scenarioID,
			Name: "test-scenario",
		}

		output := map[string]interface{}{"result": "success"}
		outputBson, _ := bson.Marshal(output)

		headers := map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Request-Id":    "req-123",
		}
		headersBson, _ := bson.Marshal(headers)

		mockAPI := &domain.MockAPI{
			Output:  outputBson,
			Headers: headersBson,
		}

		accountIdPtr := "test-account"
		accountScenarioRepo.EXPECT().
			GetActiveScenario(gomock.Any(), "test-feature", &accountIdPtr).
			Return(accountScenario, nil)

		scenarioRepo.EXPECT().
			GetByObjectID(gomock.Any(), scenarioID).
			Return(scenario, nil)

		mockAPIRepo.EXPECT().
			FindByFeatureScenarioPathMethodAndHash(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(mockAPI, nil)

		err := uc.ResponseMockData(c)
		assert.NoError(t, err)

		// Verify custom headers are set
		assert.Equal(t, "custom-value", rec.Header().Get("X-Custom-Header"))
		assert.Equal(t, "req-123", rec.Header().Get("X-Request-Id"))
	})
}

func TestForwardUC_HeaderSanitization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIRepo := mocks.NewMockIMockAPIRepository(ctrl)
	scenarioRepo := mocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := mocks.NewMockIAccountScenarioRepository(ctrl)

	uc := NewForwardUC(mockAPIRepo, scenarioRepo, accountScenarioRepo)

	tests := []struct {
		name           string
		headers        map[string]string
		expectedSet    map[string]string
		expectedBlocked []string
	}{
		{
			name: "dangerous Set-Cookie header blocked",
			headers: map[string]string{
				"Set-Cookie":   "admin=true",
				"Content-Type": "application/json",
				"X-Request-ID": "123",
			},
			expectedSet: map[string]string{
				"Content-Type": "application/json",
				"X-Request-ID": "123",
			},
			expectedBlocked: []string{"Set-Cookie"},
		},
		{
			name: "XSS in header value sanitized",
			headers: map[string]string{
				"X-Custom-Header": "<script>alert('xss')</script>",
			},
			expectedSet: map[string]string{
				"X-Custom-Header": "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
			},
			expectedBlocked: []string{},
		},
		{
			name: "CRLF injection blocked",
			headers: map[string]string{
				"X-Request-ID": "test\r\nSet-Cookie: evil=true",
			},
			expectedSet: map[string]string{
				"X-Request-ID": "testSet-Cookie: evil=true", // CRLF removed
			},
			expectedBlocked: []string{},
		},
		{
			name: "safe headers pass through",
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "no-cache",
				"X-Request-ID":  "abc123",
			},
			expectedSet: map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "no-cache",
				"X-Request-ID":  "abc123",
			},
			expectedBlocked: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{"field1": "value1"}
			bodyBytes, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/forward/api/v1/test", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Account-Id", "test-account")
			req.Header.Set("X-Feature-Name", "test-feature")

			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			scenarioID := primitive.NewObjectID()
			accountScenario := &domain.AccountScenario{
				ScenarioID: scenarioID,
			}
			scenario := &domain.Scenario{
				ID:   scenarioID,
				Name: "test-scenario",
			}

			output := map[string]interface{}{"result": "success"}
			outputBson, _ := bson.Marshal(output)

			headersBson, _ := bson.Marshal(tt.headers)

			mockAPI := &domain.MockAPI{
				Output:  outputBson,
				Headers: headersBson,
			}

			accountIdPtr := "test-account"
			accountScenarioRepo.EXPECT().
				GetActiveScenario(gomock.Any(), "test-feature", &accountIdPtr).
				Return(accountScenario, nil)

			scenarioRepo.EXPECT().
				GetByObjectID(gomock.Any(), scenarioID).
				Return(scenario, nil)

			mockAPIRepo.EXPECT().
				FindByFeatureScenarioPathMethodAndHash(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).
				Return(mockAPI, nil)

			err := uc.ResponseMockData(c)
			assert.NoError(t, err)

			// Verify expected headers are set
			for key, expectedValue := range tt.expectedSet {
				actualValue := rec.Header().Get(key)
				assert.Equal(t, expectedValue, actualValue, "Header %s should match", key)
			}

			// Verify blocked headers are not set
			for _, blockedHeader := range tt.expectedBlocked {
				assert.Empty(t, rec.Header().Get(blockedHeader), "Header %s should be blocked", blockedHeader)
			}
		})
	}
}

func TestNewForwardUC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIRepo := mocks.NewMockIMockAPIRepository(ctrl)
	scenarioRepo := mocks.NewMockIScenarioRepository(ctrl)
	accountScenarioRepo := mocks.NewMockIAccountScenarioRepository(ctrl)

	uc := NewForwardUC(mockAPIRepo, scenarioRepo, accountScenarioRepo)

	assert.NotNil(t, uc)
	assert.Implements(t, (*IForwardUC)(nil), uc)
}
