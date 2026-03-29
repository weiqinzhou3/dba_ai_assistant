package persistence

import (
	"context"

	"dba_ai_assistant/internal/application/approval"
	"dba_ai_assistant/internal/application/audit"
	"dba_ai_assistant/internal/application/evidence"
	"dba_ai_assistant/internal/domain/action"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/task"
)

type ActionRequestRepository interface {
	Save(ctx context.Context, request action.Request) error
	Get(ctx context.Context, requestID string) (action.Request, error)
}

type OrderRepository interface {
	Save(ctx context.Context, ord order.AssistantOrder) error
	Get(ctx context.Context, orderID string) (order.AssistantOrder, error)
}

type PlanRepository interface {
	Save(ctx context.Context, executionPlan plan.ExecutionPlan) error
	GetByOrderID(ctx context.Context, orderID string) (plan.ExecutionPlan, error)
}

type TaskRepository interface {
	Save(ctx context.Context, executionTask task.ExecutionTask) error
	Get(ctx context.Context, taskID string) (task.ExecutionTask, error)
}

type ApprovalRepository interface {
	Save(ctx context.Context, state approval.State) error
	GetByOrderID(ctx context.Context, orderID string) (approval.State, error)
}

type ApprovalPolicyRepository interface {
	Save(ctx context.Context, approvalPolicy policy.ApprovalPolicy) error
	ListByAction(ctx context.Context, actionName string) ([]policy.ApprovalPolicy, error)
}

type ExecutePolicyRepository interface {
	Save(ctx context.Context, executePolicy policy.ExecutePolicy) error
	ListByAction(ctx context.Context, actionName string) ([]policy.ExecutePolicy, error)
}

type AuditRepository interface {
	Append(ctx context.Context, event audit.Event) error
	ListByRequestID(ctx context.Context, requestID string) ([]audit.Event, error)
}

type EvidenceRepository interface {
	Save(ctx context.Context, pack evidence.Pack) error
	GetByOrderID(ctx context.Context, orderID string) (evidence.Pack, error)
}
