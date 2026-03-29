package actionrequest

import (
	"context"

	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/action"
	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/task"
)

type ActionRequestDTO struct {
	PrincipalID      string         `json:"principal_id"`
	ActionHint       string         `json:"action_hint"`
	ResourceSelector asset.Selector `json:"resource_selector"`
	Parameters       map[string]any `json:"parameters"`
	RequestContext   map[string]any `json:"request_context,omitempty"`
}

type ActionSubmissionResult struct {
	RequestID        string       `json:"request_id"`
	OrderID          string       `json:"order_id"`
	ActionName       action.Name  `json:"action_name"`
	Status           order.Status `json:"status"`
	ApprovalRequired bool         `json:"approval_required"`
	TaskID           string       `json:"task_id,omitempty"`
	UserMessage      string       `json:"user_message"`
	NextPollURI      string       `json:"next_poll_uri"`
	TraceID          string       `json:"trace_id"`
}

type ExecuteOrderInput struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

type ExecuteOrderResult struct {
	OrderID    string       `json:"order_id"`
	Status     order.Status `json:"status"`
	TaskID     string       `json:"task_id,omitempty"`
	ExecutorID string       `json:"executor_id,omitempty"`
	TraceID    string       `json:"trace_id"`
}

type AssistantOrderView = order.AssistantOrder
type ExecutionTaskView = task.ExecutionTask

type Service interface {
	Submit(ctx context.Context, req ActionRequestDTO) (ActionSubmissionResult, error)
	GetOrder(ctx context.Context, orderID string) (AssistantOrderView, error)
	GetTask(ctx context.Context, taskID string) (ExecutionTaskView, error)
	ExecuteApprovedOrder(ctx context.Context, authCtx appauth.AuthContext, input ExecuteOrderInput) (ExecuteOrderResult, error)
}

type ActionRequestService = Service

type Dependencies struct {
	PrincipalResolver appauth.PrincipalResolver
	AssetResolver     appauth.AssetResolver
	Authorization     appauth.AuthorizationService
	ExecuteAuth       appauth.ExecuteAuthorizationService
	Planner           appexec.ExecutionPlanner
	Audit             appaudit.Service
}
