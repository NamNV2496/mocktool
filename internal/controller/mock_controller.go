package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/pkg/observability"
	"github.com/namnv2496/mocktool/pkg/security"
	"github.com/namnv2496/mocktool/pkg/utils"
	customValidator "github.com/namnv2496/mocktool/pkg/validator"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IMockController interface {
	StartHttpServer() error
}

type MockController struct {
	config              *configs.Config
	FeatureRepo         repository.IFeatureRepository
	ScenarioRepo        repository.IScenarioRepository
	AccountScenarioRepo repository.IAccountScenarioRepository
	MockAPIRepo         repository.IMockAPIRepository
	loadTestController  ILoadTestController
}

func NewMockController(
	config *configs.Config,
	featureRepo repository.IFeatureRepository,
	scenarioRepo repository.IScenarioRepository,
	accountScenarioRepo repository.IAccountScenarioRepository,
	mockAPIRepo repository.IMockAPIRepository,
	loadTestController ILoadTestController,
) IMockController {

	return &MockController{
		config:              config,
		FeatureRepo:         featureRepo,
		ScenarioRepo:        scenarioRepo,
		AccountScenarioRepo: accountScenarioRepo,
		MockAPIRepo:         mockAPIRepo,
		loadTestController:  loadTestController,
	}
}

func (_self *MockController) StartHttpServer() error {
	c := echo.New()
	c.Validator = customValidator.NewValidator()

	cleanup, err := observability.InitTracing("mocktool", "1.0.0")
	if err != nil {
		slog.Error("Failed to initialize tracing", "error", err)
	} else {
		defer cleanup()
		slog.Info("OpenTelemetry tracing initialized")
	}

	observability.AppInfo.WithLabelValues("1.0.0", runtime.Version()).Set(1)

	startTime := time.Now()
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			observability.UptimeSeconds.Set(time.Since(startTime).Seconds())
		}
	}()

	c.Use(observability.TracingMiddleware()) // OpenTelemetry tracing
	c.Use(observability.MetricsMiddleware()) // Prometheus metrics
	c.Use(middleware.CORS())                 // enable CORS for web interface
	c.Use(middleware.RequestLogger())        // use the default RequestLogger middleware with slog logger
	c.Use(middleware.Recover())              // recover panics as errors for proper error handling

	// Health check endpoints
	c.GET("/health", _self.HealthCheck)
	c.GET("/ready", _self.ReadinessCheck)
	c.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Routes
	v1 := c.Group("/api/v1/mocktool")
	v1.GET("/features", _self.GetFeatures)                 // list all features
	v1.GET("/features/search", _self.SearchFeaturesByName) // list all features has name likely
	v1.POST("/features", _self.CreateNewFeature)           // create new feature
	v1.PUT("/features/:feature_id", _self.UpdateFeature)   // update or inactive

	v1.GET("/scenarios", _self.ListScenariosByFeature)                        // list all scenarios by feature
	v1.GET("/scenarios/search", _self.SearchScenariosByFeatureAndName)        // list all scenarios by feature has name likely
	v1.GET("/scenarios/active", _self.ListActiveScenariosByFeature)           // get active scenario for feature+account
	v1.POST("/scenarios", _self.CreateNewScenariosByFeature)                  // create new scenario
	v1.PUT("/scenarios/:scenario_id", _self.UpdateScenarioByFeature)          // update scenario
	v1.POST("/scenarios/:scenario_id/activate", _self.ActivateScenario)       // activate scenario for account
	v1.DELETE("/scenarios/:scenario_id/deactivate", _self.DeactivateScenario) // deactivate scenario for account

	v1.GET("/mockapis", _self.ListMockAPIsByScenario)                       // list all APIs by scenario
	v1.GET("/mockapis/search", _self.SearchMockAPIsByScenarioAndNameOrPath) // search APIs by scenario and name/path
	v1.POST("/mockapis", _self.CreateMockAPIByScenario)                     // create new scenario
	v1.PUT("/mockapis/:api_id", _self.UpdateMockAPIByScenario)              // update or inactive scenario

	// Load test scenarios - delegate to LoadTestController
	_self.loadTestController.RegisterRoutes(v1)
	if err := c.Start(_self.config.AppConfig.HTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
		return err
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.Shutdown(ctx)
}

/* ---------- GET /features ---------- */

func (_self *MockController) SearchFeaturesByName(c echo.Context) error {
	ctx := c.Request().Context()

	// Get search query from query parameter
	query := c.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "search query 'q' is required")
	}

	params := parsePaginationParams(c)

	features, total, err := _self.FeatureRepo.SearchByName(ctx, query, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	response := domain.NewPaginatedResponse(features, total, params)
	return c.JSON(http.StatusOK, response)
}

func (_self *MockController) GetFeatures(c echo.Context) error {
	ctx := c.Request().Context()

	params := parsePaginationParams(c)

	features, total, err := _self.FeatureRepo.ListAllPaginated(ctx, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	response := domain.NewPaginatedResponse(features, total, params)
	return c.JSON(http.StatusOK, response)
}

// parsePaginationParams extracts and validates pagination parameters from query string
func parsePaginationParams(c echo.Context) domain.PaginationParams {
	params := domain.PaginationParams{
		Page:     domain.DefaultPage,
		PageSize: domain.DefaultPageSize,
	}

	if pageStr := c.QueryParam("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			params.Page = page
		}
	}

	if pageSizeStr := c.QueryParam("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			params.PageSize = pageSize
		}
	}

	params.Normalize()
	return params
}

/* ---------- POST /features ---------- */

func (_self *MockController) CreateNewFeature(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		Name        string `json:"name" validate:"required,no_spaces"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate the request
	if err := c.Validate(&req); err != nil {
		return err
	}

	feature := &domain.Feature{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := _self.FeatureRepo.Create(ctx, feature); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	return c.JSON(http.StatusCreated, feature)
}

/* ---------- PUT /features/:feature_id ---------- */

func (_self *MockController) UpdateFeature(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("feature_id")
	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid feature_id")
	}

	var req domain.Feature

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	update := bson.M{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	update["is_active"] = req.IsActive
	update["updated_at"] = time.Now().UTC()

	if len(update) == 0 {
		return c.NoContent(http.StatusNoContent)
	}

	if err := _self.FeatureRepo.UpdateByObjectID(ctx, objectID, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

/* ---------- GET /scenarios?feature_name= ---------- */

func (_self *MockController) SearchScenariosByFeatureAndName(c echo.Context) error {
	ctx := c.Request().Context()

	// Get search query from query parameter
	query := c.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "search query 'q' is required")
	}

	// Feature name is optional - if provided, will filter by feature
	featureName := c.QueryParam("feature_name")

	params := parsePaginationParams(c)

	scenarios, total, err := _self.ScenarioRepo.SearchByFeatureAndName(ctx, featureName, query, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	response := domain.NewPaginatedResponse(scenarios, total, params)
	return c.JSON(http.StatusOK, response)
}
func (_self *MockController) ListScenariosByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	featureName := c.QueryParam("feature_name")
	if featureName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "feature_name is required")
	}
	var resp []domain.Scenario
	var globalScenarioId primitive.ObjectID
	// always put global active scenario in first line of first page
	pagination := parsePaginationParams(c)
	if pagination.Page == 1 {
		globalScenarioActive, err := _self.AccountScenarioRepo.GetActiveScenario(ctx, featureName, nil)
		if err == nil {
			// has global scenario
			// Get the actual scenario details
			globalScenario, err := _self.ScenarioRepo.GetByObjectID(ctx, globalScenarioActive.ScenarioID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			if globalScenario != nil {
				globalScenarioId = globalScenario.ID
				resp = append(resp, domain.Scenario{
					ID:          globalScenario.ID,
					FeatureName: globalScenario.FeatureName,
					Name:        globalScenario.Name,
					Description: globalScenario.Description,
					CreatedAt:   globalScenario.CreatedAt,
					UpdatedAt:   globalScenario.UpdatedAt,
				})
			}
		}
	}
	scenarios, total, err := _self.ScenarioRepo.ListByFeatureNamePaginated(ctx, featureName, pagination)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for _, scenario := range scenarios {
		if scenario.ID == globalScenarioId {
			continue
		}
		resp = append(resp, scenario)
	}

	response := domain.NewPaginatedResponse(resp, total, pagination)
	return c.JSON(http.StatusOK, response)
}

/* ---------- GET /scenarios?feature_name= ---------- */

func (_self *MockController) ListActiveScenariosByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	featureName := c.QueryParam("feature_name")
	if featureName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "feature_name is required")
	}

	// Extract accountId from header - OPTIONAL
	// If not provided, will fetch global active scenario
	var accountId *string
	accountIdHeader := c.Request().Header.Get("X-Account-Id")
	if accountIdHeader != "" {
		accountId = &accountIdHeader
	}

	// Get active scenario mapping for this feature and account (or global if accountId is nil)
	accountScenario, err := _self.AccountScenarioRepo.GetActiveScenario(ctx, featureName, accountId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Get the actual scenario details
	scenario, err := _self.ScenarioRepo.GetByObjectID(ctx, accountScenario.ScenarioID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, scenario)
}

/* ---------- POST /scenarios ---------- */

func (_self *MockController) CreateNewScenariosByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		FeatureName string `json:"feature_name" validate:"required,no_spaces"`
		Name        string `json:"name" validate:"required,no_spaces"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate the request
	if err := c.Validate(&req); err != nil {
		return err
	}

	scenario := &domain.Scenario{
		FeatureName: req.FeatureName,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Just create the scenario - activation is handled separately via AccountScenario
	if err := _self.ScenarioRepo.Create(ctx, scenario); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	return c.JSON(http.StatusCreated, scenario)
}

/* ---------- PUT /scenarios/:scenario_id ---------- */

func (_self *MockController) UpdateScenarioByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("scenario_id")
	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	var req domain.Scenario
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Just update the scenario details - activation is handled separately via AccountScenario
	update := req.ToMap()
	if err := _self.ScenarioRepo.UpdateByObjectID(ctx, objectID, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

/* ---------- POST /scenarios/:scenario_id/activate ---------- */

func (_self *MockController) ActivateScenario(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("scenario_id")
	scenarioID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	// Get the scenario to know its feature
	scenario, err := _self.ScenarioRepo.GetByObjectID(ctx, scenarioID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scenario not found")
	}

	// Extract accountId from query or default to global
	var accountId *string
	accountIdParam := c.QueryParam("account_id")
	if accountIdParam != "" {
		accountId = &accountIdParam
	}

	// If activating globally (accountId is nil), remove ALL account-specific mappings for this feature
	if accountId == nil {
		// Delete all account-specific mappings
		if err := _self.AccountScenarioRepo.DeactivateAllAccountSpecificMappings(ctx, scenario.FeatureName); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to deactivate account-specific scenarios: "+err.Error())
		}
	}

	// Deactivate existing active scenario for this feature+account
	if err := _self.AccountScenarioRepo.DeactivateByFeatureAndAccount(ctx, scenario.FeatureName, accountId); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to deactivate existing scenario: "+err.Error())
	}

	// Create new AccountScenario mapping
	now := time.Now().UTC()
	accountScenario := &domain.AccountScenario{
		FeatureName: scenario.FeatureName,
		ScenarioID:  scenarioID,
		AccountId:   accountId,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := _self.AccountScenarioRepo.Create(ctx, accountScenario); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to activate scenario: "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "scenario activated successfully"})
}

/* ---------- DELETE /scenarios/:scenario_id/deactivate ---------- */

func (_self *MockController) DeactivateScenario(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("scenario_id")
	scenarioID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	// Get the scenario to know its feature
	scenario, err := _self.ScenarioRepo.GetByObjectID(ctx, scenarioID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scenario not found")
	}

	// Extract accountId from query or default to global
	var accountId *string
	accountIdParam := c.QueryParam("account_id")
	if accountIdParam != "" {
		accountId = &accountIdParam
	}

	// Deactivate the scenario for this account
	if err := _self.AccountScenarioRepo.DeactivateByFeatureAndAccount(ctx, scenario.FeatureName, accountId); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to deactivate scenario: "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "scenario deactivated successfully"})
}

/* ---------- GET /mockapis?scenario_name= ---------- */

func (_self *MockController) ListMockAPIsByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	scenarioName := c.QueryParam("scenario_name")
	if scenarioName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "scenario_name is required")
	}

	params := parsePaginationParams(c)

	apis, total, err := _self.MockAPIRepo.ListByScenarioNamePaginated(ctx, scenarioName, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Convert to response format with JSON objects for frontend
	var data []map[string]any
	for _, api := range apis {
		// Convert bson.Raw to JSON object for input
		var inputJSON any = nil
		if len(api.Input) > 0 {
			var result bson.M
			if err := bson.Unmarshal(api.Input, &result); err == nil {
				inputJSON = result
			}
		}

		// Convert bson.Raw to JSON object
		var outputJSON any = nil
		if len(api.Output) > 0 {
			var result bson.M
			if err := bson.Unmarshal(api.Output, &result); err == nil {
				outputJSON = result
			}
		}

		// Convert bson.Raw to JSON object
		var headers any = nil
		if len(api.Headers) > 0 {
			var result bson.M
			if err := bson.Unmarshal(api.Headers, &result); err == nil {
				headers = result
			}
		}

		data = append(data, map[string]any{
			"id":            api.ID.Hex(),
			"feature_name":  api.FeatureName,
			"scenario_name": api.ScenarioName,
			"name":          api.Name,
			"description":   api.Description,
			"is_active":     api.IsActive,
			"path":          api.Path,
			"method":        api.Method,
			"input":         inputJSON,
			// "hash_input":    api.HashInput,
			"output":     outputJSON,
			"headers":    headers,
			"created_at": api.CreatedAt.Format(time.RFC3339),
			"updated_at": api.UpdatedAt.Format(time.RFC3339),
		})
	}

	// Calculate total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	response := map[string]any{
		"data":        data,
		"total":       total,
		"page":        params.Page,
		"page_size":   params.PageSize,
		"total_pages": totalPages,
	}

	return c.JSON(http.StatusOK, response)
}

/* ---------- GET /mockapis/search ---------- */

func (_self *MockController) SearchMockAPIsByScenarioAndNameOrPath(c echo.Context) error {
	ctx := c.Request().Context()

	scenarioName := c.QueryParam("scenario_name")
	if scenarioName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "scenario_name is required")
	}

	query := c.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "search query 'q' is required")
	}

	params := parsePaginationParams(c)

	apis, total, err := _self.MockAPIRepo.SearchByScenarioAndNameOrPath(ctx, scenarioName, query, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Convert to response format with JSON objects for frontend
	var data []map[string]any
	for _, api := range apis {
		// Convert bson.Raw to JSON object for input
		var inputJSON any = nil
		if len(api.Input) > 0 {
			var result bson.M
			if err := bson.Unmarshal(api.Input, &result); err == nil {
				inputJSON = result
			}
		}

		// Convert bson.Raw to JSON object
		var outputJSON any = nil
		if len(api.Output) > 0 {
			var result bson.M
			if err := bson.Unmarshal(api.Output, &result); err == nil {
				outputJSON = result
			}
		}

		// Convert bson.Raw to JSON object
		var headers any = nil
		if len(api.Headers) > 0 {
			var result bson.M
			if err := bson.Unmarshal(api.Headers, &result); err == nil {
				headers = result
			}
		}

		data = append(data, map[string]any{
			"id":            api.ID.Hex(),
			"feature_name":  api.FeatureName,
			"scenario_name": api.ScenarioName,
			"name":          api.Name,
			"description":   api.Description,
			"path":          api.Path,
			"method":        api.Method,
			"input":         inputJSON,
			"hash_input":    api.HashInput,
			"headers":       headers,
			"output":        outputJSON,
			"is_active":     api.IsActive,
			"created_at":    api.CreatedAt,
		})
	}

	// Calculate total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	response := map[string]any{
		"data":        data,
		"total":       total,
		"page":        params.Page,
		"page_size":   params.PageSize,
		"total_pages": totalPages,
	}

	return c.JSON(http.StatusOK, response)
}

/* ---------- POST /mockapis ---------- */

func (_self *MockController) CreateMockAPIByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	// Use a temporary struct for binding with json.RawMessage
	var reqBody struct {
		FeatureName  string          `json:"feature_name" validate:"required,no_spaces"`
		ScenarioName string          `json:"scenario_name" validate:"required,no_spaces"`
		Name         string          `json:"name" validate:"required,no_spaces"`
		Description  string          `json:"description"`
		Path         string          `json:"path" validate:"required,no_spaces"`
		Method       string          `json:"method" validate:"required,no_spaces"`
		Input        json.RawMessage `json:"input"`
		Headers      json.RawMessage `json:"headers"`
		Output       json.RawMessage `json:"output"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate the request
	if err := c.Validate(&reqBody); err != nil {
		return err // Already formatted as HTTPError by CustomValidator
	}

	// Convert to domain.MockAPI
	var req domain.MockAPI
	req.FeatureName = reqBody.FeatureName
	req.ScenarioName = reqBody.ScenarioName
	req.Name = reqBody.Name
	req.Description = reqBody.Description
	req.Path = reqBody.Path
	req.Method = reqBody.Method

	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.IsActive = true

	// Process input - store original JSON and compute hash
	if len(reqBody.Input) > 0 && string(reqBody.Input) != "null" {
		var inputData any
		if err := json.Unmarshal(reqBody.Input, &inputData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid input JSON: "+err.Error())
		}

		// If the input is a string, try to parse it as JSON
		if inputStr, ok := inputData.(string); ok {
			var parsedData any
			if err := json.Unmarshal([]byte(inputStr), &parsedData); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid nested JSON in input: "+err.Error())
			}
			inputData = parsedData
		}

		// Store original input as bson.Raw
		inputBsonData, err := bson.Marshal(inputData)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert input to BSON: "+err.Error())
		}
		req.Input = inputBsonData

		// Generate hash from sorted input
		req.HashInput = utils.GenerateHashFromInput(inputBsonData)
	}

	// Process output - required field
	if len(reqBody.Output) == 0 || string(reqBody.Output) == "null" || string(reqBody.Output) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "output is required")
	}

	// Convert json.RawMessage to bson.Raw
	var outputData any
	if err := json.Unmarshal(reqBody.Output, &outputData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid output JSON: "+err.Error())
	}

	// If the output is a string, try to parse it as JSON
	if outputStr, ok := outputData.(string); ok {
		var parsedData any
		if err := json.Unmarshal([]byte(outputStr), &parsedData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid nested JSON in output: "+err.Error())
		}
		outputData = parsedData
	}

	bsonData, err := bson.Marshal(outputData)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert output to BSON: "+err.Error())
	}
	req.Output = bsonData
	// headers
	if len(reqBody.Headers) != 0 {
		// Step 1: json.RawMessage -> string
		var headersStr string
		if err := json.Unmarshal(reqBody.Headers, &headersStr); err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"headers must be a JSON string: "+err.Error(),
			)
		}

		// Step 2: normalize string to valid JSON object
		headersStr = strings.TrimSpace(headersStr)

		if !strings.HasPrefix(headersStr, "{") {
			headersStr = "{" + headersStr + "}"
		}

		// Step 3: string -> map
		var headersMap map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headersMap); err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"invalid headers JSON object: "+err.Error(),
			)
		}

		// Step 3.5: Validate and sanitize headers for security
		sanitizedHeaders, warnings := security.ValidateAndSanitizeHeaders(headersMap)
		if len(warnings) > 0 {
			slog.Warn("Headers sanitized or blocked", "warnings", warnings)
		}

		// Step 4: map -> bson.Raw
		headerData, err := bson.Marshal(sanitizedHeaders)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"failed to convert headers to BSON: "+err.Error(),
			)
		}

		req.Headers = bson.Raw(headerData)
	}

	// create
	if err := _self.MockAPIRepo.Create(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	// Convert response to JSON-friendly format for frontend
	var inputJSON any
	if err := json.Unmarshal(reqBody.Input, &inputJSON); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid input JSON: "+err.Error())
	}

	var outputJSON any
	if len(req.Output) > 0 {
		var result bson.M
		if err := bson.Unmarshal(req.Output, &result); err == nil {
			outputJSON = result
		}
	}

	response := map[string]any{
		"id":            req.ID.Hex(),
		"feature_name":  req.FeatureName,
		"scenario_name": req.ScenarioName,
		"name":          req.Name,
		"description":   req.Description,
		"is_active":     req.IsActive,
		"path":          req.Path,
		"method":        req.Method,
		"input":         inputJSON,
		// "hash_input":    req.HashInput,
		"output":     outputJSON,
		"headers":    req.Headers,
		"created_at": req.CreatedAt.Format(time.RFC3339),
		"updated_at": req.UpdatedAt.Format(time.RFC3339),
	}

	return c.JSON(http.StatusCreated, response)
}

/* ---------- PUT /mockapis/:api_id ---------- */

func (_self *MockController) UpdateMockAPIByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("api_id")
	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid api_id")
	}

	// Use a temporary struct for binding with json.RawMessage
	var reqBody struct {
		FeatureName  string          `json:"feature_name"`
		ScenarioName string          `json:"scenario_name"`
		Name         string          `json:"name"`
		Description  string          `json:"description"`
		Path         string          `json:"path"`
		Method       string          `json:"method"`
		Input        json.RawMessage `json:"input"`
		Headers      json.RawMessage `json:"headers"`
		Output       json.RawMessage `json:"output"`
		IsActive     bool            `json:"is_active"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	update := bson.M{}

	if reqBody.Name != "" {
		update["name"] = reqBody.Name
	}
	if reqBody.Description != "" {
		update["description"] = reqBody.Description
	}
	if reqBody.Path != "" {
		update["path"] = reqBody.Path
	}
	if reqBody.Method != "" {
		update["method"] = reqBody.Method
	}
	if reqBody.FeatureName != "" {
		update["feature_name"] = reqBody.FeatureName
	}
	if reqBody.ScenarioName != "" {
		update["scenario_name"] = reqBody.ScenarioName
	}

	// Process input if provided
	if len(reqBody.Input) > 0 && string(reqBody.Input) != "null" {
		var inputData any
		if err := json.Unmarshal(reqBody.Input, &inputData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid input JSON: "+err.Error())
		}

		// If the input is a string, try to parse it as JSON
		if inputStr, ok := inputData.(string); ok {
			var parsedData any
			if err := json.Unmarshal([]byte(inputStr), &parsedData); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid nested JSON in input: "+err.Error())
			}
			inputData = parsedData
		}

		inputBsonData, err := bson.Marshal(inputData)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert input to BSON: "+err.Error())
		}
		// Store original input
		update["input"] = inputData

		// Generate and store hash
		update["hash_input"] = utils.GenerateHashFromInput(inputBsonData)
	}

	// Process output if provided
	// Only process if not empty and not null
	if len(reqBody.Output) == 0 || string(reqBody.Output) == "null" || string(reqBody.Output) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "output is required")
	}

	// Convert json.RawMessage to bson.Raw
	var outputData any
	if err := json.Unmarshal(reqBody.Output, &outputData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid output JSON: "+err.Error())
	}

	// If the output is a string, try to parse it as JSON
	if outputStr, ok := outputData.(string); ok {
		var parsedData any
		if err := json.Unmarshal([]byte(outputStr), &parsedData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid nested JSON in output: "+err.Error())
		}
		outputData = parsedData
	}

	bsonData, err := bson.Marshal(outputData)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert output to BSON: "+err.Error())
	}
	var outputJSON any
	if len(bsonData) > 0 {
		var result bson.M
		if err := bson.Unmarshal(bsonData, &result); err == nil {
			outputJSON = result
		}
	}
	update["output"] = outputJSON
	// headers
	if len(reqBody.Headers) != 0 {
		// Step 1: json.RawMessage -> string
		var headersStr string
		if err := json.Unmarshal(reqBody.Headers, &headersStr); err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"headers must be a JSON string: "+err.Error(),
			)
		}

		// Step 2: normalize string to valid JSON object
		headersStr = strings.TrimSpace(headersStr)

		if !strings.HasPrefix(headersStr, "{") {
			headersStr = "{" + headersStr + "}"
		}

		// Step 3: string -> map
		var headersMap map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headersMap); err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"invalid headers JSON object: "+err.Error(),
			)
		}

		// Step 3.5: Validate and sanitize headers for security
		sanitizedHeaders, warnings := security.ValidateAndSanitizeHeaders(headersMap)
		if len(warnings) > 0 {
			slog.Warn("Headers sanitized or blocked", "warnings", warnings)
		}

		// Step 4: map -> bson.Raw
		headerData, err := bson.Marshal(sanitizedHeaders)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"failed to convert headers to BSON: "+err.Error(),
			)
		}

		update["headers"] = bson.Raw(headerData)
	}
	update["is_active"] = reqBody.IsActive
	update["updated_at"] = time.Now().UTC()

	if err := _self.MockAPIRepo.UpdateByObjectID(ctx, objectID, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

/* ---------- Health Check Endpoints ---------- */

// HealthCheck returns the basic health status of the application
func (_self *MockController) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "mocktool",
	})
}

// ReadinessCheck verifies that the application is ready to serve requests
// This includes checking database connectivity
func (_self *MockController) ReadinessCheck(c echo.Context) error {
	ctx := c.Request().Context()

	// Try to ping the database through a simple query
	// We'll just try to list features with a limit to check connectivity
	_, _, err := _self.FeatureRepo.ListAllPaginated(ctx, domain.PaginationParams{
		Page:     1,
		PageSize: 1,
	})

	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":  "not ready",
			"service": "mocktool",
			"error":   "database connection failed",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "ready",
		"service":  "mocktool",
		"database": "connected",
	})
}
