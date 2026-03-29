package actionrequest

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	appapproval "dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/action"
	"dba_ai_assistant/internal/domain/asset"
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
	router            appexec.ExecutionRouter
	runtime           appexec.TaskRuntime
	approval          appapproval.Service
	audit             appaudit.Service
	evidence          appevidence.Service
	requests          requestRepository
	orders            orderRepository
	plans             planRepository
	tasks             taskRepository
	idempotency       appexec.IdempotencyRepository

	mu       sync.Mutex
	sequence int
}

var databaseNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

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
		idempotency:       deps.Idempotency,
	}
}

func (s *service) Submit(ctx context.Context, req ActionRequestDTO) (ActionSubmissionResult, error) {
	if req.PrincipalID == "" {
		return ActionSubmissionResult{}, common.NewError(common.CodeRequestInvalid, "principal_id is required", nil)
	}
	if req.ActionHint == "" {
		req.ActionHint = string(action.NameMySQLDatabaseCreate)
	}
	if err := validateSubmitRequest(req); err != nil {
		return ActionSubmissionResult{}, err
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
	approvalRefs := s.approvalRefs(ctx, current.OrderID)

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
	revalidation, err := s.planner.Revalidate(ctx, current, candidatePlan)
	if err != nil {
		return ExecuteOrderResult{}, common.NewError(common.CodePlanRevalidationFailed, "plan revalidation failed", map[string]any{"order_id": current.OrderID})
	}
	if !revalidation.Valid || revalidation.Status == plan.StatusStale {
		return s.markPlanStale(ctx, request, current, existingPlan, candidatePlan, traceID, approvalRefs, staleReason(revalidation.Reason, candidatePlan.StaleReason), nil, executor.PrincipalID)
	}

	resolvedAssets, err := s.assetResolver.ResolveExact(ctx, current.ActionName, request.ResourceSelector)
	if err != nil {
		return s.markPlanStale(ctx, request, current, existingPlan, candidatePlan, traceID, approvalRefs, err.Error(), nil, executor.PrincipalID)
	}
	if len(resolvedAssets.AssetIDs) != len(current.ResolvedAssetIDs) || len(resolvedAssets.Assets) != 1 || !sameStrings(resolvedAssets.AssetIDs, current.ResolvedAssetIDs) {
		return s.markPlanStale(ctx, request, current, existingPlan, candidatePlan, traceID, approvalRefs, "resolved asset set no longer matches frozen order target", nil, executor.PrincipalID)
	}
	resolvedAsset := resolvedAssets.Assets[0]

	binding, err := s.router.Route(ctx, candidatePlan)
	if err != nil {
		return ExecuteOrderResult{}, common.NewError(common.CodeAdapterNotAvailable, "execution router could not bind adapter", map[string]any{"order_id": current.OrderID})
	}

	adapterRequest := s.buildAdapterRequest(traceID, current, resolvedAsset, request, "")
	dryRun, err := binding.Adapter.DryRun(ctx, adapterRequest)
	if err != nil {
		return s.blockedExecutionFailure(ctx, request, current, traceID, approvalRefs, common.CodePlanRevalidationFailed, "adapter dry run failed", nil, executor.PrincipalID)
	}
	if !dryRun.Supported {
		return ExecuteOrderResult{}, common.NewError(common.CodeAdapterNotAvailable, "execution router returned unsupported adapter", map[string]any{"order_id": current.OrderID})
	}

	beforeSnapshot := beforeSnapshotForExecution(current, existingPlan, resolvedAsset, adapterRequest.IdempotencyKey, dryRun)
	idempotencyRecord, hasIdempotencyRecord, err := s.getIdempotencyRecord(ctx, adapterRequest.IdempotencyKey)
	if err != nil {
		return ExecuteOrderResult{}, err
	}
	if hasIdempotencyRecord && idempotencyRecord.TaskStatus == task.StatusRunning {
		return s.blockedExecutionFailure(ctx, request, current, traceID, approvalRefs, common.CodeIdempotencyConflict, "another execution with the same idempotency key is already running", beforeSnapshot, executor.PrincipalID)
	}
	if previewBool(dryRun.RenderedPreview, "database_exists") {
		if hasIdempotencyRecord && idempotencyRecord.TaskStatus == task.StatusSucceeded {
			return s.replayIdempotentSuccess(ctx, request, current, candidatePlan, traceID, approvalRefs, beforeSnapshot, idempotencyRecord, executor.PrincipalID)
		}
		return s.markPlanStale(ctx, request, current, existingPlan, candidatePlan, traceID, approvalRefs, "database already exists before execute", beforeSnapshot, executor.PrincipalID)
	}
	if !dryRun.Ready {
		if containsAny(dryRun.Issues, "CONNECTION_REF_MISSING", "CONNECTION_REF_UNRESOLVED", "TARGET_ENGINE_UNSUPPORTED") {
			return s.markPlanStale(ctx, request, current, existingPlan, candidatePlan, traceID, approvalRefs, strings.Join(dryRun.Issues, ", "), beforeSnapshot, executor.PrincipalID)
		}
		return s.blockedExecutionFailure(ctx, request, current, traceID, approvalRefs, common.CodePlanRevalidationFailed, strings.Join(dryRun.Issues, ", "), beforeSnapshot, executor.PrincipalID)
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

	taskID := "task_" + current.OrderID
	adapterRequest.TaskID = taskID
	if err := s.saveIdempotencyRecord(ctx, appexec.IdempotencyRecord{
		IdempotencyKey: adapterRequest.IdempotencyKey,
		OrderID:        current.OrderID,
		TaskID:         taskID,
		TaskStatus:     task.StatusRunning,
		Summary:        "execution reserved",
		UpdatedAt:      time.Now().UTC(),
	}); err != nil {
		return ExecuteOrderResult{}, err
	}

	startResult, err := s.runtime.Start(ctx, appexec.TaskStartInput{
		Order:          current,
		Plan:           candidatePlan,
		Binding:        binding,
		AdapterRequest: adapterRequest,
	})
	if err != nil {
		_ = s.saveIdempotencyRecord(ctx, appexec.IdempotencyRecord{
			IdempotencyKey: adapterRequest.IdempotencyKey,
			OrderID:        current.OrderID,
			TaskID:         taskID,
			TaskStatus:     task.StatusFailed,
			Summary:        "task runtime failed to start",
			Error: map[string]any{
				"code":    string(common.CodeExecutionFailed),
				"message": err.Error(),
			},
			UpdatedAt: time.Now().UTC(),
		})
		return s.recordTerminalFailure(ctx, request, current, candidatePlan, traceID, approvalRefs, taskID, executor.PrincipalID, beforeSnapshot, map[string]any{
			"code":    string(common.CodeExecutionFailed),
			"message": err.Error(),
		}, "task runtime failed to start", nil)
	}

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventExecutionStarted,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TaskID:    startResult.Task.TaskID,
		TraceID:   traceID,
		Success:   boolPtr(true),
		Metadata: map[string]any{
			"order_status":     string(order.StatusExecuting),
			"approval_status":  string(current.ApprovalStatus),
			"selected_adapter": binding.AdapterType,
		},
	})

	candidatePlan.PlanStatus = plan.StatusConsumed
	if err := s.plans.SavePlan(ctx, candidatePlan); err != nil {
		return ExecuteOrderResult{}, err
	}

	afterSnapshot := afterSnapshotForExecution(startResult, binding.AdapterType)
	if startResult.Task.Status == task.StatusSucceeded && startResult.AdapterResult.Success {
		current.Status = order.StatusSucceeded
		current.LastExecuteTriggeredBy = executor.PrincipalID
		current.UpdatedAt = time.Now().UTC()
		if err := s.orders.SaveOrder(ctx, current); err != nil {
			return ExecuteOrderResult{}, err
		}
		_ = s.saveIdempotencyRecord(ctx, appexec.IdempotencyRecord{
			IdempotencyKey: adapterRequest.IdempotencyKey,
			OrderID:        current.OrderID,
			TaskID:         startResult.Task.TaskID,
			TaskStatus:     startResult.Task.Status,
			Summary:        startResult.AdapterResult.Summary,
			Outputs:        cloneMap(startResult.AdapterResult.Outputs),
			Artifacts:      cloneArtifacts(startResult.AdapterResult.Artifacts),
			UpdatedAt:      time.Now().UTC(),
		})
		_ = s.audit.AppendEvent(ctx, appaudit.Event{
			EventType: appaudit.EventExecutionSucceeded,
			RequestID: current.RequestID,
			OrderID:   current.OrderID,
			TaskID:    startResult.Task.TaskID,
			TraceID:   traceID,
			Success:   boolPtr(true),
			Metadata: map[string]any{
				"order_status":      string(current.Status),
				"approval_status":   string(current.ApprovalStatus),
				"execution_summary": startResult.AdapterResult.Summary,
			},
		})
		s.writeEvidence(ctx, request, current, traceID, approvalRefs, startResult.Task.TaskID, true, beforeSnapshot, afterSnapshot, artifactRefs(startResult.AdapterResult.Artifacts), nil, startResult.AdapterResult.Summary)
		return ExecuteOrderResult{
			OrderID:    current.OrderID,
			Status:     current.Status,
			TaskID:     startResult.Task.TaskID,
			ExecutorID: executor.PrincipalID,
			TraceID:    traceID,
		}, nil
	}

	return s.recordTerminalFailure(ctx, request, current, candidatePlan, traceID, approvalRefs, startResult.Task.TaskID, executor.PrincipalID, beforeSnapshot, startResult.AdapterResult.Error, startResult.AdapterResult.Summary, startResult.AdapterResult.Artifacts)
}

func validateSubmitRequest(req ActionRequestDTO) error {
	if req.ActionHint != string(action.NameMySQLDatabaseCreate) {
		return common.NewError(common.CodeRequestInvalid, "only mysql.database.create is supported in the current MVP", map[string]any{
			"action_name": req.ActionHint,
		})
	}
	databaseName := stringValue(req.Parameters["database_name"])
	if !databaseNamePattern.MatchString(databaseName) {
		return common.NewError(common.CodeRequestInvalid, "database_name violates platform naming convention", map[string]any{
			"database_name": databaseName,
		})
	}
	if strings.TrimSpace(req.ResourceSelector.Project) == "" || strings.TrimSpace(req.ResourceSelector.Environment) == "" || strings.TrimSpace(req.ResourceSelector.ServiceInstance) == "" {
		return common.NewError(common.CodeRequestInvalid, "resource_selector.project/environment/service_instance are required", nil)
	}
	return nil
}

func (s *service) buildAdapterRequest(traceID string, ord order.AssistantOrder, resolvedAsset asset.Asset, request action.Request, taskID string) appexec.AdapterExecutionRequest {
	databaseName := stringValue(request.Parameters["database_name"])
	idempotencyKey := appexec.BuildIdempotencyKey(ord.ActionName, resolvedAsset.AssetID, databaseName)
	return appexec.AdapterExecutionRequest{
		TraceID:        traceID,
		OrderID:        ord.OrderID,
		TaskID:         taskID,
		StepID:         "mysql.database.create",
		ActionName:     ord.ActionName,
		IdempotencyKey: idempotencyKey,
		Target: map[string]any{
			"asset_id":       resolvedAsset.AssetID,
			"asset_type":     string(resolvedAsset.AssetType),
			"connection_ref": resolvedAsset.ConnectionRef,
			"canonical_name": resolvedAsset.CanonicalName,
			"engine":         "mysql",
			"environment":    resolvedAsset.Environment,
		},
		Parameters: map[string]any{
			"database_name": databaseName,
		},
		ExecutionControls: map[string]any{
			"timeout_seconds": 30,
			"retry_policy":    "none",
			"idempotency_key": idempotencyKey,
		},
		EvidenceRequirements: map[string]any{
			"capture_before_state": true,
			"capture_after_state":  true,
			"capture_sql_summary":  true,
		},
	}
}

func (s *service) approvalRefs(ctx context.Context, orderID string) []string {
	if s.approval == nil {
		return nil
	}
	state, err := s.approval.Get(ctx, orderID)
	if err != nil {
		return nil
	}
	refs := make([]string, 0, len(state.Records))
	for _, record := range state.Records {
		if record.ApprovalID != "" {
			refs = append(refs, record.ApprovalID)
		}
	}
	return refs
}

func (s *service) getIdempotencyRecord(ctx context.Context, key string) (appexec.IdempotencyRecord, bool, error) {
	if s.idempotency == nil {
		return appexec.IdempotencyRecord{}, false, nil
	}
	return s.idempotency.GetIdempotencyRecord(ctx, key)
}

func (s *service) saveIdempotencyRecord(ctx context.Context, record appexec.IdempotencyRecord) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.SaveIdempotencyRecord(ctx, record)
}

func (s *service) markPlanStale(ctx context.Context, request action.Request, current order.AssistantOrder, existingPlan plan.ExecutionPlan, candidatePlan plan.ExecutionPlan, traceID string, approvalRefs []string, reason string, beforeSnapshot map[string]any, executorID string) (ExecuteOrderResult, error) {
	candidatePlan.PlanStatus = plan.StatusStale
	candidatePlan.StaleReason = staleReason(reason, candidatePlan.StaleReason)
	if err := s.plans.SavePlan(ctx, candidatePlan); err != nil {
		return ExecuteOrderResult{}, err
	}

	current.Status = order.StatusPlanStale
	current.LastExecuteTriggeredBy = executorID
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

	if beforeSnapshot == nil {
		beforeSnapshot = map[string]any{
			"order_status": string(order.StatusApproved),
			"plan_status":  string(existingPlan.PlanStatus),
		}
	}
	s.writeEvidence(ctx, request, current, traceID, approvalRefs, "", false, beforeSnapshot, map[string]any{
		"order_status":    string(order.StatusPlanStale),
		"plan_status":     string(plan.StatusStale),
		"database_exists": beforeSnapshot["database_exists"],
	}, nil, map[string]any{
		"code":   string(common.CodePlanStale),
		"reason": candidatePlan.StaleReason,
	}, "计划失效，未启动执行任务")

	return ExecuteOrderResult{
		OrderID:    current.OrderID,
		Status:     current.Status,
		ExecutorID: executorID,
		TraceID:    traceID,
	}, nil
}

func (s *service) blockedExecutionFailure(ctx context.Context, request action.Request, current order.AssistantOrder, traceID string, approvalRefs []string, code common.Code, message string, beforeSnapshot map[string]any, executorID string) (ExecuteOrderResult, error) {
	s.writeEvidence(ctx, request, current, traceID, approvalRefs, "", false, beforeSnapshot, map[string]any{
		"order_status": string(current.Status),
	}, nil, map[string]any{
		"code":    string(code),
		"message": message,
	}, message)
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventExecutionFailed,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TraceID:   traceID,
		Success:   boolPtr(false),
		Metadata: map[string]any{
			"order_status":      string(current.Status),
			"approval_status":   string(current.ApprovalStatus),
			"execution_summary": message,
			"error_code":        string(code),
			"error_message":     message,
		},
	})
	return ExecuteOrderResult{}, common.NewError(code, message, map[string]any{
		"order_id": current.OrderID,
	})
}

func (s *service) replayIdempotentSuccess(ctx context.Context, request action.Request, current order.AssistantOrder, candidatePlan plan.ExecutionPlan, traceID string, approvalRefs []string, beforeSnapshot map[string]any, record appexec.IdempotencyRecord, executorID string) (ExecuteOrderResult, error) {
	candidatePlan.PlanStatus = plan.StatusConsumed
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
			"plan_status":     string(plan.StatusRevalidated),
		},
	})

	taskID := "task_" + current.OrderID
	replayTask := task.ExecutionTask{
		TaskID:      taskID,
		OrderID:     current.OrderID,
		TraceID:     current.TraceID,
		ActionName:  current.ActionName,
		Status:      task.StatusSucceeded,
		StartedAt:   time.Now().UTC(),
		EndedAt:     time.Now().UTC(),
		HeartbeatAt: time.Now().UTC(),
	}
	if err := s.tasks.SaveTask(ctx, replayTask); err != nil {
		return ExecuteOrderResult{}, err
	}

	current.Status = order.StatusSucceeded
	current.LastExecuteTriggeredBy = executorID
	current.UpdatedAt = time.Now().UTC()
	if err := s.orders.SaveOrder(ctx, current); err != nil {
		return ExecuteOrderResult{}, err
	}
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventExecutionSucceeded,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TaskID:    replayTask.TaskID,
		TraceID:   traceID,
		Success:   boolPtr(true),
		Metadata: map[string]any{
			"order_status":      string(current.Status),
			"approval_status":   string(current.ApprovalStatus),
			"execution_summary": "idempotent replay reused previous success",
		},
	})
	s.writeEvidence(ctx, request, current, traceID, approvalRefs, replayTask.TaskID, true, beforeSnapshot, map[string]any{
		"order_status":      string(current.Status),
		"task_status":       string(replayTask.Status),
		"database_exists":   true,
		"idempotent_replay": true,
	}, artifactRefs(record.Artifacts), nil, "idempotent replay reused previous success")
	return ExecuteOrderResult{
		OrderID:    current.OrderID,
		Status:     current.Status,
		TaskID:     replayTask.TaskID,
		ExecutorID: executorID,
		TraceID:    traceID,
	}, nil
}

func (s *service) recordTerminalFailure(ctx context.Context, request action.Request, current order.AssistantOrder, candidatePlan plan.ExecutionPlan, traceID string, approvalRefs []string, taskID string, executorID string, beforeSnapshot map[string]any, failureDetail map[string]any, summary string, artifacts []map[string]any) (ExecuteOrderResult, error) {
	current.Status = order.StatusFailed
	current.LastExecuteTriggeredBy = executorID
	current.UpdatedAt = time.Now().UTC()
	if err := s.orders.SaveOrder(ctx, current); err != nil {
		return ExecuteOrderResult{}, err
	}
	_ = s.saveIdempotencyRecord(ctx, appexec.IdempotencyRecord{
		IdempotencyKey: appexec.BuildIdempotencyKey(current.ActionName, current.ResolvedAssetIDs[0], stringValue(request.Parameters["database_name"])),
		OrderID:        current.OrderID,
		TaskID:         taskID,
		TaskStatus:     task.StatusFailed,
		Summary:        summary,
		Artifacts:      cloneArtifacts(artifacts),
		Error:          cloneMap(failureDetail),
		UpdatedAt:      time.Now().UTC(),
	})
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventExecutionFailed,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TaskID:    taskID,
		TraceID:   traceID,
		Success:   boolPtr(false),
		Metadata: map[string]any{
			"order_status":      string(current.Status),
			"approval_status":   string(current.ApprovalStatus),
			"execution_summary": summary,
			"error_code":        stringValue(failureDetail["code"]),
			"error_message":     stringValue(failureDetail["message"]),
		},
	})
	s.writeEvidence(ctx, request, current, traceID, approvalRefs, taskID, false, beforeSnapshot, map[string]any{
		"order_status": string(current.Status),
		"task_status":  string(task.StatusFailed),
	}, artifactRefs(artifacts), failureDetail, summary)
	return ExecuteOrderResult{}, common.NewError(common.CodeExecutionFailed, summary, map[string]any{
		"order_id": current.OrderID,
		"task_id":  taskID,
	})
}

func (s *service) writeEvidence(ctx context.Context, request action.Request, current order.AssistantOrder, traceID string, approvalRefs []string, taskID string, success bool, beforeSnapshot map[string]any, afterSnapshot map[string]any, artifacts []string, failureDetail map[string]any, summary string) {
	_, _ = s.evidence.Build(ctx, appevidence.BuildInput{
		OrderID:             current.OrderID,
		TaskID:              taskID,
		TraceID:             traceID,
		ArtifactRefs:        artifacts,
		RequestSummary:      requestSummary(request),
		BeforeStateSnapshot: beforeSnapshot,
		AfterStateSnapshot:  afterSnapshot,
		ApprovalRefs:        approvalRefs,
		ExecutionSuccess:    success,
		FailureDetail:       cloneMap(failureDetail),
		ResultSummary:       summary,
		RollbackSuggestion:  rollbackSuggestion(success),
	})
	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventEvidenceWritten,
		RequestID: current.RequestID,
		OrderID:   current.OrderID,
		TaskID:    taskID,
		TraceID:   traceID,
		Success:   boolPtr(success),
		Metadata: map[string]any{
			"order_status":      string(current.Status),
			"approval_status":   string(current.ApprovalStatus),
			"execution_summary": summary,
			"error_code":        stringValue(failureDetail["code"]),
			"error_message":     stringValue(failureDetail["message"]),
		},
	})
}

func beforeSnapshotForExecution(current order.AssistantOrder, existingPlan plan.ExecutionPlan, resolvedAsset asset.Asset, idempotencyKey string, dryRun appexec.AdapterDryRunResult) map[string]any {
	return map[string]any{
		"order_status":    string(current.Status),
		"plan_status":     string(existingPlan.PlanStatus),
		"asset_id":        resolvedAsset.AssetID,
		"connection_ref":  resolvedAsset.ConnectionRef,
		"database_exists": previewBool(dryRun.RenderedPreview, "database_exists"),
		"database_name":   dryRun.RenderedPreview["database_name"],
		"idempotency_key": idempotencyKey,
		"sql_summary":     dryRun.RenderedPreview["sql_summary"],
	}
}

func afterSnapshotForExecution(startResult appexec.TaskStartResult, adapterType string) map[string]any {
	return map[string]any{
		"task_status":     string(startResult.Task.Status),
		"database_exists": boolValue(startResult.AdapterResult.Outputs["database_exists_after"]),
		"database_name":   startResult.AdapterResult.Outputs["database_name"],
		"adapter":         adapterType,
		"sql_summary":     startResult.AdapterResult.Outputs["sql_summary"],
	}
}

func artifactRefs(artifacts []map[string]any) []string {
	refs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		if ref := stringValue(artifact["ref"]); ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func cloneArtifacts(artifacts []map[string]any) []map[string]any {
	if len(artifacts) == 0 {
		return nil
	}
	cloned := make([]map[string]any, 0, len(artifacts))
	for _, artifact := range artifacts {
		cloned = append(cloned, cloneMap(artifact))
	}
	return cloned
}

func rollbackSuggestion(success bool) string {
	if success {
		return "drop the database manually if the created schema must be rolled back"
	}
	return "investigate the failure and retry only after verifying target state"
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func containsAny(values []string, targets ...string) bool {
	for _, value := range values {
		for _, target := range targets {
			if value == target {
				return true
			}
		}
	}
	return false
}

func previewBool(preview map[string]any, key string) bool {
	return boolValue(preview[key])
}

func boolValue(value any) bool {
	boolean, _ := value.(bool)
	return boolean
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
