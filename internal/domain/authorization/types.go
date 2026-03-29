package authorization

import (
	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
)

type FinalDecision string

const (
	FinalDecisionAllowNoApproval   FinalDecision = "ALLOW_NO_APPROVAL"
	FinalDecisionAllowWithApproval FinalDecision = "ALLOW_WITH_APPROVAL"
	FinalDecisionDeny              FinalDecision = "DENY"
)

type Input struct {
	ActionName string                 `json:"action_name"`
	Principal  principal.Principal    `json:"principal"`
	Assets     asset.ResolvedAssetSet `json:"assets"`
}

type Decision struct {
	Authorized          bool          `json:"authorized"`
	FinalDecision       FinalDecision `json:"final_decision"`
	ApprovalRequired    bool          `json:"approval_required"`
	RiskLevel           risk.Level    `json:"risk_level"`
	PolicyDecision      policy.Effect `json:"policy_decision"`
	RiskDecision        risk.Effect   `json:"risk_decision"`
	EffectiveExemptions []string      `json:"effective_exemptions,omitempty"`
	DenyReasons         []string      `json:"deny_reasons,omitempty"`
	ApprovalPolicyRef   string        `json:"approval_policy_ref,omitempty"`
}
