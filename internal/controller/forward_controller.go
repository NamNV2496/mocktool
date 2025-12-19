package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/namnv2496/mocktool/internal/configs"
)

type IFowardController interface {
	StartHttpServer() error
}

type FowardController struct {
	config *configs.Config
}

func NewFowardController(
	config *configs.Config,
) IFowardController {

	return &FowardController{
		config: config,
	}
}

func (_self *FowardController) StartHttpServer() error {
	e := echo.New()
	// Middleware
	e.Use(middleware.RequestLogger()) // use the default RequestLogger middleware with slog logger
	e.Use(middleware.Recover())       // recover panics as errors for proper error handling
	// Routes
	forward := e.Group("/forward")
	forward.POST("/*", _self.forwardRequest)
	forward.GET("/*", _self.forwardRequest)
	forward.PUT("/*", _self.forwardRequest)
	forward.DELETE("/*", _self.forwardRequest)

	if err := e.Start(_self.config.AppConfig.FowardHTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
		return err
	}
	return nil
}

func (_self *FowardController) forwardRequest(c echo.Context) error {
	// Read the request body into a buffer so it can be used multiple times
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read request body: " + err.Error()})
	}
	c.Request().Body.Close()

	// Restore the body for the current context
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Convert request body to map[string]string, sort by key, and hash
	var bodyMap map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body: " + err.Error()})
		}
	} else {
		bodyMap = make(map[string]interface{})
	}

	// Extract query parameters and add them to the body
	queryParams := c.QueryParams()
	for key, values := range queryParams {
		if len(values) > 0 {
			bodyMap[key] = values[0] // Use the first value if multiple values exist
		}
	}

	// Convert to map[string]string and sort by key
	sortedMap := make(map[string]string)
	keys := make([]string, 0, len(bodyMap))

	for k, v := range bodyMap {
		keys = append(keys, k)
		sortedMap[k] = fmt.Sprintf("%v", v)
	}
	sort.Strings(keys)

	// Create a sorted structure for hashing
	sortedData := make([]struct {
		Key   string
		Value string
	}, 0, len(keys))

	for _, k := range keys {
		sortedData = append(sortedData, struct {
			Key   string
			Value string
		}{Key: k, Value: sortedMap[k]})
	}

	// Hash the sorted data
	hash, err := hashstructure.Hash(sortedData, hashstructure.FormatV2, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash data: " + err.Error()})
	}

	// Add hash_input to the body
	bodyMap["hash_input"] = fmt.Sprintf("%d", hash)

	// Marshal the updated body
	newBodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal body: " + err.Error()})
	}

	// Create the forward request with the updated body
	targetURL := "http://localhost" + _self.config.AppConfig.HTTPPort + c.Request().RequestURI
	req, err := http.NewRequest(c.Request().Method, targetURL, bytes.NewBuffer(newBodyBytes))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Copy headers
	for k, v := range c.Request().Header {
		req.Header[k] = v
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		c.Response().Header()[k] = v
	}

	return c.Stream(resp.StatusCode, resp.Header.Get("Content-Type"), resp.Body)
}
