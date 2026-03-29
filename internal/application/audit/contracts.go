package audit

import "context"

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
	EventType   EventType      `json:"event_type"`
	RequestID   string         `json:"request_id,omitempty"`
	OrderID     string         `json:"order_id,omitempty"`
	TaskID      string         `json:"task_id,omitempty"`
	PrincipalID string         `json:"principal_id,omitempty"`
	TraceID     string         `json:"trace_id,omitempty"`
	Success     bool           `json:"success,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type LedgerView struct {
	RequestID string  `json:"request_id"`
	Events    []Event `json:"events"`
}

type Service interface {
	AppendEvent(ctx context.Context, event Event) error
	ListEventsByRequestID(ctx context.Context, requestID string) ([]Event, error)
	GetViewByRequestID(ctx context.Context, requestID string) (LedgerView, error)
}

type AuditService = Service
