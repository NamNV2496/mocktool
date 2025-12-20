package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
)

type Test struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

func TestSend() {
	testUri := "/api/v1/test"

	request := Test{
		Id:   123,
		Name: "nam",
	}
	data, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodGet, testUri, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	forwardRequest(c)
}
