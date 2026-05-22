package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/repository"
)

func listScenarios(d Deps) Tool {
	type args struct {
		Feature  string `json:"feature"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	return Tool{
		Name:        "list_scenarios",
		Description: "List scenarios of a feature, plus a flag indicating which one is currently globally active.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature"],
            "properties": {
                "feature":   {"type": "string"},
                "page":      {"type": "integer", "minimum": 1, "default": 1},
                "page_size": {"type": "integer", "minimum": 1, "maximum": 100, "default": 50}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" {
				return nil, fmt.Errorf("feature is required")
			}
			params := normalizePagination(a.Page, a.PageSize)
			scenarios, total, err := d.Scenario.ListByFeatureNamePaginated(ctx, a.Feature, params)
			if err != nil {
				return nil, err
			}

			active, _ := d.AccountScenario.GetActiveScenario(ctx, a.Feature, nil)
			out := make([]map[string]any, 0, len(scenarios))
			for _, s := range scenarios {
				entry := map[string]any{
					"id":          s.ID.Hex(),
					"name":        s.Name,
					"description": s.Description,
					"is_global_active": active != nil && active.ScenarioID == s.ID,
				}
				out = append(out, entry)
			}
			return map[string]any{
				"scenarios": out,
				"total":     total,
				"page":      params.Page,
				"page_size": params.PageSize,
			}, nil
		},
	}
}

func getActiveScenario(d Deps) Tool {
	type args struct {
		Feature string `json:"feature"`
	}
	return Tool{
		Name:        "get_active_scenario",
		Description: "Return the globally active scenario for a feature, or null if none is active.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature"],
            "properties": {"feature": {"type": "string"}}
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" {
				return nil, fmt.Errorf("feature is required")
			}
			as, err := d.AccountScenario.GetActiveScenario(ctx, a.Feature, nil)
			if err != nil || as == nil {
				return map[string]any{"feature": a.Feature, "active": nil}, nil
			}
			scenario, err := d.Scenario.GetByObjectID(ctx, as.ScenarioID)
			if err != nil || scenario == nil {
				return map[string]any{"feature": a.Feature, "active": nil}, nil
			}
			return map[string]any{
				"feature": a.Feature,
				"active": map[string]any{
					"id":          scenario.ID.Hex(),
					"name":        scenario.Name,
					"description": scenario.Description,
				},
			}, nil
		},
	}
}

// searchScenarios searches scenarios by name within an optional feature.
func searchScenarios(d Deps) Tool {
	type args struct {
		Query    string `json:"query"`
		Feature  string `json:"feature"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	return Tool{
		Name:        "search_scenarios",
		Description: "Search scenarios by name (case-insensitive substring). Optionally filter by feature.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["query"],
            "properties": {
                "query":     {"type": "string", "description": "case-insensitive substring to match on scenario name"},
                "feature":   {"type": "string", "description": "optional feature filter"},
                "page":      {"type": "integer", "minimum": 1, "default": 1},
                "page_size": {"type": "integer", "minimum": 1, "maximum": 100, "default": 50}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Query == "" {
				return nil, fmt.Errorf("query is required")
			}
			params := normalizePagination(a.Page, a.PageSize)
			scenarios, total, err := d.Scenario.SearchByFeatureAndName(ctx, a.Feature, a.Query, params)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"scenarios": scenarios,
				"total":     total,
				"page":      params.Page,
				"page_size": params.PageSize,
			}, nil
		},
	}
}

// updateScenario updates a scenario's description by feature + scenario name.
func updateScenario(d Deps) Tool {
	type args struct {
		Feature     string `json:"feature"`
		Scenario    string `json:"scenario"`
		Description string `json:"description"`
	}
	return Tool{
		Name:        "update_scenario",
		Description: "Update a scenario's description.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario", "description"],
            "properties": {
                "feature":     {"type": "string"},
                "scenario":    {"type": "string"},
                "description": {"type": "string"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Scenario == "" {
				return nil, fmt.Errorf("feature and scenario are required")
			}
			scenario, err := d.Scenario.FindByFeatureNameAndName(ctx, a.Feature, a.Scenario)
			if err != nil || scenario == nil || scenario.Name == "" {
				return nil, fmt.Errorf("scenario %q not found in feature %q", a.Scenario, a.Feature)
			}
			if err := d.Scenario.UpdateByObjectID(ctx, scenario.ID, map[string]any{
				"description": a.Description,
				"updated_at":  time.Now().UTC(),
			}); err != nil {
				return nil, fmt.Errorf("update scenario: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyScnarioTemplate, a.Feature, a.Scenario))
			return map[string]any{"feature": a.Feature, "scenario": a.Scenario, "updated": true}, nil
		},
	}
}

// createScenario creates a new scenario under a feature.
func createScenario(d Deps) Tool {
	type args struct {
		Feature     string `json:"feature"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	return Tool{
		Name:        "create_scenario",
		Description: "Create a new scenario under an existing feature. Scenario name must be unique within the feature.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "name"],
            "properties": {
                "feature":     {"type": "string"},
                "name":        {"type": "string", "description": "unique scenario name within the feature"},
                "description": {"type": "string"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Name == "" {
				return nil, fmt.Errorf("feature and name are required")
			}
			feature, _ := d.Feature.FindByName(ctx, a.Feature)
			if feature == nil || feature.Name == "" {
				return nil, fmt.Errorf("feature %q not found", a.Feature)
			}
			now := time.Now().UTC()
			s := &domain.Scenario{
				FeatureName: a.Feature,
				Name:        a.Name,
				Description: a.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := d.Scenario.Create(ctx, s); err != nil {
				return nil, fmt.Errorf("create scenario: %w", err)
			}
			return map[string]any{
				"feature":     a.Feature,
				"name":        a.Name,
				"description": a.Description,
				"created_at":  now.Format(time.RFC3339),
			}, nil
		},
	}
}

// setScenarioInactive removes the global activation for a specific named scenario.
// If that scenario is not currently active, this is a no-op.
func setScenarioInactive(d Deps) Tool {
	type args struct {
		Feature  string `json:"feature"`
		Scenario string `json:"scenario"`
	}
	return Tool{
		Name:        "set_scenario_inactive",
		Description: "Remove the global active mapping for a specific scenario of a feature. If the scenario is not currently active this is a no-op. Does not delete any data.",
		Destructive: true,
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario"],
            "properties": {
                "feature":  {"type": "string"},
                "scenario": {"type": "string"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Scenario == "" {
				return nil, fmt.Errorf("feature and scenario are required")
			}
			scenario, err := d.Scenario.FindByFeatureNameAndName(ctx, a.Feature, a.Scenario)
			if err != nil || scenario == nil || scenario.Name == "" {
				return nil, fmt.Errorf("scenario %q not found in feature %q", a.Scenario, a.Feature)
			}
			active, _ := d.AccountScenario.GetActiveScenario(ctx, a.Feature, nil)
			if active == nil || active.ScenarioID != scenario.ID {
				return map[string]any{
					"feature":  a.Feature,
					"scenario": a.Scenario,
					"message":  "scenario was not active, nothing changed",
				}, nil
			}
			if err := d.AccountScenario.DeactivateByFeatureAndAccount(ctx, a.Feature, nil); err != nil {
				return nil, fmt.Errorf("deactivate: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyFeatureTemplate, a.Feature))
			return map[string]any{
				"feature":     a.Feature,
				"scenario":    a.Scenario,
				"is_inactive": true,
			}, nil
		},
	}
}

func activateScenario(d Deps) Tool {
	type args struct {
		Feature  string `json:"feature"`
		Scenario string `json:"scenario"`
	}
	return Tool{
		Name:        "activate_scenario",
		Description: "Activate a scenario globally for a feature. Replaces any existing global activation and clears all account-specific mappings.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario"],
            "properties": {
                "feature":  {"type": "string"},
                "scenario": {"type": "string"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Scenario == "" {
				return nil, fmt.Errorf("feature and scenario are required")
			}
			scenario, err := d.Scenario.FindByFeatureNameAndName(ctx, a.Feature, a.Scenario)
			if err != nil || scenario == nil || scenario.Name == "" {
				return nil, fmt.Errorf("scenario %q not found in feature %q", a.Scenario, a.Feature)
			}

			// Global activation: clear all account-specific mappings, then the
			// global mapping, then create a fresh global mapping.
			if err := d.AccountScenario.DeactivateAllAccountSpecificMappings(ctx, a.Feature); err != nil {
				return nil, fmt.Errorf("clear account-specific mappings: %w", err)
			}
			if err := d.AccountScenario.DeactivateByFeatureAndAccount(ctx, a.Feature, nil); err != nil {
				return nil, fmt.Errorf("clear global mapping: %w", err)
			}
			now := time.Now().UTC()
			if err := d.AccountScenario.Create(ctx, &domain.AccountScenario{
				FeatureName: a.Feature,
				ScenarioID:  scenario.ID,
				CreatedAt:   now,
				UpdatedAt:   now,
			}); err != nil {
				return nil, fmt.Errorf("activate: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyFeatureTemplate, a.Feature))
			return map[string]any{
				"feature":     a.Feature,
				"scenario":    a.Scenario,
				"activated":   true,
				"activated_at": now.Format(time.RFC3339),
			}, nil
		},
	}
}

func deactivateScenario(d Deps) Tool {
	type args struct {
		Feature  string `json:"feature"`
		Scenario string `json:"scenario"`
	}
	return Tool{
		Name:        "deactivate_scenario",
		Description: "Remove the global activation for a feature (no scenario will be active). Does not delete the scenario itself.",
		Destructive: true,
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature"],
            "properties": {
                "feature":  {"type": "string"},
                "scenario": {"type": "string", "description": "ignored; reserved for symmetry with activate_scenario"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" {
				return nil, fmt.Errorf("feature is required")
			}
			if err := d.AccountScenario.DeactivateByFeatureAndAccount(ctx, a.Feature, nil); err != nil {
				return nil, fmt.Errorf("deactivate: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyFeatureTemplate, a.Feature))
			return map[string]any{"feature": a.Feature, "deactivated": true}, nil
		},
	}
}

func deleteScenario(d Deps) Tool {
	type args struct {
		Feature  string `json:"feature"`
		Scenario string `json:"scenario"`
	}
	return Tool{
		Name:        "delete_scenario",
		Description: "Delete a scenario and all its mock APIs. Cache is invalidated. Irreversible.",
		Destructive: true,
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature", "scenario"],
            "properties": {
                "feature":  {"type": "string"},
                "scenario": {"type": "string"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Feature == "" || a.Scenario == "" {
				return nil, fmt.Errorf("feature and scenario are required")
			}
			scenario, err := d.Scenario.FindByFeatureNameAndName(ctx, a.Feature, a.Scenario)
			if err != nil || scenario == nil || scenario.Name == "" {
				return nil, fmt.Errorf("scenario %q not found in feature %q", a.Scenario, a.Feature)
			}
			if err := d.Scenario.DeleteByObjectID(ctx, scenario.ID); err != nil {
				return nil, fmt.Errorf("delete scenario: %w", err)
			}
			if err := d.MockAPI.DeleteByScenarioName(ctx, scenario.Name); err != nil {
				return nil, fmt.Errorf("delete mock apis: %w", err)
			}
			if err := d.AccountScenario.DeleteByScenarioId(ctx, scenario.ID); err != nil {
				return nil, fmt.Errorf("delete account-scenario mapping: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyScnarioTemplate, scenario.FeatureName, scenario.Name))
			return map[string]any{
				"feature":    a.Feature,
				"scenario":   a.Scenario,
				"deleted_at": time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}
}
