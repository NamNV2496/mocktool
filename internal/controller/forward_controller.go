package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/namnv2496/mocktool/internal/configs"
	"github.com/namnv2496/mocktool/internal/usecase"
)

type IFowardController interface {
	StartHttpServer() error
}

type FowardController struct {
	config *configs.Config
	trie   usecase.ITrie
}

func NewFowardController(
	config *configs.Config,
	trie usecase.ITrie,
) IFowardController {

	return &FowardController{
		config: config,
		trie:   trie,
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
	targetURL := "http://localhost" + _self.config.AppConfig.HTTPPort + c.Request().RequestURI
	req, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
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
