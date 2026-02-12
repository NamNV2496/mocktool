package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/internal/usecase/loadtest"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ILoadTestController interface {
	RegisterRoutes(g *echo.Group)
}

type LoadTestController struct {
	scenarioRepo repository.ILoadTestScenarioRepository
	runner       *loadtest.Runner
}

func NewLoadTestController(
	scenarioRepo repository.ILoadTestScenarioRepository,
) ILoadTestController {
	return &LoadTestController{
		scenarioRepo: scenarioRepo,
		runner:       loadtest.NewRunner(),
	}
}

func (c *LoadTestController) RegisterRoutes(g *echo.Group) {
	// Scenario endpoints
	g.GET("/loadtest/scenarios", c.ListScenarios)
	g.GET("/loadtest/scenarios/search", c.SearchScenarios)
	g.GET("/loadtest/scenarios/:scenario_id", c.GetScenario)
	g.POST("/loadtest/scenarios", c.CreateScenario)
	g.PUT("/loadtest/scenarios/:scenario_id", c.UpdateScenario)
	g.DELETE("/loadtest/scenarios/:scenario_id", c.DeleteScenario)

	// Run endpoint
	g.POST("/loadtest/scenarios/:scenario_id/run", c.RunLoadTest)
}

/* ---------- GET /loadtest/scenarios ---------- */

func (c *LoadTestController) ListScenarios(ctx echo.Context) error {
	params := parsePaginationParams(ctx)

	scenarios, total, err := c.scenarioRepo.ListPaginated(ctx.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if scenarios == nil {
		scenarios = []domain.LoadTestScenario{}
	}

	response := domain.NewPaginatedResponse(scenarios, total, params)
	return ctx.JSON(http.StatusOK, response)
}

/* ---------- GET /loadtest/scenarios/search ---------- */

func (c *LoadTestController) SearchScenarios(ctx echo.Context) error {
	query := ctx.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "search query 'q' is required")
	}

	params := parsePaginationParams(ctx)

	scenarios, total, err := c.scenarioRepo.SearchByName(ctx.Request().Context(), query, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if scenarios == nil {
		scenarios = []domain.LoadTestScenario{}
	}

	response := domain.NewPaginatedResponse(scenarios, total, params)
	return ctx.JSON(http.StatusOK, response)
}

/* ---------- GET /loadtest/scenarios/:scenario_id ---------- */

func (c *LoadTestController) GetScenario(ctx echo.Context) error {
	idStr := ctx.Param("scenario_id")
	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	scenario, err := c.scenarioRepo.GetByID(ctx.Request().Context(), objectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scenario not found")
	}

	return ctx.JSON(http.StatusOK, scenario)
}

/* ---------- POST /loadtest/scenarios ---------- */

func (c *LoadTestController) CreateScenario(ctx echo.Context) error {
	var req domain.LoadTestScenario
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if len(req.Steps) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one step is required")
	}

	req.IsActive = true
	req.CreatedAt = time.Now().UTC()
	req.UpdatedAt = time.Now().UTC()

	if err := c.scenarioRepo.Create(ctx.Request().Context(), &req); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	return ctx.JSON(http.StatusCreated, req)
}

/* ---------- PUT /loadtest/scenarios/:scenario_id ---------- */

func (c *LoadTestController) UpdateScenario(ctx echo.Context) error {
	idStr := ctx.Param("scenario_id")
	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	var req domain.LoadTestScenario
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	update := bson.M{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	// Always update description (even if empty)
	update["description"] = req.Description
	if req.Accounts != "" {
		update["accounts"] = req.Accounts
	}
	if len(req.Steps) > 0 {
		update["steps"] = req.Steps
	}
	update["is_active"] = req.IsActive
	update["updated_at"] = time.Now().UTC()

	if err := c.scenarioRepo.Update(ctx.Request().Context(), objectID, update); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.NoContent(http.StatusOK)
}

/* ---------- DELETE /loadtest/scenarios/:scenario_id ---------- */

func (c *LoadTestController) DeleteScenario(ctx echo.Context) error {
	idStr := ctx.Param("scenario_id")
	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	if err := c.scenarioRepo.Delete(ctx.Request().Context(), objectID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.NoContent(http.StatusOK)
}

/* ---------- POST /loadtest/scenarios/:scenario_id/run ---------- */

func (c *LoadTestController) RunLoadTest(ctx echo.Context) error {
	idStr := ctx.Param("scenario_id")
	objectID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scenario_id")
	}

	// Get scenario from database
	scenario, err := c.scenarioRepo.GetByID(ctx.Request().Context(), objectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scenario not found")
	}

	if !scenario.IsActive {
		return echo.NewHTTPError(http.StatusBadRequest, "scenario is not active")
	}

	if scenario.Accounts == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no accounts configured for this scenario")
	}

	// Convert domain scenario to loadtest scenario
	ltScenario := loadtest.FromDomain(scenario)

	// Parse accounts string (format: "username1-password1,username2-password2")
	accounts := parseAccountsString(scenario.Accounts)
	if len(accounts) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse accounts")
	}

	// Run the load test
	result := c.runner.Run(*ltScenario, accounts)

	return ctx.JSON(http.StatusOK, result)
}

// parseAccountsString parses comma-separated "username-password" pairs
func parseAccountsString(accountsStr string) []loadtest.Account {
	if accountsStr == "" {
		return nil
	}

	pairs := strings.Split(accountsStr, ",")
	accounts := make([]loadtest.Account, 0, len(pairs))

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.Split(pair, "-")
		if len(parts) != 2 {
			continue
		}

		accounts = append(accounts, loadtest.Account{
			Username: strings.TrimSpace(parts[0]),
			Password: strings.TrimSpace(parts[1]),
			Extra:    make(map[string]string),
		})
	}

	return accounts
}
