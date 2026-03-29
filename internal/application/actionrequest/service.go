package actionrequest

import (
	"context"
	"fmt"
	"sync"
	"time"

	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/action"
	dauth "dba_ai_assistant/internal/domain/authorization"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
	"dba_ai_assistant/internal/domain/task"
)

type service struct {
	principalResolver appauth.PrincipalResolver
	assetResolver     appauth.AssetResolver
	authorization     appauth.AuthorizationService
	executeAuth       appauth.ExecuteAuthorizationService
	planner           appexec.ExecutionPlanner
	audit             appaudit.Service

	mu       sync.Mutex
	sequence int
	orders   map[string]order.AssistantOrder
	plans    map[string]plan.ExecutionPlan
	tasks    map[string]task.ExecutionTask
	traces   map[string]string
}

func NewService(
	principalResolver appauth.PrincipalResolver,
	assetResolver appauth.AssetResolver,
	authorization appauth.AuthorizationService,
	executeAuth appauth.ExecuteAuthorizationService,
	planner appexec.ExecutionPlanner,
	audit appaudit.Service,
) Service {
	return &service{
		principalResolver: principalResolver,
		assetResolver:     assetResolver,
		authorization:     authorization,
		executeAuth:       executeAuth,
		planner:           planner,
		audit:             audit,
		orders:            map[string]order.AssistantOrder{},
		plans:             map[string]plan.ExecutionPlan{},
		tasks:             map[string]task.ExecutionTask{},
		traces:            map[string]string{},
	}
}

func (s *service) Submit(ctx context.Context, req ActionRequestDTO) (ActionSubmissionResult, error) {
	if req.PrincipalID == "" {
		return ActionSubmissionResult{}, common.NewError(common.CodeRequestInvalid, "principal_id is required", nil)
	}
	if req.ActionHint == "" {
		req.ActionHint = string(action.NameMySQLDatabaseCreate)
	}

	now := time.Now().UTC()
	requestID, traceID, orderID := s.nextIDs()
	request := action.Request{
		RequestID:        requestID,
		TraceID:          traceID,
		PrincipalID:      req.PrincipalID,
		ActionName:       action.Name(req.ActionHint),
		ResourceSelector: req.ResourceSelector,
		Parameters:       req.Parameters,
		RequestContext:   req.RequestContext,
		Source:           "api",
		Status:           action.RequestStatusAccepted,
		CreatedAt:        now,
	}

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType:   appaudit.EventRequestAccepted,
		RequestID:   request.RequestID,
		PrincipalID: request.PrincipalID,
		TraceID:     request.TraceID,
	})

	resolvedPrincipal, err := s.principalResolver.Resolve(ctx, req.PrincipalID, appauth.AuthContext{
		AuthenticatedPrincipalID: req.PrincipalID,
		Source:                   "submit",
	})
	if err != nil {
		return ActionSubmissionResult{}, err
	}

	resolvedAssets, err := s.assetResolver.ResolveExact(ctx, req.ActionHint, req.ResourceSelector)
	if err != nil {
		return ActionSubmissionResult{}, err
	}

	authzDecision, err := s.authorization.Evaluate(ctx, dauth.Input{
		ActionName: req.ActionHint,
		Principal:  resolvedPrincipal,
		Assets:     resolvedAssets,
	})
	if err != nil {
		return ActionSubmissionResult{}, err
	}

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventAuthorizationDecided,
		RequestID: request.RequestID,
		OrderID:   orderID,
		TraceID:   traceID,
		Metadata: map[string]any{
			"final_decision": authzDecision.FinalDecision,
			"risk_level":     authzDecision.RiskLevel,
		},
	})

	orderStatus := order.StatusApproved
	approvalStatus := order.ApprovalStatusNotRequired
	if authzDecision.FinalDecision == dauth.FinalDecisionDeny {
		orderStatus = order.StatusPolicyRejected
	}
	if authzDecision.ApprovalRequired {
		orderStatus = order.StatusWaitingApproval
		approvalStatus = order.ApprovalStatusWaitingApproval
	}

	assistantOrder := order.AssistantOrder{
		OrderID:          orderID,
		RequestID:        request.RequestID,
		TraceID:          traceID,
		ActionName:       req.ActionHint,
		ResolvedAssetIDs: append([]string(nil), resolvedAssets.AssetIDs...),
		RiskLevel:        authzDecision.RiskLevel,
		ApprovalRequired: authzDecision.ApprovalRequired,
		ApprovalStatus:   approvalStatus,
		Status:           orderStatus,
		CreatedBy:        req.PrincipalID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	builtPlan, err := s.planner.Build(ctx, assistantOrder)
	if err != nil {
		return ActionSubmissionResult{}, common.NewError(common.CodePlanBuildFailed, "failed to build execution plan", nil)
	}
	builtPlan.OrderID = assistantOrder.OrderID
	builtPlan.PlanStatus = plan.StatusFrozen
	builtPlan.SnapshotFrozen = true

	assistantOrder.PlanID = builtPlan.PlanID
	assistantOrder.PlanVersion = builtPlan.PlanVersion

	s.mu.Lock()
	s.orders[assistantOrder.OrderID] = assistantOrder
	s.plans[assistantOrder.OrderID] = builtPlan
	s.traces[assistantOrder.OrderID] = traceID
	s.mu.Unlock()

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventOrderCreated,
		RequestID: request.RequestID,
		OrderID:   assistantOrder.OrderID,
		TraceID:   traceID,
	})
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventPlanFrozen,
		RequestID: request.RequestID,
		OrderID:   assistantOrder.OrderID,
		TraceID:   traceID,
	})

	userMessage := "请求已创建，无需审批；请通过 execute 显式触发执行。"
	if assistantOrder.Status == order.StatusWaitingApproval {
		userMessage = "请求已创建，等待审批。"
	}

	return ActionSubmissionResult{
		RequestID:        request.RequestID,
		OrderID:          assistantOrder.OrderID,
		ActionName:       request.ActionName,
		Status:           assistantOrder.Status,
		ApprovalRequired: assistantOrder.ApprovalRequired,
		UserMessage:      userMessage,
		NextPollURI:      fmt.Sprintf("/api/v1/orders/%s", assistantOrder.OrderID),
		TraceID:          request.TraceID,
	}, nil
}

func (s *service) GetOrder(_ context.Context, orderID string) (AssistantOrderView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.orders[orderID]
	if !ok {
		return AssistantOrderView{}, common.NewError(common.CodeRequestInvalid, "order not found", map[string]any{"order_id": orderID})
	}
	return value, nil
}

func (s *service) GetTask(_ context.Context, taskID string) (ExecutionTaskView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.tasks[taskID]
	if !ok {
		return ExecutionTaskView{}, common.NewError(common.CodeRequestInvalid, "task not found", map[string]any{"task_id": taskID})
	}
	return value, nil
}

func (s *service) ExecuteApprovedOrder(ctx context.Context, authCtx appauth.AuthContext, input ExecuteOrderInput) (ExecuteOrderResult, error) {
	s.mu.Lock()
	current, ok := s.orders[input.OrderID]
	if !ok {
		s.mu.Unlock()
		return ExecuteOrderResult{}, common.NewError(common.CodeRequestInvalid, "order not found", map[string]any{"order_id": input.OrderID})
	}
	s.mu.Unlock()

	switch current.Status {
	case order.StatusWaitingApproval:
		return ExecuteOrderResult{}, common.NewError(common.CodeApprovalRequired, "order still requires approval", map[string]any{"order_id": current.OrderID})
	case order.StatusPlanStale, order.StatusRejected, order.StatusPolicyRejected, order.StatusExpired, order.StatusCancelled, order.StatusFailed:
		return ExecuteOrderResult{}, common.NewError(common.CodeOrderNotExecutable, "order current status does not allow execution", map[string]any{"order_id": current.OrderID, "status": current.Status})
	case order.StatusSucceeded:
		return ExecuteOrderResult{}, common.NewError(common.CodeOrderAlreadyExecuted, "order already executed", map[string]any{"order_id": current.OrderID})
	case order.StatusExecuting:
		return ExecuteOrderResult{
			OrderID:    current.OrderID,
			Status:     current.Status,
			ExecutorID: authCtx.AuthenticatedPrincipalID,
			TraceID:    s.traces[current.OrderID],
		}, nil
	}

	executor, err := s.principalResolver.Resolve(ctx, authCtx.AuthenticatedPrincipalID, authCtx)
	if err != nil {
		return ExecuteOrderResult{}, err
	}
	if s.executeAuth == nil {
		return ExecuteOrderResult{}, common.NewError(common.CodeSystemInternalError, "execute authorization service is not configured", map[string]any{
			"order_id": current.OrderID,
		})
	}

	decision, err := s.executeAuth.Authorize(ctx, appauth.ExecuteAuthorizationInput{
		ActionName: current.ActionName,
		OrderID:    current.OrderID,
		Executor:   executor,
	})
	if err != nil {
		return ExecuteOrderResult{}, err
	}
	if !decision.Allowed {
		return ExecuteOrderResult{}, common.NewError(common.CodeExecutorNotAllowed, "executor is not allowed to trigger this order", map[string]any{
			"order_id": current.OrderID,
			"reasons":  decision.Reasons,
		})
	}

	return ExecuteOrderResult{
		OrderID:    current.OrderID,
		Status:     current.Status,
		ExecutorID: authCtx.AuthenticatedPrincipalID,
		TraceID:    s.traces[current.OrderID],
	}, nil
}

func (s *service) nextIDs() (string, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequence++
	value := s.sequence
	return fmt.Sprintf("req_%04d", value), fmt.Sprintf("trace_%04d", value), fmt.Sprintf("ord_%04d", value)
}
