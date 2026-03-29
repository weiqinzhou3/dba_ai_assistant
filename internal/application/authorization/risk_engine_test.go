package authorization

import (
	"context"
	"testing"

	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
)

func TestStaticRiskEngineRequiresApprovalForProd(t *testing.T) {
	engine := NewStaticRiskEngine()

	decision, err := engine.Evaluate(context.Background(), risk.Input{
		ActionName: "mysql.database.create",
		Principal: principal.Principal{
			PrincipalID: "u_1001",
			Roles:       []string{principal.RoleMySQLOperator},
		},
		Assets: asset.ResolvedAssetSet{
			AssetIDs: []string{"dbt_prod_01"},
			Assets: []asset.Asset{
				{
					AssetID:         "dbt_prod_01",
					Environment:     "prod",
					ServiceInstance: "mysql-order-main",
					Sensitivity:     "normal",
				},
			},
			MatchedExactly: true,
			AssetType:      asset.TypeDatabaseTarget,
		},
		PolicyDecision: policy.Decision{
			BasicAllow:   true,
			ScopeAllowed: true,
			Decision:     policy.EffectAllow,
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.RiskLevel != risk.LevelR2 {
		t.Fatalf("expected prod risk to be R2, got %s", decision.RiskLevel)
	}
	if decision.Decision != risk.EffectRequireApproval {
		t.Fatalf("expected prod risk to require approval, got %s", decision.Decision)
	}
}

func TestStaticRiskEngineRequiresApprovalForHighSensitivityNonProd(t *testing.T) {
	engine := NewStaticRiskEngine()

	decision, err := engine.Evaluate(context.Background(), risk.Input{
		ActionName: "mysql.database.create",
		Principal: principal.Principal{
			PrincipalID: "u_1001",
			Roles:       []string{principal.RoleMySQLOperator},
		},
		Assets: asset.ResolvedAssetSet{
			AssetIDs: []string{"dbt_test_01"},
			Assets: []asset.Asset{
				{
					AssetID:         "dbt_test_01",
					Environment:     "test",
					ServiceInstance: "mysql-order-main",
					Sensitivity:     "high",
				},
			},
			MatchedExactly: true,
			AssetType:      asset.TypeDatabaseTarget,
		},
		PolicyDecision: policy.Decision{
			BasicAllow:   true,
			ScopeAllowed: true,
			Decision:     policy.EffectAllow,
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.Decision != risk.EffectRequireApproval {
		t.Fatalf("expected high-sensitivity asset to require approval, got %s", decision.Decision)
	}
}
