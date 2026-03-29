package actionrequest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	appapproval "dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/action"
	dauth "dba_ai_assistant/internal/domain/authorization"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
)

type service struct {
	principalResolver appauth.PrincipalResolver
	assetResolver     appauth.AssetResolver
	authorization     appauth.AuthorizationService
	executeAuth       appauth.ExecuteAuthorizationService
	planner           appexec.ExecutionPlanner
	router            appexec.ExecutionRouter
	runtime           appexec.TaskRuntime
	approval          appapproval.Service
	audit             appaudit.Service
	evidence          appevidence.Service
	requests          requestRepository
	orders            orderRepository
	plans             planRepository
	tasks             taskRepository

	mu       sync.Mutex
	sequence int
}

func NewService(deps Dependencies) Service {
	return &service{
		principalResolver: deps.PrincipalResolver,
		assetResolver:     deps.AssetResolver,
		authorization:     deps.Authorization,
		executeAuth:       deps.ExecuteAuth,
		planner:           deps.Planner,
		router:            deps.Router,
		runtime:           deps.Runtime,
		approval:          deps.Approval,
		audit:             deps.Audit,
		evidence:          deps.Evidence,
		requests:          deps.Requests,
		orders:            deps.Orders,
		plans:             deps.Plans,
		tasks:             deps.Tasks,
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
		Parameters:       cloneMap(req.Parameters),
		RequestContext:   cloneMap(req.RequestContext),
		Source:           "api",
		Status:           action.RequestStatusAccepted,
		CreatedAt:        now,
	}
	if err := s.requests.SaveRequest(ctx, request); err != nil {
		return ActionSubmissionResult{}, err
	}

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

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType:   appaudit.EventRequestAccepted,
		RequestID:   request.RequestID,
		PrincipalID: request.PrincipalID,
		TraceID:     request.TraceID,
		Metadata: map[string]any{
			"normalized_action":    req.ActionHint,
			"resolved_asset_ids":   append([]string(nil), resolvedAssets.AssetIDs...),
			"order_status":         string(order.StatusDraft),
			"approval_status":      string(order.ApprovalStatusNotRequired),
			"service_instance":     req.ResourceSelector.ServiceInstance,
			"database_name":        stringValue(req.Parameters["database_name"]),
			"resource_environment": req.ResourceSelector.Environment,
		},
	})

	authzDecision, err := s.authorization.Evaluate(ctx, dauth.Input{
		ActionName: req.ActionHint,
		Principal:  resolvedPrincipal,
		Assets:     resolvedAssets,
	})
	if err != nil {
		return ActionSubmissionResult{}, err
	}

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

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventAuthorizationDecided,
		RequestID: request.RequestID,
		OrderID:   orderID,
		TraceID:   traceID,
		Metadata: map[string]any{
			"final_decision":         string(authzDecision.FinalDecision),
			"authorization_decision": string(authzDecision.FinalDecision),
			"policy_decision":        string(authzDecision.PolicyDecision),
			"risk_level":             string(authzDecision.RiskLevel),
			"order_status":           string(orderStatus),
			"approval_status":        string(approvalStatus),
		},
	})

	builtPlan, err := s.planner.Build(ctx, assistantOrder)
	if err != nil {
		return ActionSubmissionResult{}, common.NewError(common.CodePlanBuildFailed, "failed to build execution plan", nil)
	}
	builtPlan.OrderID = assistantOrder.OrderID
	builtPlan.PlanStatus = plan.StatusFrozen
	builtPlan.SnapshotFrozen = true

	assistantOrder.PlanID = builtPlan.PlanID
	assistantOrder.PlanVersion = builtPlan.PlanVersion

	if err := s.orders.SaveOrder(ctx, assistantOrder); err != nil {
		return ActionSubmissionResult{}, err
	}
	if err := s.plans.SavePlan(ctx, builtPlan); err != nil {
		return ActionSubmissionResult{}, err
	}

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventOrderCreated,
		RequestID: request.RequestID,
		OrderID:   assistantOrder.OrderID,
		TraceID:   traceID,
		Metadata: map[string]any{
			"order_status":    string(assistantOrder.Status),
			"approval_status": string(assistantOrder.ApprovalStatus),
		},
	})
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventPlanFrozen,
		RequestID: request.RequestID,
		OrderID:   assistantOrder.OrderID,
		TraceID:   traceID,
		Metadata: map[string]any{
			"order_status":    string(assistantOrder.Status),
			"approval_status": string(assistantOrder.ApprovalStatus),
			"plan_status":     string(builtPlan.PlanStatus),
		},
	})

	if assistantOrder.ApprovalRequired && s.approval != nil {
		if _, err := s.approval.Create(ctx, assistantOrder, appapproval.CreateInput{
			ApprovalPolicyRef: authzDecision.ApprovalPolicyRef,
		}); err != nil {
			return ActionSubmissionResult{}, err
		}
	}

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

func (s *service) GetOrder(ctx context.Context, orderID string) (AssistantOrderView, error) {
	return s.orders.GetOrder(ctx, orderID)
}

func (s *service) GetTask(ctx context.Context, taskID string) (ExecutionTaskView, error) {
	return s.tasks.GetTask(ctx, taskID)
}

func (s *service) ExecuteApprovedOrder(ctx context.Context, authCtx appauth.AuthContext, input ExecuteOrderInput) (ExecuteOrderResult, error) {
	current, err := s.orders.GetOrder(ctx, input.OrderID)
	if err != nil {
		return ExecuteOrderResult{}, err
	}

	switch current.Status {
	case order.StatusWaitingApproval:
		return ExecuteOrderResult{}, common.NewError(common.CodeApprovalRequired, "order still requires approval", map[string]any{"order_id": current.OrderID})
	case order.StatusPlanStale, order.StatusRejected, order.StatusPolicyRejected, order.StatusExpired, order.StatusCancelled, order.StatusFailed:
		return ExecuteOrderResult{}, common.NewError(common.CodeOrderNotExecutable, "order current status does not allow execution", map[string]any{"order_id": current.OrderID, "status": current.Status})
	case order.StatusSucceeded:
		return ExecuteOrderResult{}, common.NewError(common.CodeOrderAlreadyExecuted, "order already executed", map[string]any{"order_id": current.OrderID})
	case order.StatusExecuting:
		existingTask, taskErr := s.tasks.GetTaskByOrderID(ctx, current.OrderID)
		if taskErr != nil {
			return ExecuteOrderResult{}, taskErr
		}
		return ExecuteOrderResult{
			OrderID:    current.OrderID,
			Status:     current.Status,
			TaskID:     existingTask.TaskID,
			ExecutorID: authCtx.AuthenticatedPrincipalID,
			TraceID:    s.traceID(ctx, current.RequestID),
		}, nil
	}

	executor, err := s.principalResolver.Resolve(ctx, authCtx.AuthenticatedPrincipalID, authCtx)
	if err != nil {
		return ExecuteOrderResult{}, err
	}
	if s.executeAuth == nil {
		return ExecuteOrderResult{}, common.NewError(common.CodeSystemInternalError, "execute authorization service is not configured", map[string]any{"order_id": current.OrderID})
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

	request, err := s.requests.GetRequest(ctx, current.RequestID)
	if err != nil {
		return ExecuteOrderResult{}, err
	}
	existingPlan, err := s.plans.GetPlanByOrderID(ctx, current.OrderID)
	if err != nil {
		return ExecuteOrderResult{}, err
	}
	traceID := request.TraceID

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType:      appaudit.EventExecuteTriggered,
		RequestID:      current.RequestID,
		OrderID:        current.OrderID,
		ExecuteActorID: executor.PrincipalID,
		TraceID:        traceID,
		Metadata: map[string]any{
			"order_status":    string(current.Status),
			"approval_status": string(current.ApprovalStatus),
			"execute_reason":  input.Reason,
		},
	})

	candidatePlan := existingPlan
	if existingPlan.PlanVersion != current.PlanVersion {
		candidatePlan.StaleReason = "order plan version does not match frozen execution plan"
	}
	if staleFlag, ok := request.RequestContext["simulate_plan_stale"].(bool); ok && staleFlag && candidatePlan.StaleReason == "" {
		candidatePlan.StaleReason = "simulated by request_context.simulate_plan_stale"
	}
	revalidation, err := s.planner.Revalidate(ctx, current, candidatePlan)
	if err != nil {
		return ExecuteOrderResult{}, common.NewError(common.CodePlanRevalidationFailed, "plan revalidation failed", map[string]any{"order_id": current.OrderID})
	}
	if !revalidation.Valid || revalidation.Status == plan.StatusStale {
		candidatePlan.PlanStatus = plan.StatusStale
		candidatePlan.StaleReason = staleReason(revalidation.Reason, candidatePlan.StaleReason)
		if err := s.plans.SavePlan(ctx, candidatePlan); err != nil {
			return ExecuteOrderResult{}, err
		}

		current.Status = order.StatusPlanStale
		current.UpdatedAt = time.Now().UTC()
		if err := s.orders.SaveOrder(ctx, current); err != nil {
			return ExecuteOrderResult{}, err
		}

		_ = s.audit.AppendEvent(ctx, appaudit.Event{
			EventType: appaudit.EventPlanStale,
			RequestID: current.RequestID,
			OrderID:   current.OrderID,
			TraceID:   traceID,
			Success:   boolPtr(false),
			Metadata: map[string]any{
				"order_status":    string(current.Status),
				"approval_status": string(current.ApprovalStatus),
				"error_code":      string(common.CodePlanStale),
				"error_message":   candidatePlan.StaleReason,
			},
		})

		_, _ = s.evidence.Build(ctx, appevidence.BuildInput{
			OrderID:          current.OrderID,
			RequestSummary:   requestSummary(request),
			ExecutionSuccess: false,
			FailureDetail: map[string]any{
				"code":   string(common.CodePlanStale),
				"reason": candidatePlan.StaleReason,
			},
			BeforeStateSnapshot: map[string]any{
				"order_status": string(order.StatusApproved),
				"plan_status":  string(existingPlan.PlanStatus),
			},
			AfterStateSnapshot: map[string]any{
				"order_status": string(order.StatusPlanStale),
				"plan_status":  string(plan.StatusStale),
			},
			ResultSummary:      "计划失效，未启动执行任务",
			RollbackSuggestion: "重新提交新的 action request 并生成新的 execution plan",
		})
		_ = s.audit.AppendEvent(ctx, appaudit.Event{
			EventType: appaudit.EventEvidenceWritten,
			RequestID: current.RequestID,
			OrderID:   current.OrderID,
			TraceID:   traceID,
			Success:   boolPtr(false),
			Metadata: map[string]any{
				"order_status":      string(current.Status),
				"approval_status":   string(current.ApprovalStatus),
				"execution_summary": "plan stale evidence written",
				"error_code":        string(common.CodePlanStale),
				"error_message":     candidatePlan.StaleReason,
			},
		})

		return ExecuteOrderResult{
			OrderID:    current.OrderID,
			Status:     current.Status,
			ExecutorID: executor.PrincipalID,
			TraceID:    traceID,
		}, nil
	}

	candidatePlan.PlanStatus = revalidation.Status
	candidatePlan.ValidatedAt = time.Now().UTC()
	if err := s.plans.SavePlan(ctx, candidatePlan); err != nil {
		return ExecuteOrderResult{}, err
	}
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventPlanRevalidated,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TraceID:   traceID,
		Success:   boolPtr(true),
		Metadata: map[string]any{
			"order_status":    string(current.Status),
			"approval_status": string(current.ApprovalStatus),
			"plan_status":     string(candidatePlan.PlanStatus),
		},
	})

	binding, err := s.router.Route(ctx, candidatePlan)
	if err != nil {
		return ExecuteOrderResult{}, common.NewError(common.CodeAdapterNotAvailable, "execution router could not bind adapter", map[string]any{"order_id": current.OrderID})
	}

	executionTask, err := s.runtime.Start(ctx, current, candidatePlan)
	if err != nil {
		return ExecuteOrderResult{}, common.NewError(common.CodeExecutionFailed, "task runtime failed to start", map[string]any{"order_id": current.OrderID})
	}

	candidatePlan.PlanStatus = plan.StatusConsumed
	if err := s.plans.SavePlan(ctx, candidatePlan); err != nil {
		return ExecuteOrderResult{}, err
	}

	current.Status = order.StatusExecuting
	current.LastExecuteTriggeredBy = executor.PrincipalID
	current.UpdatedAt = time.Now().UTC()
	if err := s.orders.SaveOrder(ctx, current); err != nil {
		return ExecuteOrderResult{}, err
	}

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventExecutionStarted,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TaskID:    executionTask.TaskID,
		TraceID:   traceID,
		Success:   boolPtr(true),
		Metadata: map[string]any{
			"order_status":      string(current.Status),
			"approval_status":   string(current.ApprovalStatus),
			"selected_adapter":  binding.AdapterType,
			"execution_summary": "phase-2 task skeleton started; real DB execution is still disabled",
		},
	})

	_, _ = s.evidence.Build(ctx, appevidence.BuildInput{
		OrderID:          current.OrderID,
		TaskID:           executionTask.TaskID,
		ArtifactRefs:     []string{"artifact://phase-2/task-skeleton"},
		RequestSummary:   requestSummary(request),
		ExecutionSuccess: true,
		BeforeStateSnapshot: map[string]any{
			"order_status": string(order.StatusApproved),
			"plan_status":  string(plan.StatusRevalidated),
		},
		AfterStateSnapshot: map[string]any{
			"order_status": string(order.StatusExecuting),
			"task_status":  string(executionTask.Status),
			"adapter":      binding.AdapterType,
		},
		ResultSummary:      "Execution task skeleton started; real DB execution remains disabled in Phase 02",
		RollbackSuggestion: "wait for terminal task runtime in later phases; do not assume the database was created",
	})
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventEvidenceWritten,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TaskID:    executionTask.TaskID,
		TraceID:   traceID,
		Success:   boolPtr(true),
		Metadata: map[string]any{
			"order_status":      string(current.Status),
			"approval_status":   string(current.ApprovalStatus),
			"execution_summary": "phase-2 evidence written for task skeleton",
		},
	})

	return ExecuteOrderResult{
		OrderID:    current.OrderID,
		Status:     current.Status,
		TaskID:     executionTask.TaskID,
		ExecutorID: executor.PrincipalID,
		TraceID:    traceID,
	}, nil
}

func (s *service) traceID(ctx context.Context, requestID string) string {
	request, err := s.requests.GetRequest(ctx, requestID)
	if err != nil {
		return ""
	}
	return request.TraceID
}

func (s *service) nextIDs() (string, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequence++
	value := s.sequence
	return fmt.Sprintf("req_%04d", value), fmt.Sprintf("trace_%04d", value), fmt.Sprintf("ord_%04d", value)
}

func boolPtr(value bool) *bool {
	return &value
}

func staleReason(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "plan revalidation reported stale"
}

func requestSummary(request action.Request) string {
	databaseName := stringValue(request.Parameters["database_name"])
	if databaseName == "" {
		databaseName = "unknown_database"
	}
	return fmt.Sprintf("create mysql database %s on %s/%s/%s",
		databaseName,
		request.ResourceSelector.Project,
		request.ResourceSelector.Environment,
		request.ResourceSelector.ServiceInstance,
	)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
