package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/repository"
	"github.com/namnv2496/mocktool/pkg/utils"
)

func listAPIs(d Deps) Tool {
	type args struct {
		Feature  string `json:"feature"`
		Scenario string `json:"scenario"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	return Tool{
		Name:        "list_apis",
		Description: "List mock APIs (path + method + request hash + response) of a scenario.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["scenario"],
            "properties": {
                "feature":   {"type": "string", "description": "optional, not enforced by the underlying repo but included for context"},
                "scenario":  {"type": "string"},
                "page":      {"type": "integer", "minimum": 1, "default": 1},
                "page_size": {"type": "integer", "minimum": 1, "maximum": 100, "default": 50}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Scenario == "" {
				return nil, fmt.Errorf("scenario is required")
			}
			params := normalizePagination(a.Page, a.PageSize)
			apis, total, err := d.MockAPI.ListByScenarioNamePaginated(ctx, a.Scenario, params)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"apis":      mockAPIsToJSON(apis),
				"total":     total,
				"page":      params.Page,
				"page_size": params.PageSize,
			}, nil
		},
	}
}

func searchMocks(d Deps) Tool {
	type args struct {
		Scenario string `json:"scenario"`
		Query    string `json:"query"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	return Tool{
		Name:        "search_mocks",
		Description: "Search mock APIs within a scenario by case-insensitive substring on name or path.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["scenario", "query"],
            "properties": {
                "scenario":  {"type": "string"},
                "query":     {"type": "string"},
                "page":      {"type": "integer", "minimum": 1, "default": 1},
                "page_size": {"type": "integer", "minimum": 1, "maximum": 100, "default": 50}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Scenario == "" || a.Query == "" {
				return nil, fmt.Errorf("scenario and query are required")
			}
			params := normalizePagination(a.Page, a.PageSize)
			apis, total, err := d.MockAPI.SearchByScenarioAndNameOrPath(ctx, a.Scenario, a.Query, params)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"apis":      mockAPIsToJSON(apis),
				"total":     total,
				"page":      params.Page,
				"page_size": params.PageSize,
			}, nil
		},
	}
}

func createMockAPI(d Deps) Tool {
	type args struct {
		Feature      string          `json:"feature"`
		Scenario     string          `json:"scenario"`
		Name         string          `json:"name"`
		Description  string          `json:"description"`
		BaseURL      string          `json:"base_url"`
		Path         string          `json:"path"`
		Method       string          `json:"method"`
		RequestBody  json.RawMessage `json:"request_body"`
		Response     json.RawMessage `json:"response"`
		Headers      map[string]string `json:"headers"`
		LatencyMs    int64           `json:"latency_ms"`
	}
	return Tool{
		Name:        "create_mock_api",
		Description: "Create a new mock API under a feature+scenario. The request_body hash uniquely identifies an entry along with path+method. Headers are arbitrary string->string map.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario", "name", "path", "method", "response"],
            "properties": {
                "feature":      {"type": "string"},
                "scenario":     {"type": "string"},
                "name":         {"type": "string"},
                "description":  {"type": "string"},
                "base_url":     {"type": "string"},
                "path":         {"type": "string", "description": "supports :param placeholders, e.g. /api/v1/user/:id"},
                "method":       {"type": "string", "enum": ["GET","POST","PUT","PATCH","DELETE"]},
                "request_body": {"description": "JSON value used as match filter; the hash of this determines which mock entry matches"},
                "response":     {"description": "JSON value returned to the caller"},
                "headers":      {"type": "object", "additionalProperties": {"type": "string"}},
                "latency_ms":   {"type": "integer", "minimum": 0}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Scenario == "" || a.Name == "" || a.Path == "" || a.Method == "" {
				return nil, fmt.Errorf("feature, scenario, name, path, method are required")
			}
			if len(a.Response) == 0 || string(a.Response) == "null" {
				return nil, fmt.Errorf("response is required")
			}

			// Check duplicate by name within the same scenario.
			if existing, _ := d.MockAPI.FindByNameAndFeatureAndScenario(ctx, a.Name, a.Feature, a.Scenario); existing != nil && existing.Name != "" {
				return nil, fmt.Errorf("mock api %q already exists in %s/%s", a.Name, a.Feature, a.Scenario)
			}

			req := domain.MockAPI{
				FeatureName:  a.Feature,
				ScenarioName: a.Scenario,
				Name:         a.Name,
				Description:  a.Description,
				BaseURL:      a.BaseURL,
				Path:         a.Path,
				Method:       a.Method,
				Latency:      a.LatencyMs,
				IsActive:     true,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}

			// Convert request body -> bson.Raw and compute hash.
			if len(a.RequestBody) > 0 && string(a.RequestBody) != "null" {
				var data any
				if err := json.Unmarshal(a.RequestBody, &data); err != nil {
					return nil, fmt.Errorf("invalid request_body: %w", err)
				}
				inputBSON, err := bson.Marshal(data)
				if err != nil {
					return nil, fmt.Errorf("marshal request_body to bson: %w", err)
				}
				req.Input = inputBSON
				req.HashInput = utils.GenerateHashFromInput(inputBSON)
			}

			// Duplicate-by-shape guard: path + method + hash within feature/scenario.
			if existing, _ := d.MockAPI.FindByFeatureScenarioPathMethodAndHash(ctx, req.FeatureName, req.ScenarioName, req.Path, req.Method, req.HashInput); existing != nil && existing.Name != "" {
				return nil, fmt.Errorf("a mock api with same path+method+request_body already exists (%s)", existing.Name)
			}

			// Convert response -> bson.Raw.
			var respData any
			if err := json.Unmarshal(a.Response, &respData); err != nil {
				return nil, fmt.Errorf("invalid response: %w", err)
			}
			outputBSON, err := bson.Marshal(respData)
			if err != nil {
				return nil, fmt.Errorf("marshal response to bson: %w", err)
			}
			req.Output = outputBSON

			// Headers — pass through as-is; existing controller does additional
			// sanitization for HTTP-clients; mocking should not silently mutate
			// what the user asked for here.
			if len(a.Headers) > 0 {
				hBSON, err := bson.Marshal(a.Headers)
				if err != nil {
					return nil, fmt.Errorf("marshal headers to bson: %w", err)
				}
				req.Headers = bson.Raw(hBSON)
			}

			if err := d.MockAPI.Create(ctx, &req); err != nil {
				return nil, fmt.Errorf("create mock api: %w", err)
			}

			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyScnarioTemplate, a.Feature, a.Scenario))

			return map[string]any{
				"id":            req.ID.Hex(),
				"feature":       req.FeatureName,
				"scenario":      req.ScenarioName,
				"name":          req.Name,
				"path":          req.Path,
				"method":        req.Method,
				"hash_input":    req.HashInput,
				"created_at":    req.CreatedAt.Format(time.RFC3339),
			}, nil
		},
	}
}

func updateMockAPI(d Deps) Tool {
	type args struct {
		APIID       string            `json:"api_id"`
		Name        string            `json:"name"`
		Description string            `json:"description"`
		BaseURL     string            `json:"base_url"`
		Path        string            `json:"path"`
		Method      string            `json:"method"`
		Response    json.RawMessage   `json:"response"`
		Headers     map[string]string `json:"headers"`
		LatencyMs   *int64            `json:"latency_ms"`
		IsActive    *bool             `json:"is_active"`
	}
	return Tool{
		Name:        "update_mock_api",
		Description: "Update an existing mock API by id. Only provided fields are changed.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["api_id"],
            "properties": {
                "api_id":      {"type": "string", "description": "ObjectID hex"},
                "name":        {"type": "string"},
                "description": {"type": "string"},
                "base_url":    {"type": "string"},
                "path":        {"type": "string"},
                "method":      {"type": "string"},
                "response":    {},
                "headers":     {"type": "object", "additionalProperties": {"type": "string"}},
                "latency_ms":  {"type": "integer", "minimum": 0},
                "is_active":   {"type": "boolean"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.APIID == "" {
				return nil, fmt.Errorf("api_id is required")
			}
			id, err := primitive.ObjectIDFromHex(a.APIID)
			if err != nil {
				return nil, fmt.Errorf("invalid api_id: %w", err)
			}
			update := bson.M{}
			if a.Name != "" {
				update["name"] = a.Name
			}
			if a.Description != "" {
				update["description"] = a.Description
			}
			if a.BaseURL != "" {
				update["base_url"] = a.BaseURL
			}
			if a.Path != "" {
				update["path"] = a.Path
			}
			if a.Method != "" {
				update["method"] = a.Method
			}
			if a.LatencyMs != nil {
				update["latency"] = *a.LatencyMs
			}
			if a.IsActive != nil {
				update["is_active"] = *a.IsActive
			}
			if len(a.Response) > 0 && string(a.Response) != "null" {
				var respData any
				if err := json.Unmarshal(a.Response, &respData); err != nil {
					return nil, fmt.Errorf("invalid response: %w", err)
				}
				outputBSON, err := bson.Marshal(respData)
				if err != nil {
					return nil, fmt.Errorf("marshal response to bson: %w", err)
				}
				update["output"] = outputBSON
			}
			if len(a.Headers) > 0 {
				hBSON, err := bson.Marshal(a.Headers)
				if err != nil {
					return nil, fmt.Errorf("marshal headers to bson: %w", err)
				}
				update["headers"] = bson.Raw(hBSON)
			}
			if len(update) == 0 {
				return nil, fmt.Errorf("no fields to update")
			}
			update["updated_at"] = time.Now().UTC()

			if err := d.MockAPI.UpdateByObjectID(ctx, id, update); err != nil {
				return nil, fmt.Errorf("update mock api: %w", err)
			}
			updated, _ := d.MockAPI.FindByObjectID(ctx, id)
			if updated != nil {
				_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyScnarioTemplate, updated.FeatureName, updated.ScenarioName))
			}
			return map[string]any{
				"id":         a.APIID,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
				"fields":     keysOf(update),
			}, nil
		},
	}
}

func deleteMockAPI(d Deps) Tool {
	type args struct {
		APIID string `json:"api_id"`
	}
	return Tool{
		Name:        "delete_mock_api",
		Description: "Delete a mock API by id. Cache is invalidated. Irreversible.",
		Destructive: true,
		InputSchema: schema(`{
            "type": "object",
            "required": ["api_id"],
            "properties": {"api_id": {"type": "string", "description": "ObjectID hex"}}
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.APIID == "" {
				return nil, fmt.Errorf("api_id is required")
			}
			id, err := primitive.ObjectIDFromHex(a.APIID)
			if err != nil {
				return nil, fmt.Errorf("invalid api_id: %w", err)
			}
			mockAPI, err := d.MockAPI.FindByObjectID(ctx, id)
			if err != nil || mockAPI == nil || mockAPI.Name == "" {
				return nil, fmt.Errorf("mock api %s not found", a.APIID)
			}
			if err := d.MockAPI.DeletByObjectID(ctx, id); err != nil {
				return nil, fmt.Errorf("delete mock api: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyScnarioTemplate, mockAPI.FeatureName, mockAPI.ScenarioName))
			return map[string]any{
				"id":         a.APIID,
				"feature":    mockAPI.FeatureName,
				"scenario":   mockAPI.ScenarioName,
				"name":       mockAPI.Name,
				"deleted_at": time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}
}

func mockAPIsToJSON(apis []domain.MockAPI) []map[string]any {
	out := make([]map[string]any, 0, len(apis))
	for _, api := range apis {
		entry := map[string]any{
			"id":            api.ID.Hex(),
			"feature_name":  api.FeatureName,
			"scenario_name": api.ScenarioName,
			"name":          api.Name,
			"description":   api.Description,
			"is_active":     api.IsActive,
			"base_url":      api.BaseURL,
			"path":          api.Path,
			"method":        api.Method,
			"hash_input":    api.HashInput,
			"latency_ms":    api.Latency,
			"input":         bsonRawToJSON(api.Input),
			"output":        bsonRawToJSON(api.Output),
			"headers":       bsonRawToJSON(api.Headers),
		}
		out = append(out, entry)
	}
	return out
}

func bsonRawToJSON(raw bson.Raw) any {
	if len(raw) == 0 {
		return nil
	}
	var m bson.M
	if err := bson.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

func keysOf(m bson.M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
