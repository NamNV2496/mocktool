package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	StartHttpServer()
}

func StartHttpServer() error {
	e := echo.New()
	// Middleware
	e.Use(middleware.RequestLogger()) // use the default RequestLogger middleware with slog logger
	e.Use(middleware.Recover())       // recover panics as errors for proper error handling
	// Routes
	e.POST("/*", PostHandler)
	e.GET("/*", GetHandler)
	e.PUT("/*", PutHandler)
	e.DELETE("/*", DeleteHandler)

	if err := e.Start(":9090"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
		return err
	}
	return nil
}

func PostHandler(c echo.Context) error {
	return forwardRequest(c)
}

func GetHandler(c echo.Context) error {
	return forwardRequest(c)
}

func PutHandler(c echo.Context) error {
	return forwardRequest(c)
}

func DeleteHandler(c echo.Context) error {
	return forwardRequest(c)
}

func forwardRequest(c echo.Context) error {
	// Read the request body into a buffer so it can be used multiple times
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read request body: " + err.Error()})
	}
	c.Request().Body.Close()

	// Log the request body being forwarded
	slog.Info("Forwarding requestBody", "body", string(bodyBytes))

	// Create the forward request with the updated body + append feature_name
	targetURL := "http://localhost:8081/forward" + c.Request().RequestURI + "?feature_name=iav3_job"
	req, err := http.NewRequest(c.Request().Method, targetURL, bytes.NewBuffer(bodyBytes))
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
