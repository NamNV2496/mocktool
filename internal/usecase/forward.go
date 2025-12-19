package usecase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
)

type IForwardUC interface {
	ResponseMockData(c echo.Context) error
}
type ForwardUC struct {
	trie         ITrie
	ScenarioRepo repository.IScenarioRepository
}

func NewForwardUC(
	trie ITrie,
	ScenarioRepo repository.IScenarioRepository,
) IForwardUC {
	return &ForwardUC{
		trie:         trie,
		ScenarioRepo: ScenarioRepo,
	}
}
func (_self *ForwardUC) ResponseMockData(c echo.Context) error {
	var request entity.APIRequest
	if err := c.Bind(&request); err != nil {
		return err
	}
	// Remove /forward prefix from path
	request.Path = strings.TrimPrefix(c.Request().URL.Path, "/forward")
	// Bind query parameters
	request.FeatureName = c.QueryParam("feature_name")

	// get active scenario by featureName
	activeScenario, reqerr := _self.ScenarioRepo.GetActiveScenarioByFeatureName(context.Background(), request.FeatureName)
	if reqerr != nil || activeScenario == nil {
		return reqerr
	}

	request.Scenario = activeScenario.Name
	response := _self.trie.Search(request)
	if response == nil {
		io.Copy(c.Response().Writer, strings.NewReader("not found"))
		return nil
	}

	var outputBytes []byte
	var err error

	// Handle different output types
	switch v := response.Output.(type) {
	case string:
		outputBytes = []byte(v)
	case bson.Raw:
		// Convert bson.Raw to JSON format
		var result bson.M
		if err = bson.Unmarshal(v, &result); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to unmarshal bson data")
		}
		outputBytes, err = json.Marshal(result)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert to json")
		}
	default:
		// For any other type, try to marshal it as JSON
		outputBytes, err = json.Marshal(v)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "invalid response output type")
		}
	}

	c.Response().Header().Set("Content-Type", "application/json")
	_, err = io.Copy(c.Response().Writer, strings.NewReader(string(outputBytes)))
	if err != nil {
		return err
	}
	return nil
}
