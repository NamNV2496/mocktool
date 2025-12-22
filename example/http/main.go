package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	targetURL := "http://localhost:8081/forward" + c.Request().RequestURI
	req, err := http.NewRequest(c.Request().Method, targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Copy headers
	for k, v := range c.Request().Header {
		req.Header[k] = v
	}
	// parse accountId from token
	accountId := "1"
	// add accountId to header
	req.Header["X-Account-Id"] = []string{accountId}
	req.Header["X-Feature-Name"] = []string{"test_feature"}

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
	// respBody, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return status.Errorf(codes.Internal, "read response failed: %v", err)
	// }

	// var errResp errorcustome.ErrorResponse
	// json.Unmarshal(respBody, &errResp)

	// // Reconstruct gRPC status error with details
	// st := status.New(errResp.GrpcCode, errResp.ErrorMessage)
	// stWithDetails, _ := st.WithDetails(&pb.ErrorDetail{
	// 	ErrorCode: errResp.ErrorCode,
	// 	Metadata:  errResp.Details,
	// })

	// return c.Stream(resp.StatusCode, resp.Header.Get("Content-Type"), stWithDetails.Err())

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return status.Errorf(codes.Internal, "read response failed: %v", err)
	}

	return c.Stream(resp.StatusCode, resp.Header.Get("Content-Type"), bytes.NewBuffer(respBody))
}
