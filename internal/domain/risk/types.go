package risk

import (
	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
)

type Level string

const (
	LevelR0 Level = "R0"
	LevelR1 Level = "R1"
	LevelR2 Level = "R2"
	LevelR3 Level = "R3"
)

type Effect string

const (
	EffectAllow           Effect = "ALLOW"
	EffectRequireApproval Effect = "REQUIRE_APPROVAL"
	EffectDeny            Effect = "DENY"
)

type Input struct {
	ActionName     string                 `json:"action_name"`
	Principal      principal.Principal    `json:"principal"`
	Assets         asset.ResolvedAssetSet `json:"assets"`
	PolicyDecision policy.Decision        `json:"policy_decision"`
}

type Decision struct {
	RiskLevel           Level          `json:"risk_level"`
	Decision            Effect         `json:"decision"`
	Reasons             []string       `json:"reasons,omitempty"`
	SensitivitySnapshot map[string]any `json:"sensitivity_snapshot,omitempty"`
}
