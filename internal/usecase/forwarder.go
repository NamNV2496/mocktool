package usecase

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
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
	// Read request body FIRST before c.Bind() consumes it
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body: "+err.Error())
	}
	// Log what we received
	slog.Info("Received forwarded request", "bodyLength", len(bodyBytes), "body", string(bodyBytes))

	var request entity.APIRequest
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
	request.Method = c.Request().Method

	// Store raw JSON for comparison
	if len(bodyBytes) > 0 {
		// Validate it's proper JSON
		var bodyMap map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
		}
		// Store raw JSON bytes (comparison function handles both JSON and BSON)
		request.HashInput = bson.Raw(bodyBytes)
	} else {
		request.HashInput = bson.Raw{}
	}

	response := _self.trie.Search(request)
	if response == nil {
		_, err = io.Copy(c.Response().Writer, strings.NewReader("not found"))
		return err
	}

	var outputBytes []byte

	// Handle different output types
	switch v := response.Output.(type) {
	case string:
		outputBytes = []byte(v)
	case bson.Raw:
		var outputMap map[string]interface{}
		if err := bson.Unmarshal(v, &outputMap); err != nil {
			if err := json.Unmarshal(v, &outputMap); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to unmarshal output: "+err.Error())
			}
		}
		if outputBytes, err = json.Marshal(outputMap); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal output to JSON")
		}
	default:
		// For any other type, try to marshal it as JSON
		if outputBytes, err = json.Marshal(v); err != nil {
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
