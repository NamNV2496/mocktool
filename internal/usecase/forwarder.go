package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/pkg/errorcustome"
	"github.com/namnv2496/mocktool/pkg/observability"
	"github.com/namnv2496/mocktool/pkg/security"
	"github.com/namnv2496/mocktool/pkg/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
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
}

func NewForwardUC(
	MockAPIRepo repository.IMockAPIRepository,
	ScenarioRepo repository.IScenarioRepository,
	AccountScenarioRepo repository.IAccountScenarioRepository,
) IForwardUC {
	return &ForwardUC{
		MockAPIRepo:         MockAPIRepo,
		ScenarioRepo:        ScenarioRepo,
		AccountScenarioRepo: AccountScenarioRepo,
	}
}

func (_self *ForwardUC) ResponseMockData(c echo.Context) error {
	ctx := c.Request().Context()
	ctx, span := observability.GetTracer().Start(ctx, "ForwardUC.ResponseMockData")
	defer span.End()

	// Read request body FIRST before c.Bind() consumes it
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to read request body")
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body: "+err.Error())
	}
	// Log what we received
	slog.Info("Received forwarded request", "bodyLength", len(bodyBytes), "body", string(bodyBytes))

	var request entity.APIRequest
	// Remove /forward prefix from path
	request.Path = strings.TrimPrefix(c.Request().URL.Path, "/forward")

	// Extract accountId from header
	var accountId *string
	accountIdHeader := c.Request().Header.Get("X-Account-Id")
	if accountIdHeader != "" {
		accountId = &accountIdHeader
	} else {
		err := fmt.Errorf("Header X-Account-Id is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "missing account ID header")
		return err
	}
	// Extract featureName from header
	var featureName string
	featureNameHeader := c.Request().Header.Get("X-Feature-Name")
	if featureNameHeader != "" {
		featureName = featureNameHeader
	} else {
		err := fmt.Errorf("Header X-Feature-Name is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "missing feature name header")
		return err
	}
	request.FeatureName = featureName

	// Add span attributes
	span.SetAttributes(
		attribute.String("mocktool.feature", featureName),
		attribute.String("mocktool.account_id", *accountId),
		attribute.String("mocktool.path", request.Path),
	)
	// Include query parameters in the path (excluding feature_name which is used by mocktool)
	// Query parameters are automatically sorted alphabetically by Encode() for consistent matching
	if queryString := c.Request().URL.RawQuery; queryString != "" {
		queryValues, err := url.ParseQuery(queryString)
		if err == nil {
			// Remove feature_name from query parameters as it's internal to mocktool
			// queryValues.Del("feature_name")
			if len(queryValues) > 0 {
				// Encode() sorts keys alphabetically
				request.Path = request.Path + "?" + queryValues.Encode()
			}
		}
	}

	// get active scenario by featureName and accountId
	_, scenarioSpan := observability.GetTracer().Start(ctx, "GetActiveScenario")
	activeAccountScenario, reqerr := _self.AccountScenarioRepo.GetActiveScenario(ctx, request.FeatureName, accountId)
	if reqerr != nil || activeAccountScenario == nil {
		scenarioSpan.RecordError(reqerr)
		scenarioSpan.SetStatus(otelcodes.Error, "failed to get active scenario")
		scenarioSpan.End()
		return reqerr
	}
	scenarioSpan.End()

	// Get the actual scenario details
	_, detailSpan := observability.GetTracer().Start(ctx, "GetScenarioDetails")
	activeScenario, err := _self.ScenarioRepo.GetByObjectID(ctx, activeAccountScenario.ScenarioID)
	if err != nil || activeScenario == nil {
		detailSpan.RecordError(err)
		detailSpan.SetStatus(otelcodes.Error, "failed to get scenario details")
		detailSpan.End()
		return err
	}
	detailSpan.End()

	request.Scenario = activeScenario.Name
	request.Method = c.Request().Method

	span.SetAttributes(
		attribute.String("mocktool.scenario", activeScenario.Name),
		attribute.String("mocktool.method", request.Method),
	)

	// Generate hash from request body
	var hashInput string
	if len(bodyBytes) > 0 {
		// Validate it's proper JSON
		var bodyMap map[string]any
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "invalid JSON body")
			return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
		}
		// Generate hash from sorted input
		hashInput = utils.GenerateHashFromInput(bson.Raw(bodyBytes))
	} else {
		hashInput = ""
	}

	// Query database for matching mock API with metrics
	start := time.Now()
	_, lookupSpan := observability.GetTracer().Start(ctx, "MockAPILookup")
	lookupSpan.SetAttributes(
		attribute.String("db.operation", "FindByFeatureScenarioPathMethodAndHash"),
		attribute.String("db.feature", request.FeatureName),
		attribute.String("db.scenario", activeScenario.Name),
	)

	mockAPI, err := _self.MockAPIRepo.FindByFeatureScenarioPathMethodAndHash(
		ctx,
		request.FeatureName,
		activeScenario.Name,
		request.Path,
		request.Method,
		hashInput,
	)

	// Record metrics
	duration := time.Since(start).Seconds()
	observability.MockAPILookupDuration.Observe(duration)

	if err != nil {
		lookupSpan.RecordError(err)
		lookupSpan.SetStatus(otelcodes.Error, "mock API lookup failed")
		lookupSpan.End()

		observability.MockAPILookupTotal.WithLabelValues(request.FeatureName, activeScenario.Name, "not_found").Inc()

		if err == mongo.ErrNoDocuments {
			metadata := make(map[string]string, 0)
			metadata["x-trace-id"] = uuid.NewString()
			return errorcustome.NewError(codes.Internal, "ERR.001", "Mock API not found: %s", metadata, "not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to query mock API: "+err.Error())
	}

	observability.MockAPILookupTotal.WithLabelValues(request.FeatureName, activeScenario.Name, "found").Inc()
	lookupSpan.SetStatus(otelcodes.Ok, "mock API found")
	lookupSpan.End()

	var outputBytes []byte

	// Handle output - convert bson.Raw to JSON
	var outputMap map[string]interface{}
	if err := bson.Unmarshal(mockAPI.Output, &outputMap); err != nil {
		if err := json.Unmarshal(mockAPI.Output, &outputMap); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to unmarshal output: "+err.Error())
		}
	}
	if outputBytes, err = json.Marshal(outputMap); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal output to JSON")
	}

	// Set response headers
	c.Response().Header().Set("Content-Type", "application/json")

	// Parse and set custom headers from bson.Raw with security validation
	var headersMap map[string]string
	if len(mockAPI.Headers) > 0 {
		if err := bson.Unmarshal(mockAPI.Headers, &headersMap); err == nil {
			// Sanitize headers before setting them in response
			sanitizedHeaders, warnings := security.ValidateAndSanitizeHeaders(headersMap)
			if len(warnings) > 0 {
				slog.Warn("Response headers sanitized", "warnings", warnings)
			}
			for key, value := range sanitizedHeaders {
				c.Response().Header().Set(key, value)
			}
		}
	}
	_, err = io.Copy(c.Response().Writer, strings.NewReader(string(outputBytes)))
	if err != nil {
		return err
	}
	return nil
}

func (_self *ForwardUC) ResponsePublicMockData(c echo.Context) error {
	// Read request body FIRST before c.Bind() consumes it
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body: "+err.Error())
	}
	slog.Info("Received public forwarded request", "bodyLength", len(bodyBytes), "body", string(bodyBytes))

	var request entity.APIRequest
	// Remove /public/forward prefix from path
	request.Path = strings.TrimPrefix(c.Request().URL.Path, "/public/forward")

	// For public API, accountId is nil - will use global scenario
	var accountId *string = nil

	// Extract featureName from header - still required
	var featureName string
	featureNameHeader := c.Request().Header.Get("X-Feature-Name")
	if featureNameHeader != "" {
		featureName = featureNameHeader
	} else {
		return fmt.Errorf("Header X-Feature-Name is required")
	}
	request.FeatureName = featureName

	// Include query parameters in the path
	if queryString := c.Request().URL.RawQuery; queryString != "" {
		queryValues, err := url.ParseQuery(queryString)
		if err == nil {
			if len(queryValues) > 0 {
				request.Path = request.Path + "?" + queryValues.Encode()
			}
		}
	}

	// get active scenario by featureName - accountId is nil for global scenario
	activeAccountScenario, reqerr := _self.AccountScenarioRepo.GetActiveScenario(context.Background(), request.FeatureName, accountId)
	if reqerr != nil || activeAccountScenario == nil {
		return reqerr
	}

	// Get the actual scenario details
	activeScenario, err := _self.ScenarioRepo.GetByObjectID(context.Background(), activeAccountScenario.ScenarioID)
	if err != nil || activeScenario == nil {
		return err
	}

	request.Scenario = activeScenario.Name
	request.Method = c.Request().Method

	// Generate hash from request body
	var hashInput string
	if len(bodyBytes) > 0 {
		var bodyMap map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
		}
		// Generate hash from sorted input
		hashInput = utils.GenerateHashFromInput(bson.Raw(bodyBytes))
	} else {
		hashInput = ""
	}

	// Query database for matching mock API
	mockAPI, err := _self.MockAPIRepo.FindByFeatureScenarioPathMethodAndHash(
		context.Background(),
		request.FeatureName,
		activeScenario.Name,
		request.Path,
		request.Method,
		hashInput,
	)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			metadata := make(map[string]string, 0)
			metadata["x-trace-id"] = uuid.NewString()
			return errorcustome.NewError(codes.Internal, "ERR.001", "Mock API not found: %s", metadata, "not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to query mock API: "+err.Error())
	}

	var outputBytes []byte

	// Handle output - convert bson.Raw to JSON
	var outputMap map[string]interface{}
	if err := bson.Unmarshal(mockAPI.Output, &outputMap); err != nil {
		if err := json.Unmarshal(mockAPI.Output, &outputMap); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to unmarshal output: "+err.Error())
		}
	}
	if outputBytes, err = json.Marshal(outputMap); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal output to JSON")
	}

	// Set response headers
	c.Response().Header().Set("Content-Type", "application/json")

	// Parse and set custom headers from bson.Raw with security validation
	var headersMap map[string]string
	if len(mockAPI.Headers) > 0 {
		if err := bson.Unmarshal(mockAPI.Headers, &headersMap); err == nil {
			// Sanitize headers before setting them in response
			sanitizedHeaders, warnings := security.ValidateAndSanitizeHeaders(headersMap)
			if len(warnings) > 0 {
				slog.Warn("Response headers sanitized", "warnings", warnings)
			}
			for key, value := range sanitizedHeaders {
				c.Response().Header().Set(key, value)
			}
		}
	}
	_, err = io.Copy(c.Response().Writer, strings.NewReader(string(outputBytes)))
	if err != nil {
		return err
	}
	return nil
}
