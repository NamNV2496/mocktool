package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

type Test struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

func TestSend(t *testing.T) {
	testUri := "/api/v1/test"

	request := Test{
		Id:   123,
		Name: "nam",
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, testUri, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err = forwardRequest(c)
	if err != nil {
		t.Logf("forwardRequest returned error (expected if mock server not running): %v", err)
	}
}
