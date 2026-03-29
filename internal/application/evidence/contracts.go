package evidence

import "context"

type Pack struct {
	EvidenceID          string         `json:"evidence_id"`
	OrderID             string         `json:"order_id"`
	TaskID              string         `json:"task_id,omitempty"`
	RequestSummary      string         `json:"request_summary,omitempty"`
	BeforeStateSnapshot map[string]any `json:"before_state_snapshot,omitempty"`
	AfterStateSnapshot  map[string]any `json:"after_state_snapshot,omitempty"`
	ApprovalRefs        []string       `json:"approval_refs,omitempty"`
	ExecutionSuccess    bool           `json:"execution_success"`
	FailureDetail       map[string]any `json:"failure_detail,omitempty"`
	ResultSummary       string         `json:"result_summary,omitempty"`
	RollbackSuggestion  string         `json:"rollback_suggestion,omitempty"`
}

type BuildInput struct {
	OrderID             string         `json:"order_id"`
	TaskID              string         `json:"task_id,omitempty"`
	RequestSummary      string         `json:"request_summary,omitempty"`
	BeforeStateSnapshot map[string]any `json:"before_state_snapshot,omitempty"`
	AfterStateSnapshot  map[string]any `json:"after_state_snapshot,omitempty"`
	ApprovalRefs        []string       `json:"approval_refs,omitempty"`
	ExecutionSuccess    bool           `json:"execution_success"`
	FailureDetail       map[string]any `json:"failure_detail,omitempty"`
	ResultSummary       string         `json:"result_summary,omitempty"`
	RollbackSuggestion  string         `json:"rollback_suggestion,omitempty"`
}

type PackView = Pack

type Service interface {
	Build(ctx context.Context, input BuildInput) (Pack, error)
	GetByOrderID(ctx context.Context, orderID string) (PackView, error)
}

type EvidenceService = Service
