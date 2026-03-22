package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"log"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/singleflight"

	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/pkg/errorcustome"
	"github.com/namnv2496/mocktool/pkg/observability"
	"github.com/namnv2496/mocktool/pkg/security"
	"github.com/namnv2496/mocktool/pkg/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
)

const (
	sequenceCounterTTL = 24 * time.Hour
	notFoundSentinel   = "__not_found__"
	notFoundCacheTTL   = 30 * time.Second
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
	sfGroup             singleflight.Group
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
		"/forward",
	)
}

type sfResolved struct {
	outputBytes []byte
	headersRaw  bson.Raw
	latency     int64
	mockAPI     *domain.MockAPI
	isSequence  bool
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
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
	}

	// 2. Build path
	path := strings.TrimPrefix(c.Request().URL.Path, trimPrefix)
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
	scenario, err := _self.ScenarioRepo.GetByObjectID(ctx, accountScenario.ScenarioID)
	if err != nil {
		return err
	}
	scenarioName := scenario.Name

	// 4. Generate hash
	hash := ""
	if len(bodyBytes) > 0 {
		var tmp map[string]any
		if err := json.Unmarshal(bodyBytes, &tmp); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
		}
		hash = utils.GenerateHashFromInput(bson.Raw(bodyBytes))
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

	if cached, err := _self.cacheRepo.Get(ctx, cacheKey); err == nil {
		observability.MockAPICacheHits.WithLabelValues("hit").Inc()
		observability.MockAPILookupDuration.Observe(time.Since(start).Seconds())
		if cached.(string) == notFoundSentinel {
			return echo.NewHTTPError(http.StatusNotFound, "mock API not found")
		}
		var entry entity.CachedEntry
		if err := json.Unmarshal([]byte(cached.(string)), &entry); err == nil {
			if entry.Latency > 0 {
				time.Sleep(time.Duration(entry.Latency) * time.Second)
			}
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			_, err := c.Response().Write([]byte(entry.Output))
			return err
		}
	}

	// Cache miss: use singleflight to prevent thundering herd.
	// Use a detached context for the fetch so a cancelled caller does not abort
	// the shared in-flight request and invalidate results for other waiters.
	fetchCtx := context.WithoutCancel(ctx)
	v, err, _ := _self.sfGroup.Do(cacheKey, func() (any, error) {
		mockAPI, err := _self.MockAPIRepo.FindByFeatureScenarioPathMethodAndHash(
			fetchCtx,
			featureName,
			scenarioName,
			path,
			method,
			hash,
		)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// Exact path miss — try pattern matching (e.g. /api/users/:id).
				mockAPI, err = findByPathPattern(fetchCtx, _self.MockAPIRepo, featureName, scenarioName, path, method, hash)
			}
			if err != nil {
				// Negative cache: store sentinel so subsequent waves skip the DB.
				_self.cacheRepo.SetWithTTL(fetchCtx, cacheKey, notFoundSentinel, notFoundCacheTTL)
				if err == mongo.ErrNoDocuments {
					metadata := map[string]string{
						"x-trace-id": uuid.NewString(),
					}
					return nil, errorcustome.NewError(
						codes.Internal,
						"ERR.001",
						"Mock API not found: %s",
						metadata,
						"not found",
					)
				}
				return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
		}

		if len(mockAPI.Responses) > 0 {
			return &sfResolved{mockAPI: mockAPI, isSequence: true}, nil
		}
		outputBytes, err := rawToJSON(mockAPI.Output)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to parse output")
		}
		entry := entity.CachedEntry{Output: string(outputBytes), Latency: int64(1000)} // delay 1000ms = 1s
		if entryBytes, err := json.Marshal(entry); err == nil {
			_self.cacheRepo.Set(fetchCtx, cacheKey, string(entryBytes))
		}
		return &sfResolved{
			outputBytes: outputBytes,
			headersRaw:  mockAPI.Headers,
			latency:     mockAPI.Latency,
			isSequence:  false,
		}, nil
	})
	if err != nil {
		return err
	}

	r := v.(*sfResolved)

	var outputBytes []byte
	var headersRaw bson.Raw
	var latency int64

	if r.isSequence {
		seqKey := fmt.Sprintf(
			repository.KeySequenceTemplate,
			featureName,
			scenarioName,
			acc,
			path,
			method,
			hash,
		)
		count, err := _self.cacheRepo.IncrWithTTL(ctx, seqKey, sequenceCounterTTL)
		if err != nil {
			log.Println("failed to increment sequence counter:", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to increment sequence counter")
		}

		matched := findMatchingResponse(r.mockAPI.Responses, int(count))
		if matched != nil {
			outputBytes, err = rawToJSON(matched.Output)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse sequence output")
			}
			headersRaw = matched.Headers
			latency = matched.Latency
		} else {
			outputBytes, err = rawToJSON(r.mockAPI.Output)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse output")
			}
			headersRaw = r.mockAPI.Headers
			latency = r.mockAPI.Latency
		}
	} else {
		outputBytes = r.outputBytes
		headersRaw = r.headersRaw
		latency = r.latency
	}
	if latency > 0 {
		time.Sleep(time.Duration(latency) * time.Second)
	}

	// 10. Set headers
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	if len(headersRaw) > 0 {
		var headers map[string]string
		if err := bson.Unmarshal(headersRaw, &headers); err == nil {
			sanitized, _ := security.ValidateAndSanitizeHeaders(headers)
			for k, v := range sanitized {
				c.Response().Header().Set(k, v)
			}
		}
	}

	// 11. Write response
	_, err = c.Response().Write(outputBytes)
	if err != nil {
		return err
	}
	observability.MockAPICacheHits.WithLabelValues("miss").Inc()
	observability.MockAPILookupDuration.Observe(time.Since(start).Seconds())

	return nil
}
func rawToJSON(raw bson.Raw) ([]byte, error) {
	var m map[string]any
	if err := bson.Unmarshal(raw, &m); err != nil {
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
	}
	return json.Marshal(m)
}

// findByPathPattern fetches all active APIs for the given feature/scenario/method
// and returns the first one whose stored path pattern matches actualPath and whose
// hash_input matches hashInput. Used as a fallback after an exact-path miss.
func findByPathPattern(
	ctx context.Context,
	repo repository.IMockAPIRepository,
	featureName, scenarioName, actualPath, method, hashInput string,
) (*domain.MockAPI, error) {
	candidates, err := repo.FindCandidatesByFeatureScenarioAndMethod(ctx, featureName, scenarioName, method)
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		c := &candidates[i]
		if c.HashInput == hashInput && utils.MatchPath(c.Path, actualPath) {
			return c, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}

// applyChaos randomly injects errors and/or latency jitter based on the mock's
// ChaosConfig. It returns an HTTP error when an error is injected, nil otherwise.
// Both mechanisms are independent — jitter is applied even when no error fires.
func applyChaos(errorRate float64, errorStatus int, jitterMs int64) error {
	if jitterMs > 0 {
		time.Sleep(time.Duration(rand.Int63n(jitterMs)) * time.Millisecond)
	}
	if errorRate > 0 && rand.Float64() < errorRate {
		status := errorStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return echo.NewHTTPError(status, "chaos: injected error")
	}
	return nil
}

func findMatchingResponse(responses []domain.SequenceResponse, count int) *domain.SequenceResponse {
	for i := range responses {
		r := &responses[i]
		if count >= r.From && (r.To == 0 || count <= r.To) {
			return r
		}
	}
	return nil
}
