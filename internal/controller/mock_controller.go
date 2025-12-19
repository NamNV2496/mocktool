package controller

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/internal/usecase"
	"go.mongodb.org/mongo-driver/bson"
)

type IMockController interface {
	StartHttpServer() error
}

type MockController struct {
	config       *configs.Config
	FeatureRepo  repository.IFeatureRepository
	ScenarioRepo repository.IScenarioRepository
	MockAPIRepo  repository.IMockAPIRepository
	trie         usecase.ITrie
}

func NewMockController(
	config *configs.Config,
	featureRepo repository.IFeatureRepository,
	scenarioRepo repository.IScenarioRepository,
	mockAPIRepo repository.IMockAPIRepository,
	trie usecase.ITrie,
) IMockController {

	return &MockController{
		config:       config,
		FeatureRepo:  featureRepo,
		ScenarioRepo: scenarioRepo,
		MockAPIRepo:  mockAPIRepo,
		trie:         trie,
	}
}

func (_self *MockController) StartHttpServer() error {
	c := echo.New()
	// Middleware
	c.Use(middleware.RequestLogger()) // use the default RequestLogger middleware with slog logger
	c.Use(middleware.Recover())       // recover panics as errors for proper error handling
	// Routes
	v1 := c.Group("/api/v1/mocktool")
	v1.GET("/features", _self.GetFeatures)                // list all features
	v1.POST("/features", _self.CreateNewFeature)          // create new feature
	v1.PUT("/features/{feature_id}", _self.UpdateFeature) // update or inactive

	v1.GET("/scenarios", _self.ListScenarioByFeature)                 // list all scenarios by feature
	v1.POST("/scenarios", _self.CreateNewScenarioByFeature)           // create new scenario
	v1.PUT("/scenarios/{scenario_id}", _self.UpdateScenarioByFeature) // update or inactive scenario

	v1.GET("/mockapis", _self.ListMockAPIsByScenario)           // list all APIs by scenario
	v1.POST("/mockapis", _self.CreateMockAPIByScenario)         // create new scenario
	v1.PUT("/mockapis/{api_id}", _self.UpdateMockAPIByScenario) // update or inactive scenario

	c.GET("/forward/*", _self.responseMockData)

	if err := c.Start(_self.config.AppConfig.HTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
		return err
	}
	return nil
}

// handler
func (_self *MockController) responseMockData(c echo.Context) error {
	var request entity.APIRequest
	if err := c.Bind(&request); err != nil {
		return err
	}
	request.Path = c.Request().RequestURI
	// Bind query parameters
	request.FeatureName = c.Param("feature_name")
	response := _self.trie.Search(request)
	if response == nil {
		io.Copy(c.Response().Writer, strings.NewReader("not found"))
		return nil
	}
	output, ok := response.Output.(string)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid response output type")
	}
	io.Copy(c.Response().Writer, strings.NewReader(output))
	return nil
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

	id, err := strconv.ParseInt(c.Param("feature_id"), 10, 64)
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

	if len(update) == 0 {
		return c.NoContent(http.StatusNoContent)
	}

	if err := _self.FeatureRepo.Update(ctx, id, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

/* ---------- GET /scenarios?feature_id= ---------- */

func (_self *MockController) ListScenarioByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	featureID, err := strconv.ParseInt(c.QueryParam("feature_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid feature_id")
	}

	scenarios, err := _self.ScenarioRepo.ListByFeature(ctx, featureID)
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

	if err := _self.ScenarioRepo.Create(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	return c.JSON(http.StatusCreated, req)
}

/* ---------- PUT /scenarios/:scenario_id ---------- */

func (_self *MockController) UpdateScenarioByFeature(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("scenario_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	var req domain.Scenario
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	update := req.ToMap()
	if err := _self.ScenarioRepo.Update(ctx, id, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

/* ---------- GET /mockapis?scenario_id= ---------- */

func (_self *MockController) ListMockAPIsByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	scenarioID, err := strconv.ParseInt(c.QueryParam("scenario_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	apis, err := _self.MockAPIRepo.ListByScenario(ctx, scenarioID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, apis)
}

/* ---------- POST /mockapis ---------- */

func (_self *MockController) CreateMockAPIByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	var req domain.MockAPI
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.IsActive = true

	if err := _self.MockAPIRepo.Create(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	return c.JSON(http.StatusCreated, req)
}

/* ---------- PUT /mockapis/:api_id ---------- */

func (_self *MockController) UpdateMockAPIByScenario(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("api_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid api_id")
	}
	var req domain.MockAPI
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	update := req.ToMap()
	if err := _self.MockAPIRepo.Update(ctx, id, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}
