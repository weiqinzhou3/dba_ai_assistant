package audit

import (
	"context"
	"time"
)

type EventType string

const (
	EventRequestAccepted      EventType = "REQUEST_ACCEPTED"
	EventAuthorizationDecided EventType = "AUTHORIZATION_DECIDED"
	EventOrderCreated         EventType = "ORDER_CREATED"
	EventPlanFrozen           EventType = "PLAN_FROZEN"
	EventApprovalCreated      EventType = "APPROVAL_CREATED"
	EventApprovalApproved     EventType = "APPROVAL_APPROVED"
	EventApprovalRejected     EventType = "APPROVAL_REJECTED"
	EventApprovalExpired      EventType = "APPROVAL_EXPIRED"
	EventExecuteTriggered     EventType = "EXECUTE_TRIGGERED"
	EventPlanRevalidated      EventType = "PLAN_REVALIDATED"
	EventPlanStale            EventType = "PLAN_STALE"
	EventExecutionStarted     EventType = "EXECUTION_STARTED"
	EventExecutionSucceeded   EventType = "EXECUTION_SUCCEEDED"
	EventExecutionFailed      EventType = "EXECUTION_FAILED"
	EventEvidenceWritten      EventType = "EVIDENCE_WRITTEN"
)

type Event struct {
	EventID         string         `json:"event_id,omitempty"`
	EventType       EventType      `json:"event_type"`
	RequestID       string         `json:"request_id,omitempty"`
	OrderID         string         `json:"order_id,omitempty"`
	TaskID          string         `json:"task_id,omitempty"`
	PrincipalID     string         `json:"principal_id,omitempty"`
	ApprovalActorID string         `json:"approval_actor_id,omitempty"`
	ExecuteActorID  string         `json:"execute_actor_id,omitempty"`
	TraceID         string         `json:"trace_id,omitempty"`
	Success         *bool          `json:"success,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at,omitempty"`
}

type LedgerView struct {
	RequestID              string    `json:"request_id"`
	TraceID                string    `json:"trace_id,omitempty"`
	LatestOrderID          string    `json:"latest_order_id,omitempty"`
	LatestTaskID           string    `json:"latest_task_id,omitempty"`
	LatestOrderStatus      string    `json:"latest_order_status,omitempty"`
	LatestApprovalStatus   string    `json:"latest_approval_status,omitempty"`
	LatestExecutionSummary string    `json:"latest_execution_summary,omitempty"`
	LatestSuccess          *bool     `json:"latest_success,omitempty"`
	LatestErrorCode        string    `json:"latest_error_code,omitempty"`
	LatestErrorMessage     string    `json:"latest_error_message,omitempty"`
	EventCount             int       `json:"event_count"`
	LastEventAt            time.Time `json:"last_event_at,omitempty"`
	Events                 []Event   `json:"events"`
}

type Service interface {
	AppendEvent(ctx context.Context, event Event) error
	ListEventsByRequestID(ctx context.Context, requestID string) ([]Event, error)
	GetViewByRequestID(ctx context.Context, requestID string) (LedgerView, error)
}

type AuditService = Service
