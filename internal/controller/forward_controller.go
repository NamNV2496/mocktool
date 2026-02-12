package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/entity"
	"github.com/namnv2496/mocktool/internal/usecase"
	"github.com/namnv2496/mocktool/pkg/errorcustome"
)

type IForwardController interface {
	StartMockServer() error
}

type ForwardController struct {
	config    *configs.Config
	forwardUc usecase.IForwardUC
	TestWay   int
	// trie      usecase.ITrie
}

func NewFowardController(
	config *configs.Config,
	forwardUc usecase.IForwardUC,
	flags entity.ServiceFlags,
	// trie usecase.ITrie,
) IForwardController {

	return &ForwardController{
		config:    config,
		forwardUc: forwardUc,
		TestWay:   flags.TestWay,
		// trie:      trie,
	}
}

func (_self *ForwardController) StartMockServer() error {
	c := echo.New()
	// Middleware
	c.Use(middleware.CORS())          // enable CORS for web interface
	c.Use(middleware.RequestLogger()) // use the default RequestLogger middleware with slog logger
	c.Use(middleware.Recover())       // recover panics as errors for proper error handling

	c.GET("/forward/*", _self.responseMockData)
	c.POST("/forward/*", _self.responseMockData)
	c.PUT("/forward/*", _self.responseMockData)
	c.DELETE("/forward/*", _self.responseMockData)

	// Public API endpoints - no X-Account-Id required, uses global scenario
	c.GET("/forward/public/*", _self.responsePublicMockData)
	c.POST("/forward/public/*", _self.responsePublicMockData)
	c.PUT("/forward/public/*", _self.responsePublicMockData)
	c.DELETE("/forward/public/*", _self.responsePublicMockData)

	if err := c.Start(_self.config.AppConfig.FowardHTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
		return err
	}
	return nil
}

// handler
func (_self *ForwardController) responseMockData(c echo.Context) error {
	switch _self.TestWay {
	case 1:
		// =================================================WAY 1===================================================
		// wrapper http response
		err := _self.forwardUc.ResponseMockData(c)
		if err != nil {
			errResp := errorcustome.WrapErrorResponse(context.Background(), err)
			c.Response().Header().Set("Content-Type", "application/json")
			c.Response().WriteHeader(errResp.HttpStatus)
			errRespBytes, _ := json.Marshal(errResp)
			_, err = io.Copy(c.Response().Writer, strings.NewReader(string(errRespBytes)))
			return err
		}
		c.Response().WriteHeader(http.StatusOK)
		return nil
	case 2:
		// =================================================WAY 2===================================================
		// forward response
		return _self.forwardUc.ResponseMockData(c)
	default:
		return nil
	}
}

// handler for public API - no X-Account-Id required
func (_self *ForwardController) responsePublicMockData(c echo.Context) error {
	switch _self.TestWay {
	case 1:
		err := _self.forwardUc.ResponsePublicMockData(c)
		if err != nil {
			errResp := errorcustome.WrapErrorResponse(context.Background(), err)
			c.Response().Header().Set("Content-Type", "application/json")
			c.Response().WriteHeader(errResp.HttpStatus)
			errRespBytes, _ := json.Marshal(errResp)
			_, err = io.Copy(c.Response().Writer, strings.NewReader(string(errRespBytes)))
			return err
		}
		c.Response().WriteHeader(http.StatusOK)
		return nil
	case 2:
		return _self.forwardUc.ResponsePublicMockData(c)
	default:
		return nil
	}
}
