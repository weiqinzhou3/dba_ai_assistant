package authorization

import (
	"context"
	"strings"

	"dba_ai_assistant/internal/domain/asset"
	dauth "dba_ai_assistant/internal/domain/authorization"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
)

type StaticPrincipalResolver struct{}

func NewStaticPrincipalResolver() *StaticPrincipalResolver {
	return &StaticPrincipalResolver{}
}

func (r *StaticPrincipalResolver) Resolve(_ context.Context, principalID string, authCtx AuthContext) (principal.Principal, error) {
	id := principalID
	if id == "" {
		id = authCtx.AuthenticatedPrincipalID
	}
	if id == "" {
		return principal.Principal{}, common.NewError(common.CodePrincipalNotFound, "principal id is required", nil)
	}

	roles := append([]string(nil), authCtx.Roles...)
	if len(roles) == 0 {
		roles = []string{principal.RoleMySQLOperator}
	}

	return principal.Principal{
		PrincipalID:   id,
		PrincipalType: principal.TypeHuman,
		DisplayName:   id,
		Roles:         roles,
		IsActive:      true,
	}, nil
}

type StaticPolicyEngine struct{}

func NewStaticPolicyEngine() *StaticPolicyEngine {
	return &StaticPolicyEngine{}
}

func (e *StaticPolicyEngine) EvaluateBasic(_ context.Context, input policy.Input) (policy.Decision, error) {
	for _, role := range input.Principal.Roles {
		if role == principal.RoleAssistantUser {
			return policy.Decision{
				BasicAllow:   false,
				ScopeAllowed: false,
				Decision:     policy.EffectDeny,
				DenyReasons:  []string{"assistant_user cannot submit controlled actions"},
			}, nil
		}
	}

	return policy.Decision{
		BasicAllow:   true,
		ScopeAllowed: true,
		MatchedRoles: append([]string(nil), input.Principal.Roles...),
		Decision:     policy.EffectAllow,
	}, nil
}

type StaticRiskEngine struct{}

func NewStaticRiskEngine() *StaticRiskEngine {
	return &StaticRiskEngine{}
}

func (e *StaticRiskEngine) Evaluate(_ context.Context, input risk.Input) (risk.Decision, error) {
	environment := ""
	sensitivity := ""
	if len(input.Assets.Assets) > 0 {
		environment = strings.ToLower(input.Assets.Assets[0].Environment)
		sensitivity = strings.ToLower(input.Assets.Assets[0].Sensitivity)
	}

	if sensitivity == "high" || sensitivity == "critical" {
		return risk.Decision{
			RiskLevel: risk.LevelR2,
			Decision:  risk.EffectRequireApproval,
			Reasons:   []string{"high-sensitivity target requires explicit approval"},
			SensitivitySnapshot: map[string]any{
				"sensitivity": sensitivity,
			},
		}, nil
	}

	if environment == "prod" {
		return risk.Decision{
			RiskLevel: risk.LevelR2,
			Decision:  risk.EffectRequireApproval,
			Reasons:   []string{"prod change requires explicit approval"},
			SensitivitySnapshot: map[string]any{
				"sensitivity": sensitivity,
			},
		}, nil
	}

	return risk.Decision{
		RiskLevel: risk.LevelR1,
		Decision:  risk.EffectAllow,
		Reasons:   []string{"non-prod environment"},
		SensitivitySnapshot: map[string]any{
			"sensitivity": sensitivity,
		},
	}, nil
}

type StaticExecuteAuthorizationService struct{}

func NewStaticExecuteAuthorizationService() *StaticExecuteAuthorizationService {
	return &StaticExecuteAuthorizationService{}
}

func (s *StaticExecuteAuthorizationService) Authorize(_ context.Context, input ExecuteAuthorizationInput) (ExecuteAuthorizationDecision, error) {
	if input.ActionName == "" || input.OrderID == "" || input.Executor.PrincipalID == "" {
		return ExecuteAuthorizationDecision{
			Allowed: false,
			Reasons: []string{"action_name, order_id, and executor are required"},
		}, nil
	}

	for _, role := range input.Executor.Roles {
		switch role {
		case principal.RoleMySQLOperator, principal.RolePlatformAdmin, principal.RoleControlExecutor:
			return ExecuteAuthorizationDecision{
				Allowed: true,
			}, nil
		}
	}

	return ExecuteAuthorizationDecision{
		Allowed: false,
		Reasons: []string{"executor does not match ExecutePolicy"},
	}, nil
}

func StaticManagedAssets() []asset.Asset {
	return []asset.Asset{
		{
			AssetID:         "dbt_1001",
			AssetType:       asset.TypeDatabaseTarget,
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
			CanonicalName:   "mysql-order-main",
			Sensitivity:     "normal",
			ConnectionRef:   "secret://db-targets/mysql-order-main-test",
		},
		{
			AssetID:         "dbt_2001",
			AssetType:       asset.TypeDatabaseTarget,
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
			CanonicalName:   "mysql-order-main",
			Sensitivity:     "high",
			ConnectionRef:   "secret://db-targets/mysql-order-main-prod",
		},
	}
}

func StaticAuthorizationDecisionDenied(reason string) dauth.Decision {
	return dauth.Decision{
		Authorized:    false,
		FinalDecision: dauth.FinalDecisionDeny,
		DenyReasons:   []string{reason},
	}
}
