package authorization

import (
	"context"
	"testing"

	dauth "dba_ai_assistant/internal/domain/authorization"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
)

func TestAuthorizationServiceStopsAtPolicyDeny(t *testing.T) {
	policyEngine := &stubPolicyEngine{
		decision: policy.Decision{
			BasicAllow:   false,
			ScopeAllowed: false,
			Decision:     policy.EffectDeny,
			DenyReasons:  []string{"role denied"},
		},
	}
	riskEngine := &stubRiskEngine{
		decision: risk.Decision{
			RiskLevel: risk.LevelR2,
			Decision:  risk.EffectRequireApproval,
		},
	}

	service := NewAuthorizationService(policyEngine, riskEngine)
	result, err := service.Evaluate(context.Background(), dauth.Input{
		ActionName: "mysql.database.create",
		Principal: principal.Principal{
			PrincipalID: "u_1001",
			Roles:       []string{"assistant_user"},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.FinalDecision != dauth.FinalDecisionDeny {
		t.Fatalf("expected final deny, got %s", result.FinalDecision)
	}
	if result.Authorized {
		t.Fatalf("expected request to be unauthorized")
	}
	if riskEngine.called {
		t.Fatalf("expected risk engine to be skipped when policy denies")
	}
}

func TestAuthorizationServiceRequiresApprovalWhenRiskDemandsIt(t *testing.T) {
	policyEngine := &stubPolicyEngine{
		decision: policy.Decision{
			BasicAllow:   true,
			ScopeAllowed: true,
			Decision:     policy.EffectAllow,
			MatchedRoles: []string{"mysql_operator"},
		},
	}
	riskEngine := &stubRiskEngine{
		decision: risk.Decision{
			RiskLevel: risk.LevelR2,
			Decision:  risk.EffectRequireApproval,
			Reasons:   []string{"prod change"},
		},
	}

	service := NewAuthorizationService(policyEngine, riskEngine)
	result, err := service.Evaluate(context.Background(), dauth.Input{
		ActionName: "mysql.database.create",
		Principal: principal.Principal{
			PrincipalID: "u_1001",
			Roles:       []string{"mysql_operator"},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.FinalDecision != dauth.FinalDecisionAllowWithApproval {
		t.Fatalf("expected approval-required decision, got %s", result.FinalDecision)
	}
	if !result.Authorized {
		t.Fatalf("expected request to stay authorized pending approval")
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approval requirement to be set")
	}
	if result.RiskLevel != risk.LevelR2 {
		t.Fatalf("expected risk level R2, got %s", result.RiskLevel)
	}
}

type stubPolicyEngine struct {
	decision policy.Decision
}

func (s *stubPolicyEngine) EvaluateBasic(_ context.Context, _ policy.Input) (policy.Decision, error) {
	return s.decision, nil
}

type stubRiskEngine struct {
	decision risk.Decision
	called   bool
}

func (s *stubRiskEngine) Evaluate(_ context.Context, _ risk.Input) (risk.Decision, error) {
	s.called = true
	return s.decision, nil
}
