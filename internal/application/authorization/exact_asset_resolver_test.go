package authorization

import (
	"context"
	"testing"

	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/common"
)

func TestInMemoryExactAssetResolverResolvesSingleExactMatch(t *testing.T) {
	resolver := NewInMemoryExactAssetResolver([]asset.Asset{
		{
			AssetID:         "dbt_1001",
			AssetType:       asset.TypeDatabaseTarget,
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
			CanonicalName:   "mysql-order-main",
		},
	})

	result, err := resolver.ResolveExact(context.Background(), "mysql.database.create", asset.Selector{
		Project:         "order-platform",
		Environment:     "prod",
		ServiceInstance: "mysql-order-main",
	})
	if err != nil {
		t.Fatalf("ResolveExact returned error: %v", err)
	}

	if len(result.AssetIDs) != 1 || result.AssetIDs[0] != "dbt_1001" {
		t.Fatalf("unexpected asset ids: %+v", result.AssetIDs)
	}
	if !result.MatchedExactly {
		t.Fatalf("expected exact match to be recorded")
	}
}

func TestInMemoryExactAssetResolverReturnsAssetNotFoundOnCaseMismatch(t *testing.T) {
	resolver := NewInMemoryExactAssetResolver([]asset.Asset{
		{
			AssetID:         "dbt_1001",
			AssetType:       asset.TypeDatabaseTarget,
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
			CanonicalName:   "mysql-order-main",
		},
	})

	_, err := resolver.ResolveExact(context.Background(), "mysql.database.create", asset.Selector{
		Project:         "order-platform",
		Environment:     "prod",
		ServiceInstance: "MySQL-Order-Main",
	})
	if err == nil {
		t.Fatalf("expected error for non-exact selector")
	}
	if code := common.ErrorCode(err); code != common.CodeAssetNotFound {
		t.Fatalf("expected %s, got %s", common.CodeAssetNotFound, code)
	}
}

func TestInMemoryExactAssetResolverReturnsAmbiguousWhenMultipleExactMatchesExist(t *testing.T) {
	resolver := NewInMemoryExactAssetResolver([]asset.Asset{
		{
			AssetID:         "dbt_1001",
			AssetType:       asset.TypeDatabaseTarget,
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
			CanonicalName:   "mysql-order-main",
		},
		{
			AssetID:         "dbt_1002",
			AssetType:       asset.TypeDatabaseTarget,
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
			CanonicalName:   "mysql-order-main",
		},
	})

	_, err := resolver.ResolveExact(context.Background(), "mysql.database.create", asset.Selector{
		Project:         "order-platform",
		Environment:     "prod",
		ServiceInstance: "mysql-order-main",
	})
	if err == nil {
		t.Fatalf("expected ambiguity error")
	}
	if code := common.ErrorCode(err); code != common.CodeAssetAmbiguous {
		t.Fatalf("expected %s, got %s", common.CodeAssetAmbiguous, code)
	}
}
