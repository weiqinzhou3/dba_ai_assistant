package authorization

import (
	"context"
	"time"

	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/common"
)

type InMemoryExactAssetResolver struct {
	assets []asset.Asset
}

func NewInMemoryExactAssetResolver(assets []asset.Asset) *InMemoryExactAssetResolver {
	copied := make([]asset.Asset, len(assets))
	copy(copied, assets)
	return &InMemoryExactAssetResolver{assets: copied}
}

func (r *InMemoryExactAssetResolver) ResolveExact(_ context.Context, _ string, selector asset.Selector) (asset.ResolvedAssetSet, error) {
	var matched []asset.Asset

	for _, candidate := range r.assets {
		if candidate.Project != selector.Project {
			continue
		}
		if candidate.Environment != selector.Environment {
			continue
		}
		if candidate.ServiceInstance != selector.ServiceInstance {
			continue
		}
		matched = append(matched, candidate)
	}

	if len(matched) == 0 {
		return asset.ResolvedAssetSet{}, common.NewError(
			common.CodeAssetNotFound,
			"resource selector does not match a managed asset",
			map[string]any{
				"project":          selector.Project,
				"environment":      selector.Environment,
				"service_instance": selector.ServiceInstance,
			},
		)
	}

	if len(matched) > 1 {
		ids := make([]string, 0, len(matched))
		for _, item := range matched {
			ids = append(ids, item.AssetID)
		}
		return asset.ResolvedAssetSet{}, common.NewError(
			common.CodeAssetAmbiguous,
			"resource selector matches more than one managed asset",
			map[string]any{"matched_asset_ids": ids},
		)
	}

	return asset.ResolvedAssetSet{
		AssetIDs:       []string{matched[0].AssetID},
		Assets:         matched,
		MatchedExactly: true,
		AssetType:      matched[0].AssetType,
		ResolvedAt:     time.Now().UTC(),
	}, nil
}
