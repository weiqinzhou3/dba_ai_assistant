package authorization

import (
	"context"

	"dba_ai_assistant/internal/domain/asset"
	dauth "dba_ai_assistant/internal/domain/authorization"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
)

type AssetResolver interface {
	ResolveExact(ctx context.Context, actionName string, selector asset.Selector) (asset.ResolvedAssetSet, error)
}

type PrincipalResolver interface {
	Resolve(ctx context.Context, principalID string, authCtx AuthContext) (principal.Principal, error)
}

type PolicyEngine interface {
	EvaluateBasic(ctx context.Context, input policy.Input) (policy.Decision, error)
}

type RiskEngine interface {
	Evaluate(ctx context.Context, input risk.Input) (risk.Decision, error)
}

type AuthorizationService interface {
	Evaluate(ctx context.Context, input dauth.Input) (dauth.Decision, error)
}

type ExecuteAuthorizationService interface {
	Authorize(ctx context.Context, input ExecuteAuthorizationInput) (ExecuteAuthorizationDecision, error)
}

type AuthContext struct {
	AuthenticatedPrincipalID string   `json:"authenticated_principal_id"`
	Roles                    []string `json:"roles,omitempty"`
	Source                   string   `json:"source,omitempty"`
}

type ExecuteAuthorizationInput struct {
	ActionName string              `json:"action_name"`
	OrderID    string              `json:"order_id"`
	Executor   principal.Principal `json:"executor"`
}

type ExecuteAuthorizationDecision struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}
