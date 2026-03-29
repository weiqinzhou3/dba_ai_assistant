package policy

import (
	"time"

	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/principal"
)

type SubjectType string

const (
	SubjectTypeRole      SubjectType = "role"
	SubjectTypeGroup     SubjectType = "group"
	SubjectTypePrincipal SubjectType = "principal"
)

type Effect string

const (
	EffectAllow Effect = "ALLOW"
	EffectDeny  Effect = "DENY"
)

type Input struct {
	ActionName string                 `json:"action_name"`
	Principal  principal.Principal    `json:"principal"`
	Assets     asset.ResolvedAssetSet `json:"assets"`
}

type Decision struct {
	BasicAllow             bool     `json:"basic_allow"`
	ScopeAllowed           bool     `json:"scope_allowed"`
	MatchedRoles           []string `json:"matched_roles,omitempty"`
	Decision               Effect   `json:"decision"`
	DenyReasons            []string `json:"deny_reasons,omitempty"`
	ApprovalExemptionFlags []string `json:"approval_exemption_flags,omitempty"`
}

type ApprovalPolicy struct {
	PolicyID           string        `json:"policy_id"`
	ActionName         string        `json:"action_name"`
	Environment        []string      `json:"environment,omitempty"`
	RiskLevels         []string      `json:"risk_levels,omitempty"`
	ApproverRoles      []string      `json:"approver_roles,omitempty"`
	MinApproverCount   int           `json:"min_approver_count,omitempty"`
	ForbidSelfApproval bool          `json:"forbid_self_approval"`
	TTL                time.Duration `json:"ttl"`
	Enabled            bool          `json:"enabled"`
}

type ExecutePolicy struct {
	PolicyID    string      `json:"policy_id"`
	SubjectType SubjectType `json:"subject_type"`
	SubjectRef  string      `json:"subject_ref"`
	ActionName  string      `json:"action_name"`
	Environment []string    `json:"environment,omitempty"`
	Effect      Effect      `json:"effect"`
	Enabled     bool        `json:"enabled"`
}
