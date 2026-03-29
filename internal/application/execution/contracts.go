package execution

import (
	"context"
	"time"

	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
	"dba_ai_assistant/internal/domain/task"
)

type ExecutionPlanner interface {
	Build(ctx context.Context, order order.AssistantOrder) (plan.ExecutionPlan, error)
	Revalidate(ctx context.Context, order order.AssistantOrder, plan plan.ExecutionPlan) (PlanValidationResult, error)
}

type ExecutionRouter interface {
	Route(ctx context.Context, plan plan.ExecutionPlan) (AdapterBinding, error)
}

type TaskRuntime interface {
	Start(ctx context.Context, input TaskStartInput) (TaskStartResult, error)
	Get(ctx context.Context, taskID string) (task.ExecutionTask, error)
}

type PlanValidationResult struct {
	Valid  bool        `json:"valid"`
	Status plan.Status `json:"status"`
	Reason string      `json:"reason,omitempty"`
}

type AdapterBinding struct {
	AdapterType string  `json:"adapter_type"`
	RouteName   string  `json:"route_name"`
	Adapter     Adapter `json:"-"`
}

type AdapterType string

const (
	AdapterTypeDBNative AdapterType = "db_native"
)

type Adapter interface {
	Type() AdapterType
	Supports(ctx context.Context, req AdapterCapabilityRequest) (bool, error)
	DryRun(ctx context.Context, req AdapterExecutionRequest) (AdapterDryRunResult, error)
	Execute(ctx context.Context, req AdapterExecutionRequest) (AdapterExecutionResult, error)
}

type AdapterCapabilityRequest struct {
	ActionName         string         `json:"action_name"`
	TargetAssetType    string         `json:"target_asset_type"`
	TargetAdapterHints []string       `json:"target_adapter_hints,omitempty"`
	Environment        string         `json:"environment"`
	Parameters         map[string]any `json:"parameters,omitempty"`
}

type AdapterExecutionRequest struct {
	TraceID              string         `json:"trace_id"`
	OrderID              string         `json:"order_id"`
	TaskID               string         `json:"task_id"`
	StepID               string         `json:"step_id"`
	ActionName           string         `json:"action_name"`
	IdempotencyKey       string         `json:"idempotency_key"`
	Target               map[string]any `json:"target"`
	Parameters           map[string]any `json:"parameters,omitempty"`
	ExecutionControls    map[string]any `json:"execution_controls,omitempty"`
	EvidenceRequirements map[string]any `json:"evidence_requirements,omitempty"`
}

type AdapterDryRunResult struct {
	Supported       bool           `json:"supported"`
	Ready           bool           `json:"ready"`
	Issues          []string       `json:"issues,omitempty"`
	RenderedPreview map[string]any `json:"rendered_preview,omitempty"`
}

type AdapterExecutionResult struct {
	Success         bool             `json:"success"`
	Status          string           `json:"status"`
	ProviderTaskID  string           `json:"provider_task_id,omitempty"`
	ProviderStepRef string           `json:"provider_step_ref,omitempty"`
	Summary         string           `json:"summary,omitempty"`
	Outputs         map[string]any   `json:"outputs,omitempty"`
	Artifacts       []map[string]any `json:"artifacts,omitempty"`
	StartedAt       time.Time        `json:"started_at,omitempty"`
	EndedAt         time.Time        `json:"ended_at,omitempty"`
	Error           map[string]any   `json:"error,omitempty"`
}

type TaskStartInput struct {
	Order          order.AssistantOrder `json:"order"`
	Plan           plan.ExecutionPlan   `json:"plan"`
	Binding        AdapterBinding       `json:"binding"`
	AdapterRequest AdapterExecutionRequest
}

type TaskStartResult struct {
	Task           task.ExecutionTask      `json:"task"`
	AdapterRequest AdapterExecutionRequest `json:"adapter_request"`
	AdapterResult  AdapterExecutionResult  `json:"adapter_result"`
}

type IdempotencyRecord struct {
	IdempotencyKey string           `json:"idempotency_key"`
	OrderID        string           `json:"order_id"`
	TaskID         string           `json:"task_id,omitempty"`
	TaskStatus     task.Status      `json:"task_status"`
	Summary        string           `json:"summary,omitempty"`
	Outputs        map[string]any   `json:"outputs,omitempty"`
	Artifacts      []map[string]any `json:"artifacts,omitempty"`
	Error          map[string]any   `json:"error,omitempty"`
	UpdatedAt      time.Time        `json:"updated_at,omitempty"`
}

type IdempotencyRepository interface {
	SaveIdempotencyRecord(ctx context.Context, record IdempotencyRecord) error
	GetIdempotencyRecord(ctx context.Context, key string) (IdempotencyRecord, bool, error)
}
