package approval

import (
	"context"
	"time"

	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/risk"
)

type Decision string

const (
	DecisionApprove Decision = "APPROVE"
	DecisionReject  Decision = "REJECT"
)

type Record struct {
	ApprovalID  string     `json:"approval_id"`
	OrderID     string     `json:"order_id"`
	ApproverID  string     `json:"approver_id"`
	Decision    Decision   `json:"decision"`
	Comment     string     `json:"comment,omitempty"`
	RiskLevel   risk.Level `json:"risk_level,omitempty"`
	PlanID      string     `json:"plan_id,omitempty"`
	PlanVersion int        `json:"plan_version,omitempty"`
	ApprovedAt  time.Time  `json:"approved_at,omitempty"`
}

type State struct {
	OrderID        string               `json:"order_id"`
	TraceID        string               `json:"trace_id,omitempty"`
	ApprovalStatus order.ApprovalStatus `json:"approval_status"`
	Status         order.Status         `json:"status"`
	Records        []Record             `json:"records,omitempty"`
}

type CreateInput struct {
	ApprovalPolicyRef string `json:"approval_policy_ref,omitempty"`
}

type DecisionInput struct {
	ApproverID string   `json:"approver_id"`
	Decision   Decision `json:"decision"`
	Comment    string   `json:"comment,omitempty"`
}

type ExpiryResult struct {
	OrderID string `json:"order_id"`
	Expired bool   `json:"expired"`
}

type Service interface {
	Create(ctx context.Context, order order.AssistantOrder, input CreateInput) (State, error)
	Decide(ctx context.Context, orderID string, decision DecisionInput) (State, error)
	Get(ctx context.Context, orderID string) (State, error)
	ExpireStaleApprovals(ctx context.Context, now time.Time, limit int) ([]ExpiryResult, error)
}

type ApprovalService = Service
