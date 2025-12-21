package entity

import "go.mongodb.org/mongo-driver/bson"

type APIRequest struct {
	FeatureName string            `json:"feature_name"`
	Scenario    string            `json:"scenario"`
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	HashInput   bson.Raw          `json:"hash_input"` // hashcode of input
	Headers     map[string]string `json:"headers"`    // original map
	Output      any               `json:"output"`     // json response
}

type APIResponse struct {
	// FeatureName string            `json:"feature_name"`
	// Scenario    string            `json:"scenario"`
	// Path        string            `json:"path"`
	Output  any               `json:"output"`  // json response
	Headers map[string]string `json:"headers"` // original map
}
