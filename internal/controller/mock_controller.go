package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/internal/usecase"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IMockController interface {
	StartHttpServer() error
}

type MockController struct {
	config       *configs.Config
	FeatureRepo  repository.IFeatureRepository
	ScenarioRepo repository.IScenarioRepository
	MockAPIRepo  repository.IMockAPIRepository
	forwardUc    usecase.IForwardUC
	trie         usecase.ITrie
}

func NewMockController(
	config *configs.Config,
	featureRepo repository.IFeatureRepository,
	scenarioRepo repository.IScenarioRepository,
	mockAPIRepo repository.IMockAPIRepository,
	forwardUc usecase.IForwardUC,
	trie usecase.ITrie,
) IMockController {

	return &MockController{
		config:       config,
		FeatureRepo:  featureRepo,
		ScenarioRepo: scenarioRepo,
		MockAPIRepo:  mockAPIRepo,
		forwardUc:    forwardUc,
		trie:         trie,
	}
}

func (_self *MockController) StartHttpServer() error {
	c := echo.New()
	// Middleware
	c.Use(middleware.CORS())          // enable CORS for web interface
	c.Use(middleware.RequestLogger()) // use the default RequestLogger middleware with slog logger
	c.Use(middleware.Recover())       // recover panics as errors for proper error handling
	// Routes
	v1 := c.Group("/api/v1/mocktool")
	v1.GET("/features", _self.GetFeatures)               // list all features
	v1.POST("/features", _self.CreateNewFeature)         // create new feature
	v1.PUT("/features/:feature_id", _self.UpdateFeature) // update or inactive

	v1.GET("/scenarios", _self.ListScenarioByFeature)                // list all scenarios by feature
	v1.POST("/scenarios", _self.CreateNewScenarioByFeature)          // create new scenario
	v1.PUT("/scenarios/:scenario_id", _self.UpdateScenarioByFeature) // update or inactive scenario

	v1.GET("/mockapis", _self.ListMockAPIsByScenario)          // list all APIs by scenario
	v1.POST("/mockapis", _self.CreateMockAPIByScenario)        // create new scenario
	v1.PUT("/mockapis/:api_id", _self.UpdateMockAPIByScenario) // update or inactive scenario

	c.GET("/forward/*", _self.responseMockData)
	c.POST("/forward/*", _self.responseMockData)
	c.PUT("/forward/*", _self.responseMockData)
	c.DELETE("/forward/*", _self.responseMockData)

	if err := c.Start(_self.config.AppConfig.HTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
		return err
	}
	return nil
}

// handler
func (_self *MockController) responseMockData(c echo.Context) error {
	return _self.forwardUc.ResponseMockData(c)
}

/* ---------- GET /features ---------- */

func (_self *MockController) GetFeatures(c echo.Context) error {
	ctx := c.Request().Context()

	features, err := _self.FeatureRepo.ListAll(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, features)
}

/* ---------- POST /features ---------- */

func (_self *MockController) CreateNewFeature(c echo.Context) error {
	ctx := c.Request().Context()

	var req domain.Feature
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.IsActive = true

	if err := _self.FeatureRepo.Create(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	return c.JSON(http.StatusCreated, req)
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

func (_self *MockController) ListScenarioByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	featureName := c.QueryParam("feature_name")
	if featureName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "feature_name is required")
	}

	scenarios, err := _self.ScenarioRepo.ListByFeatureName(ctx, featureName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, scenarios)
}

/* ---------- POST /scenarios ---------- */

func (_self *MockController) CreateNewScenarioByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	var req domain.Scenario
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.IsActive = true

	// Get all active scenarios for this feature before deactivating
	activeScenarios, err := _self.ScenarioRepo.ListByFeatureName(ctx, req.FeatureName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get active scenarios: "+err.Error())
	}

	// Deactivate all existing active scenarios for this feature
	if err := _self.ScenarioRepo.UpdateByFilter(ctx,
		bson.M{"feature_name": req.FeatureName, "is_active": true},
		bson.M{"is_active": false, "updated_at": now},
	); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to deactivate existing scenarios: "+err.Error())
	}

	// Remove deactivated scenarios from trie
	for _, scenario := range activeScenarios {
		if scenario.IsActive {
			_self.trie.RemoveScenario(req.FeatureName, scenario.Name)
		}
	}

	if err := _self.ScenarioRepo.Create(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	return c.JSON(http.StatusCreated, req)
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

	// First get the scenario to know its feature_name
	scenario, err := _self.ScenarioRepo.GetByObjectID(ctx, objectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scenario not found")
	}

	// If activating this scenario, deactivate all other scenarios for the same feature
	if req.IsActive {
		// Get all active scenarios for this feature before deactivating
		activeScenarios, err := _self.ScenarioRepo.ListByFeatureName(ctx, scenario.FeatureName)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get active scenarios: "+err.Error())
		}

		// Deactivate all other active scenarios for this feature
		now := time.Now().UTC()
		if err := _self.ScenarioRepo.UpdateByFilter(ctx,
			bson.M{"feature_name": scenario.FeatureName, "is_active": true, "_id": bson.M{"$ne": objectID}},
			bson.M{"is_active": false, "updated_at": now},
		); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to deactivate existing scenarios: "+err.Error())
		}

		// Remove deactivated scenarios from trie (except the one being activated)
		for _, s := range activeScenarios {
			if s.IsActive && s.ID != objectID {
				_self.trie.RemoveScenario(scenario.FeatureName, s.Name)
			}
		}

		// Load the activated scenario's APIs into the trie
		apis, err := _self.MockAPIRepo.ListByScenarioName(ctx, scenario.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get scenario APIs: "+err.Error())
		}

		for _, api := range apis {
			if api.IsActive {
				if err := _self.trie.Insert(entity.APIRequest{
					FeatureName: api.FeatureName,
					Scenario:    api.ScenarioName,
					Path:        api.Path,
					Method:      api.Method,
					HashInput:   api.HashInput,
					Output:      api.Output,
				}); err != nil {
					slog.Error("failed to insert API into trie", "error", err, "path", api.Path)
				}
			}
		}
	} else {
		// If deactivating this scenario, remove it from trie
		_self.trie.RemoveScenario(scenario.FeatureName, scenario.Name)
	}

	update := req.ToMap()
	if err := _self.ScenarioRepo.UpdateByObjectID(ctx, objectID, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

/* ---------- GET /mockapis?scenario_name= ---------- */

func (_self *MockController) ListMockAPIsByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	scenarioName := c.QueryParam("scenario_name")
	if scenarioName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "scenario_name is required")
	}

	apis, err := _self.MockAPIRepo.ListByScenarioName(ctx, scenarioName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Convert to response format with JSON objects for frontend
	var response []map[string]any
	for _, api := range apis {
		// Convert bson.Raw to JSON object
		var hashInputJSON any = nil
		if len(api.HashInput) > 0 {
			var result bson.M
			if err := bson.Unmarshal(api.HashInput, &result); err == nil {
				hashInputJSON = result
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

		response = append(response, map[string]any{
			"id":            api.ID.Hex(),
			"feature_name":  api.FeatureName,
			"scenario_name": api.ScenarioName,
			"name":          api.Name,
			"description":   api.Description,
			"is_active":     api.IsActive,
			"path":          api.Path,
			"method":        api.Method,
			"hash_input":    hashInputJSON,
			"output":        outputJSON,
			"created_at":    api.CreatedAt.Format(time.RFC3339),
			"updated_at":    api.UpdatedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusOK, response)
}

/* ---------- POST /mockapis ---------- */

func (_self *MockController) CreateMockAPIByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	// Use a temporary struct for binding with json.RawMessage
	var reqBody struct {
		FeatureName  string          `json:"feature_name"`
		ScenarioName string          `json:"scenario_name"`
		Name         string          `json:"name"`
		Description  string          `json:"description"`
		Path         string          `json:"path"`
		Method       string          `json:"method"`
		HashInput    json.RawMessage `json:"hash_input"`
		Output       json.RawMessage `json:"output"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
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

	// Process hash input - store original JSON and compute hash
	// Only process if not empty and not null
	var inputData any
	if err := json.Unmarshal(reqBody.HashInput, &inputData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid output JSON: "+err.Error())
	}

	// If the output is a string, try to parse it as JSON
	if intputStr, ok := inputData.(string); ok {
		var parsedData any
		if err := json.Unmarshal([]byte(intputStr), &parsedData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid nested JSON in output: "+err.Error())
		}
		inputData = parsedData
	}

	inputBsonData, err := bson.Marshal(inputData)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert output to BSON: "+err.Error())
	}
	req.HashInput = inputBsonData

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

	if err := _self.MockAPIRepo.Create(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	if err := _self.trie.Insert(entity.APIRequest{
		FeatureName: req.FeatureName,
		Scenario:    req.ScenarioName,
		Path:        req.Path,
		Method:      req.Method,
		HashInput:   req.HashInput,
		Output:      req.Output,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to insert into trie: "+err.Error())
	}

	// Convert response to JSON-friendly format for frontend
	var hashInputJSON any
	if len(req.HashInput) > 0 {
		var result bson.M
		if err := bson.Unmarshal(req.HashInput, &result); err == nil {
			hashInputJSON = result
		}
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
		"hash_input":    hashInputJSON,
		"output":        outputJSON,
		"created_at":    req.CreatedAt.Format(time.RFC3339),
		"updated_at":    req.UpdatedAt.Format(time.RFC3339),
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
		HashInput    json.RawMessage `json:"hash_input"`
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

	// Process hash input if provided
	// Only process if not empty and not null
	var inputData any
	if err := json.Unmarshal(reqBody.HashInput, &inputData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid output JSON: "+err.Error())
	}

	// If the output is a string, try to parse it as JSON
	if intputStr, ok := inputData.(string); ok {
		var parsedData any
		if err := json.Unmarshal([]byte(intputStr), &parsedData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid nested JSON in output: "+err.Error())
		}
		inputData = parsedData
	}

	inputBsonData, err := bson.Marshal(inputData)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert output to BSON: "+err.Error())
	}
	var hashInputJSON any
	if len(inputBsonData) > 0 {
		var result bson.M
		if err := bson.Unmarshal(inputBsonData, &result); err == nil {
			hashInputJSON = result
		}
	}

	update["hash_input"] = hashInputJSON

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

	update["is_active"] = reqBody.IsActive
	update["updated_at"] = time.Now().UTC()

	if err := _self.MockAPIRepo.UpdateByObjectID(ctx, objectID, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}
