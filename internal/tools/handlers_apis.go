package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

type seqResponseArg struct {
	From       int               `json:"from"`
	To         int               `json:"to"`
	StatusCode int               `json:"status_code"`
	Output     json.RawMessage   `json:"output"`
	Headers    map[string]string `json:"headers"`
	LatencyMs  int64             `json:"latency_ms"`
}

func seqArgsToDomain(in []seqResponseArg) ([]domain.SequenceResponse, error) {
	out := make([]domain.SequenceResponse, 0, len(in))
	for _, s := range in {
		sr := domain.SequenceResponse{
			From:       s.From,
			To:         s.To,
			StatusCode: s.StatusCode,
			Latency:    s.LatencyMs,
		}
		if len(s.Output) > 0 && string(s.Output) != "null" {
			var d any
			if err := json.Unmarshal(s.Output, &d); err != nil {
				return nil, fmt.Errorf("invalid sequence response output: %w", err)
			}
			b, err := bson.Marshal(d)
			if err != nil {
				return nil, fmt.Errorf("marshal sequence response output: %w", err)
			}
			sr.Output = b
		}
		if len(s.Headers) > 0 {
			b, err := bson.Marshal(s.Headers)
			if err != nil {
				return nil, fmt.Errorf("marshal sequence response headers: %w", err)
			}
			sr.Headers = b
		}
		out = append(out, sr)
	}
	return out, nil
}

func createMockAPI(d Deps) Tool {
	type args struct {
		Feature     string            `json:"feature"`
		Scenario    string            `json:"scenario"`
		Name        string            `json:"name"`
		Description string            `json:"description"`
		BaseURL     string            `json:"base_url"`
		Path        string            `json:"path"`
		Method      string            `json:"method"`
		RequestBody json.RawMessage   `json:"request_body"`
		Response    json.RawMessage   `json:"response"`
		StatusCode  int               `json:"status_code"`
		Headers     map[string]string `json:"headers"`
		LatencyMs   int64             `json:"latency_ms"`
		Responses   []seqResponseArg  `json:"responses"`
	}
	return Tool{
		Name:        "create_mock_api",
		Description: "Create a new mock API under a feature+scenario. The request_body hash uniquely identifies an entry along with path+method. Use status_code to return a non-200 default response. Optionally provide a 'responses' array for sequence responses (different reply per call count, each with its own status_code).",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario", "name", "path", "method"],
            "properties": {
                "feature":      {"type": "string"},
                "scenario":     {"type": "string"},
                "name":         {"type": "string"},
                "description":  {"type": "string"},
                "base_url":     {"type": "string"},
                "path":         {"type": "string", "description": "supports :param placeholders, e.g. /api/v1/user/:id"},
                "method":       {"type": "string", "enum": ["GET","POST","PUT","PATCH","DELETE"]},
                "request_body": {"description": "JSON value used as match filter; the hash of this determines which mock entry matches"},
                "response":     {"description": "default JSON response returned when no sequence entry matches"},
                "status_code":  {"type": "integer", "description": "HTTP status code for the default response (default 200)"},
                "headers":      {"type": "object", "additionalProperties": {"type": "string"}},
                "latency_ms":   {"type": "integer", "minimum": 0},
                "responses": {
                    "type": "array",
                    "description": "optional sequence responses — each entry matches call counts in [from, to]. status_code overrides the HTTP status for that entry.",
                    "items": {
                        "type": "object",
                        "properties": {
                            "from":        {"type": "integer", "description": "first call count this entry applies to (1-based)"},
                            "to":          {"type": "integer", "description": "last call count (0 = unbounded)"},
                            "status_code": {"type": "integer", "description": "HTTP status code, e.g. 200, 400, 404"},
                            "output":      {"description": "JSON body to return"},
                            "headers":     {"type": "object", "additionalProperties": {"type": "string"}},
                            "latency_ms":  {"type": "integer"}
                        }
                    }
                }
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
			if len(a.Responses) == 0 && (len(a.Response) == 0 || string(a.Response) == "null") {
				return nil, fmt.Errorf("either response or responses is required")
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
				StatusCode:   a.StatusCode,
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

			// Default response body.
			if len(a.Response) > 0 && string(a.Response) != "null" {
				var respData any
				if err := json.Unmarshal(a.Response, &respData); err != nil {
					return nil, fmt.Errorf("invalid response: %w", err)
				}
				outputBSON, err := bson.Marshal(respData)
				if err != nil {
					return nil, fmt.Errorf("marshal response to bson: %w", err)
				}
				req.Output = outputBSON
			}

			// Sequence responses.
			if len(a.Responses) > 0 {
				seqDomain, err := seqArgsToDomain(a.Responses)
				if err != nil {
					return nil, err
				}
				req.Responses = seqDomain
			}

			// Headers.
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
				"id":             req.ID.Hex(),
				"feature":        req.FeatureName,
				"scenario":       req.ScenarioName,
				"name":           req.Name,
				"path":           req.Path,
				"method":         req.Method,
				"hash_input":     req.HashInput,
				"sequence_count": len(req.Responses),
				"created_at":     req.CreatedAt.Format(time.RFC3339),
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
		StatusCode  *int              `json:"status_code"`
		Headers     map[string]string `json:"headers"`
		LatencyMs   *int64            `json:"latency_ms"`
		IsActive    *bool             `json:"is_active"`
		Responses   *[]seqResponseArg `json:"responses"` // nil = don't touch; [] = clear all sequences
	}
	return Tool{
		Name:        "update_mock_api",
		Description: "Update an existing mock API by id. Only provided fields are changed. Supports updating status_code and sequence responses.",
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
                "response":    {"description": "default JSON response"},
                "status_code": {"type": "integer", "description": "HTTP status code for the default response"},
                "headers":     {"type": "object", "additionalProperties": {"type": "string"}},
                "latency_ms":  {"type": "integer", "minimum": 0},
                "is_active":   {"type": "boolean"},
                "responses": {
                    "type": "array",
                    "description": "replace the full sequence responses list",
                    "items": {
                        "type": "object",
                        "properties": {
                            "from":        {"type": "integer"},
                            "to":          {"type": "integer"},
                            "status_code": {"type": "integer"},
                            "output":      {},
                            "headers":     {"type": "object", "additionalProperties": {"type": "string"}},
                            "latency_ms":  {"type": "integer"}
                        }
                    }
                }
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
			if a.StatusCode != nil {
				update["status_code"] = *a.StatusCode
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
			if a.Responses != nil {
				seqDomain, err := seqArgsToDomain(*a.Responses)
				if err != nil {
					return nil, err
				}
				update["responses"] = seqDomain // empty slice clears all sequences
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

// getMockAPICurl returns a ready-to-run cURL command for a mock API.
func getMockAPICurl(d Deps) Tool {
	type args struct {
		Feature   string `json:"feature"`
		Scenario  string `json:"scenario"`
		Name      string `json:"name"`
		AccountID string `json:"account_id"`
		Host      string `json:"host"`
	}
	return Tool{
		Name:        "get_mock_api_curl",
		Description: "Generate a ready-to-run cURL command that calls a mock API on the forward server (port 8082). Includes all required headers (X-Feature-Name, X-Account-Id) and the request body if the API has a hash input.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario", "name"],
            "properties": {
                "feature":    {"type": "string"},
                "scenario":   {"type": "string"},
                "name":       {"type": "string", "description": "mock API name"},
                "account_id": {"type": "string", "description": "X-Account-Id header value, defaults to 'test-account'"},
                "host":       {"type": "string", "description": "forward server host, defaults to 'http://localhost:8082'"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Scenario == "" || a.Name == "" {
				return nil, fmt.Errorf("feature, scenario, and name are required")
			}
			if a.AccountID == "" {
				a.AccountID = "test-account"
			}
			if a.Host == "" {
				a.Host = "http://localhost:8082"
			}

			api, err := d.MockAPI.FindByNameAndFeatureAndScenario(ctx, a.Name, a.Feature, a.Scenario)
			if err != nil || api == nil {
				return nil, fmt.Errorf("mock API %q not found in %s/%s", a.Name, a.Feature, a.Scenario)
			}

			path := api.Path
			if len(path) > 0 && path[0] == '/' {
				path = path[1:]
			}
			apiURL := fmt.Sprintf("%s/forward/%s", a.Host, path)

			// Required routing headers first.
			headerParts := []string{
				"--header 'Content-Type: application/json'",
				fmt.Sprintf("--header 'X-Feature-Name: %s'", a.Feature),
				fmt.Sprintf("--header 'X-Account-Id: %s'", a.AccountID),
			}

			// Append any custom headers stored on the mock API.
			if len(api.Headers) > 0 {
				var customHeaders map[string]string
				if err := bson.Unmarshal(api.Headers, &customHeaders); err == nil {
					for k, v := range customHeaders {
						headerParts = append(headerParts, fmt.Sprintf("--header '%s: %s'", k, v))
					}
				}
			}

			var sb strings.Builder
			for _, h := range headerParts {
				sb.WriteByte(' ')
				sb.WriteString(h)
			}
			headersStr := sb.String()

			// Build body from stored input.
			body := ""
			if len(api.Input) > 0 {
				if b, err := json.Marshal(bsonRawToJSON(api.Input)); err == nil {
					body = string(b)
				}
			}

			curl := fmt.Sprintf("curl --location '%s' --request %s%s", apiURL, api.Method, headersStr)
			if body != "" {
				curl += fmt.Sprintf(" --data-raw '%s'", body)
			}

			return map[string]any{
				"curl":     curl,
				"method":   api.Method,
				"url":      apiURL,
				"feature":  a.Feature,
				"scenario": a.Scenario,
				"name":     a.Name,
			}, nil
		},
	}
}

// resetMockAPICounter clears the sequence-response counter for a mock API
// so the next request starts from the first sequence entry again.
func resetMockAPICounter(d Deps) Tool {
	type args struct {
		Feature  string `json:"feature"`
		Scenario string `json:"scenario"`
		Name     string `json:"name"`
	}
	return Tool{
		Name:        "reset_mock_api_counter",
		Description: "Reset the sequence-response counter for a mock API so the next call starts from the first sequence entry.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario", "name"],
            "properties": {
                "feature":  {"type": "string"},
                "scenario": {"type": "string"},
                "name":     {"type": "string", "description": "mock API name"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Scenario == "" || a.Name == "" {
				return nil, fmt.Errorf("feature, scenario, and name are required")
			}
			api, err := d.MockAPI.FindByNameAndFeatureAndScenario(ctx, a.Name, a.Feature, a.Scenario)
			if err != nil || api == nil {
				return nil, fmt.Errorf("mock API %q not found in %s/%s", a.Name, a.Feature, a.Scenario)
			}
			pattern := fmt.Sprintf("mocktool:seq:%s:%s:*:%s:%s:*",
				api.FeatureName, api.ScenarioName, api.Path, api.Method)
			_ = d.Cache.InvalidAllKey(ctx, pattern)
			return map[string]any{
				"feature":  a.Feature,
				"scenario": a.Scenario,
				"name":     a.Name,
				"reset":    true,
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
