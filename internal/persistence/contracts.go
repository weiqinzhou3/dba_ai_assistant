package persistence

import (
	"context"

	"dba_ai_assistant/internal/application/approval"
	"dba_ai_assistant/internal/application/audit"
	"dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/action"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/task"
)

type ActionRequestRepository interface {
	SaveRequest(ctx context.Context, request action.Request) error
	GetRequest(ctx context.Context, requestID string) (action.Request, error)
}

type OrderRepository interface {
	SaveOrder(ctx context.Context, ord order.AssistantOrder) error
	GetOrder(ctx context.Context, orderID string) (order.AssistantOrder, error)
}

type PlanRepository interface {
	SavePlan(ctx context.Context, executionPlan plan.ExecutionPlan) error
	GetPlanByOrderID(ctx context.Context, orderID string) (plan.ExecutionPlan, error)
}

type TaskRepository interface {
	SaveTask(ctx context.Context, executionTask task.ExecutionTask) error
	GetTask(ctx context.Context, taskID string) (task.ExecutionTask, error)
	GetTaskByOrderID(ctx context.Context, orderID string) (task.ExecutionTask, error)
}

type ApprovalRepository interface {
	SaveApprovalState(ctx context.Context, state approval.State) error
	GetApprovalStateByOrderID(ctx context.Context, orderID string) (approval.State, error)
	ListWaitingApprovalStates(ctx context.Context, limit int) ([]approval.State, error)
}

type ApprovalPolicyRepository interface {
	SaveApprovalPolicy(ctx context.Context, approvalPolicy policy.ApprovalPolicy) error
	ListApprovalPoliciesByAction(ctx context.Context, actionName string) ([]policy.ApprovalPolicy, error)
}

type ExecutePolicyRepository interface {
	SaveExecutePolicy(ctx context.Context, executePolicy policy.ExecutePolicy) error
	ListExecutePoliciesByAction(ctx context.Context, actionName string) ([]policy.ExecutePolicy, error)
}

type AuditRepository interface {
	AppendAuditEvent(ctx context.Context, event audit.Event) error
	ListAuditEventsByRequestID(ctx context.Context, requestID string) ([]audit.Event, error)
}

type EvidenceRepository interface {
	SaveEvidencePack(ctx context.Context, pack evidence.Pack) error
	GetEvidencePackByOrderID(ctx context.Context, orderID string) (evidence.Pack, error)
}

type IdempotencyRepository interface {
	SaveIdempotencyRecord(ctx context.Context, record appexec.IdempotencyRecord) error
	GetIdempotencyRecord(ctx context.Context, key string) (appexec.IdempotencyRecord, bool, error)
}
