package usecase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/pkg/errorcustome"
	"github.com/namnv2496/mocktool/pkg/observability"
	"github.com/namnv2496/mocktool/pkg/security"
	"github.com/namnv2496/mocktool/pkg/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
)

//go:generate mockgen -source=$GOFILE -destination=../../mocks/usecase/$GOFILE.mock.go -package=$GOPACKAGE
type IForwardUC interface {
	ResponseMockData(c echo.Context) error
	ResponsePublicMockData(c echo.Context) error
}
type ForwardUC struct {
	MockAPIRepo         repository.IMockAPIRepository
	ScenarioRepo        repository.IScenarioRepository
	AccountScenarioRepo repository.IAccountScenarioRepository
	cacheRepo           repository.ICache
}

func NewForwardUC(
	MockAPIRepo repository.IMockAPIRepository,
	ScenarioRepo repository.IScenarioRepository,
	AccountScenarioRepo repository.IAccountScenarioRepository,
	cacheRepo repository.ICache,
) IForwardUC {
	return &ForwardUC{
		MockAPIRepo:         MockAPIRepo,
		ScenarioRepo:        ScenarioRepo,
		AccountScenarioRepo: AccountScenarioRepo,
		cacheRepo:           cacheRepo,
	}
}

func (_self *ForwardUC) ResponseMockData(c echo.Context) error {

	accountId := c.Request().Header.Get("X-Account-Id")
	if accountId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Header X-Account-Id is required")
	}

	featureName := c.Request().Header.Get("X-Feature-Name")
	if featureName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Header X-Feature-Name is required")
	}

	return _self.forward(
		c,
		&accountId,
		featureName,
		"/forward",
	)
}

func (_self *ForwardUC) ResponsePublicMockData(c echo.Context) error {
	featureName := c.Request().Header.Get("X-Feature-Name")
	if featureName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Header X-Feature-Name is required")
	}
	return _self.forward(
		c,
		nil,
		featureName,
		"/forward/public",
	)
}

func (_self *ForwardUC) forward(
	c echo.Context,
	accountId *string,
	featureName string,
	trimPrefix string,
) error {
	ctx := c.Request().Context()
	start := time.Now()
	// 1. Read body
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			"failed to read request body",
		)
	}

	// 2. Build path
	path := strings.TrimPrefix(
		c.Request().URL.Path,
		trimPrefix,
	)
	if rawQuery := c.Request().URL.RawQuery; rawQuery != "" {
		values, err := url.ParseQuery(rawQuery)
		if err == nil && len(values) > 0 {
			path += "?" + values.Encode()
		}
	}
	method := c.Request().Method

	// 3. Get active scenario
	accountScenario, err := _self.AccountScenarioRepo.GetActiveScenario(
		ctx,
		featureName,
		accountId,
	)
	if err != nil {
		return err
	}
	scenario, err := _self.ScenarioRepo.GetByObjectID(
		ctx,
		accountScenario.ScenarioID,
	)
	if err != nil {
		return err
	}
	scenarioName := scenario.Name

	// 4. Generate hash
	hash := ""
	if len(bodyBytes) > 0 {
		var tmp map[string]any
		if err := json.Unmarshal(bodyBytes, &tmp); err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"invalid JSON body",
			)
		}
		hash = utils.GenerateHashFromInput(
			bson.Raw(bodyBytes),
		)
	}

	// 5. Build cache key
	acc := ""
	if accountId != nil {
		acc = *accountId
	}

	cacheKey := fmt.Sprintf(
		repository.KeyMockAPITemplate,
		featureName,
		scenarioName,
		acc,
		path,
		method,
		hash,
	)

	// 6. Try cache first
	cached, err := _self.cacheRepo.Get(ctx, cacheKey)
	if err == nil {
		observability.MockAPICacheHits.WithLabelValues("hit").Inc()
		observability.MockAPILookupDuration.Observe(time.Since(start).Seconds())
		c.Response().Header().Set(
			echo.HeaderContentType,
			echo.MIMEApplicationJSON,
		)
		_, err := c.Response().Write(
			[]byte(cached.(string)),
		)
		return err
	}

	// 7. Query DB
	mockAPI, err := _self.MockAPIRepo.FindByFeatureScenarioPathMethodAndHash(
		ctx,
		featureName,
		scenarioName,
		path,
		method,
		hash,
	)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			metadata := map[string]string{
				"x-trace-id": uuid.NewString(),
			}
			return errorcustome.NewError(
				codes.Internal,
				"ERR.001",
				"Mock API not found: %s",
				metadata,
				"not found",
			)
		}

		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}

	// 8. Convert output
	var outputMap map[string]any

	if err := bson.Unmarshal(mockAPI.Output, &outputMap); err != nil {
		if err := json.Unmarshal(
			mockAPI.Output,
			&outputMap,
		); err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"failed to parse output",
			)
		}
	}

	outputBytes, err := json.Marshal(outputMap)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"failed to marshal output",
		)
	}

	// 9. Set headers
	c.Response().Header().Set(
		echo.HeaderContentType,
		echo.MIMEApplicationJSON,
	)

	if len(mockAPI.Headers) > 0 {
		var headers map[string]string
		if err := bson.Unmarshal(
			mockAPI.Headers,
			&headers,
		); err == nil {
			sanitized, _ := security.ValidateAndSanitizeHeaders(headers)

			for k, v := range sanitized {
				c.Response().Header().Set(k, v)
			}
		}
	}

	// 10. Write response
	_, err = c.Response().Write(outputBytes)
	if err != nil {
		return err
	}
	observability.MockAPICacheHits.WithLabelValues("miss").Inc()
	observability.MockAPILookupDuration.Observe(time.Since(start).Seconds())
	// 11. Save cache
	_self.cacheRepo.Set(
		ctx,
		cacheKey,
		string(outputBytes),
	)

	return nil
}
