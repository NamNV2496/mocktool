package entity

import "encoding/json"

type MockAPIRequest struct {
	FeatureName  string          `json:"feature_name" validate:"required,no_spaces"`
	ScenarioName string          `json:"scenario_name" validate:"required,no_spaces"`
	Name         string          `json:"name" validate:"required,no_spaces"`
	Description  string          `json:"description"`
	Path         string          `json:"path" validate:"required,no_spaces"`
	Method       string          `json:"method" validate:"required,no_spaces"`
	Input        json.RawMessage `json:"input"`
	Headers      json.RawMessage `json:"headers"`
	Output       json.RawMessage `json:"output"`
	IsActive     bool            `json:"is_active"`
	Latency      int64           `json:"latency"`
}

type ActiceScenarioRequest struct {
	PrevScenarioId string `json:"prev_scenario_id"`
}
