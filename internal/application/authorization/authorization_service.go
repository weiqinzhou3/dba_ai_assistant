package authorization

import (
	"context"

	dauth "dba_ai_assistant/internal/domain/authorization"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/risk"
)

type DefaultAuthorizationService struct {
	policyEngine PolicyEngine
	riskEngine   RiskEngine
}

func NewAuthorizationService(policyEngine PolicyEngine, riskEngine RiskEngine) *DefaultAuthorizationService {
	return &DefaultAuthorizationService{
		policyEngine: policyEngine,
		riskEngine:   riskEngine,
	}
}

func (s *DefaultAuthorizationService) Evaluate(ctx context.Context, input dauth.Input) (dauth.Decision, error) {
	policyDecision, err := s.policyEngine.EvaluateBasic(ctx, policy.Input{
		ActionName: input.ActionName,
		Principal:  input.Principal,
		Assets:     input.Assets,
	})
	if err != nil {
		return dauth.Decision{}, err
	}

	if policyDecision.Decision == policy.EffectDeny || !policyDecision.BasicAllow || !policyDecision.ScopeAllowed {
		return dauth.Decision{
			Authorized:          false,
			FinalDecision:       dauth.FinalDecisionDeny,
			ApprovalRequired:    false,
			PolicyDecision:      policyDecision.Decision,
			EffectiveExemptions: policyDecision.ApprovalExemptionFlags,
			DenyReasons:         policyDecision.DenyReasons,
		}, nil
	}

	riskDecision, err := s.riskEngine.Evaluate(ctx, risk.Input{
		ActionName:     input.ActionName,
		Principal:      input.Principal,
		Assets:         input.Assets,
		PolicyDecision: policyDecision,
	})
	if err != nil {
		return dauth.Decision{}, err
	}

	decision := dauth.Decision{
		Authorized:          true,
		PolicyDecision:      policyDecision.Decision,
		RiskDecision:        riskDecision.Decision,
		RiskLevel:           riskDecision.RiskLevel,
		EffectiveExemptions: policyDecision.ApprovalExemptionFlags,
	}

	switch riskDecision.Decision {
	case risk.EffectDeny:
		decision.Authorized = false
		decision.FinalDecision = dauth.FinalDecisionDeny
		decision.DenyReasons = riskDecision.Reasons
	case risk.EffectRequireApproval:
		decision.FinalDecision = dauth.FinalDecisionAllowWithApproval
		decision.ApprovalRequired = true
	default:
		decision.FinalDecision = dauth.FinalDecisionAllowNoApproval
	}

	return decision, nil
}
