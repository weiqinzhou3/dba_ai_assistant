package order

import (
	"time"

	"dba_ai_assistant/internal/domain/risk"
)

type ApprovalStatus string

const (
	ApprovalStatusNotRequired     ApprovalStatus = "NOT_REQUIRED"
	ApprovalStatusWaitingApproval ApprovalStatus = "WAITING_APPROVAL"
	ApprovalStatusApproved        ApprovalStatus = "APPROVED"
	ApprovalStatusRejected        ApprovalStatus = "REJECTED"
	ApprovalStatusExpired         ApprovalStatus = "EXPIRED"
)

type Status string

const (
	StatusDraft           Status = "DRAFT"
	StatusWaitingApproval Status = "WAITING_APPROVAL"
	StatusApproved        Status = "APPROVED"
	StatusExecuting       Status = "EXECUTING"
	StatusSucceeded       Status = "SUCCEEDED"
	StatusFailed          Status = "FAILED"
	StatusPlanStale       Status = "PLAN_STALE"
	StatusPolicyRejected  Status = "POLICY_REJECTED"
	StatusRejected        Status = "REJECTED"
	// REVIEW: formal docs use order terminal state EXPIRED; APPROVAL_EXPIRED is the audit event, not a second order status.
	StatusExpired   Status = "EXPIRED"
	StatusCancelled Status = "CANCELLED"
)

type AssistantOrder struct {
	OrderID                string         `json:"order_id"`
	RequestID              string         `json:"request_id"`
	TraceID                string         `json:"trace_id"`
	ActionName             string         `json:"action_name"`
	ResolvedAssetIDs       []string       `json:"resolved_asset_ids"`
	RiskLevel              risk.Level     `json:"risk_level"`
	ApprovalRequired       bool           `json:"approval_required"`
	ApprovalStatus         ApprovalStatus `json:"approval_status"`
	Status                 Status         `json:"status"`
	PlanID                 string         `json:"plan_id"`
	PlanVersion            int            `json:"plan_version"`
	CreatedBy              string         `json:"created_by"`
	LastExecuteTriggeredBy string         `json:"last_execute_triggered_by,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}
