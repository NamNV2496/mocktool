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
