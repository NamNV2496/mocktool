package entity

import "encoding/json"

type GRPCMockAPIRequest struct {
	FeatureName  string          `json:"feature_name" validate:"required,no_spaces"`
	ScenarioName string          `json:"scenario_name" validate:"required,no_spaces"`
	ServiceName  string          `json:"service_name" validate:"required"`
	MethodName   string          `json:"method_name" validate:"required"`
	Input        json.RawMessage `json:"input"`
	Output       json.RawMessage `json:"output" validate:"required"`
	StatusCode   int32           `json:"status_code,omitempty"`
	Latency      int64           `json:"latency"`
}

type GRPCMockAPIResponse struct {
	ID           string          `json:"id"`
	FeatureName  string          `json:"feature_name"`
	ScenarioName string          `json:"scenario_name"`
	ServiceName  string          `json:"service_name"`
	MethodName   string          `json:"method_name"`
	Input        any             `json:"input"`
	Output       json.RawMessage `json:"output"`
	StatusCode   int32           `json:"status_code,omitempty"`
	Latency      int64           `json:"latency"`
	IsActive     bool            `json:"is_active"`
}
