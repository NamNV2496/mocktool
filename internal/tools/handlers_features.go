package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/namnv2496/mocktool/internal/domain"
	"github.com/namnv2496/mocktool/internal/repository"
)

// listFeatures returns up to page_size features. Default page=1, page_size=50.
func listFeatures(d Deps) Tool {
	type args struct {
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
		Query    string `json:"query"`
	}
	return Tool{
		Name:        "list_features",
		Description: "List all mocktool features with their metadata. Supports pagination and optional case-insensitive name search via 'query'.",
		InputSchema: schema(`{
            "type": "object",
            "properties": {
                "page":      {"type": "integer", "minimum": 1, "default": 1},
                "page_size": {"type": "integer", "minimum": 1, "maximum": 100, "default": 50},
                "query":     {"type": "string", "description": "optional case-insensitive substring filter on feature name"}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			params := normalizePagination(a.Page, a.PageSize)

			var (
				features []domain.Feature
				total    int64
				err      error
			)
			if a.Query == "" {
				features, total, err = d.Feature.ListAllPaginated(ctx, params)
			} else {
				features, total, err = d.Feature.SearchByName(ctx, a.Query, params)
			}
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"features":  features,
				"total":     total,
				"page":      params.Page,
				"page_size": params.PageSize,
			}, nil
		},
	}
}

// deleteFeature deletes a feature and cascades to its scenarios and mock APIs.
// Destructive: requires Slack-side confirmation before being invoked.
func deleteFeature(d Deps) Tool {
	type args struct {
		Feature string `json:"feature"`
	}
	return Tool{
		Name:        "delete_feature",
		Description: "Delete a feature and ALL its scenarios + mock APIs. Irreversible. Cache is invalidated.",
		Destructive: true,
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature"],
            "properties": {
                "feature": {"type": "string", "description": "feature name"}
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

			feature, _ := d.Feature.FindByName(ctx, a.Feature)
			if feature == nil || feature.Name == "" {
				return nil, fmt.Errorf("feature %q not found", a.Feature)
			}

			if err := d.Feature.DeleteById(ctx, feature.ID); err != nil {
				return nil, fmt.Errorf("delete feature: %w", err)
			}
			if err := d.Scenario.DeleteByFeatureName(ctx, feature.Name); err != nil {
				return nil, fmt.Errorf("delete scenarios: %w", err)
			}
			if err := d.MockAPI.DeleteByFeatureName(ctx, feature.Name); err != nil {
				return nil, fmt.Errorf("delete mock apis: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyFeatureTemplate, feature.Name))

			return map[string]any{
				"feature":    feature.Name,
				"deleted_at": time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}
}

// createFeature creates a new feature.
func createFeature(d Deps) Tool {
	type args struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Active      *bool  `json:"active"`
	}
	return Tool{
		Name:        "create_feature",
		Description: "Create a new mocktool feature. Name must be unique.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["name"],
            "properties": {
                "name":        {"type": "string", "description": "unique feature name"},
                "description": {"type": "string"},
                "active":      {"type": "boolean", "default": true}
            }
        }`),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := decodeArgs(raw, &a); err != nil {
				return nil, err
			}
			if a.Name == "" {
				return nil, fmt.Errorf("name is required")
			}
			active := true
			if a.Active != nil {
				active = *a.Active
			}
			now := time.Now().UTC()
			f := &domain.Feature{
				Name:        a.Name,
				Description: a.Description,
				IsActive:    active,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := d.Feature.Create(ctx, f); err != nil {
				return nil, fmt.Errorf("create feature: %w", err)
			}
			return map[string]any{
				"name":        f.Name,
				"description": f.Description,
				"is_active":   f.IsActive,
				"created_at":  now.Format(time.RFC3339),
			}, nil
		},
	}
}

// updateFeature updates a feature's description and/or active status by name.
func updateFeature(d Deps) Tool {
	type args struct {
		Feature     string  `json:"feature"`
		Description *string `json:"description"`
		Active      *bool   `json:"active"`
	}
	return Tool{
		Name:        "update_feature",
		Description: "Update a feature's description or active status. At least one of description or active must be provided.",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature"],
            "properties": {
                "feature":     {"type": "string", "description": "feature name to update"},
                "description": {"type": "string"},
                "active":      {"type": "boolean"}
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
			if a.Description == nil && a.Active == nil {
				return nil, fmt.Errorf("at least one of description or active must be provided")
			}
			feature, _ := d.Feature.FindByName(ctx, a.Feature)
			if feature == nil || feature.Name == "" {
				return nil, fmt.Errorf("feature %q not found", a.Feature)
			}
			update := map[string]any{"updated_at": time.Now().UTC()}
			if a.Description != nil {
				update["description"] = *a.Description
			}
			if a.Active != nil {
				update["is_active"] = *a.Active
			}
			if err := d.Feature.UpdateByObjectID(ctx, feature.ID, update); err != nil {
				return nil, fmt.Errorf("update feature: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyFeatureTemplate, a.Feature))
			return map[string]any{"feature": a.Feature, "updated": true}, nil
		},
	}
}

// enableFeature sets a feature's is_active flag to true.
func enableFeature(d Deps) Tool {
	type args struct {
		Feature string `json:"feature"`
	}
	return Tool{
		Name:        "enable_feature",
		Description: "Enable a feature (set is_active=true).",
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature"],
            "properties": {
                "feature": {"type": "string"}
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
			feature, _ := d.Feature.FindByName(ctx, a.Feature)
			if feature == nil || feature.Name == "" {
				return nil, fmt.Errorf("feature %q not found", a.Feature)
			}
			if err := d.Feature.UpdateByObjectID(ctx, feature.ID, map[string]any{
				"is_active":  true,
				"updated_at": time.Now().UTC(),
			}); err != nil {
				return nil, fmt.Errorf("enable feature: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyFeatureTemplate, a.Feature))
			return map[string]any{"feature": a.Feature, "is_active": true}, nil
		},
	}
}

// disableFeature sets a feature's is_active flag to false without deleting it.
func disableFeature(d Deps) Tool {
	type args struct {
		Feature string `json:"feature"`
	}
	return Tool{
		Name:        "disable_feature",
		Description: "Disable a feature (set is_active=false). The feature and its data are preserved; only its active flag is cleared.",
		Destructive: true,
		InputSchema: schema(`{
            "type": "object",
            "required": ["feature"],
            "properties": {
                "feature": {"type": "string", "description": "feature name"}
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
			feature, _ := d.Feature.FindByName(ctx, a.Feature)
			if feature == nil || feature.Name == "" {
				return nil, fmt.Errorf("feature %q not found", a.Feature)
			}
			if err := d.Feature.UpdateByObjectID(ctx, feature.ID, map[string]any{
				"is_active":  false,
				"updated_at": time.Now().UTC(),
			}); err != nil {
				return nil, fmt.Errorf("disable feature: %w", err)
			}
			_ = d.Cache.InvalidAllKey(ctx, fmt.Sprintf(repository.KeyFeatureTemplate, a.Feature))
			return map[string]any{"feature": a.Feature, "is_active": false}, nil
		},
	}
}

func normalizePagination(page, pageSize int) domain.PaginationParams {
	if page <= 0 {
		page = domain.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > domain.MaxPageSize {
		pageSize = domain.MaxPageSize
	}
	p := domain.PaginationParams{Page: page, PageSize: pageSize}
	p.Normalize()
	return p
}
