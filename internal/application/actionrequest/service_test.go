package actionrequest

import (
	"context"
	"strings"
	"testing"
	"time"

	appapproval "dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
	"dba_ai_assistant/internal/persistence"
)

func TestServiceSubmitCreatesWaitingApprovalOrderAndApprovalState(t *testing.T) {
	ctx := context.Background()
	service, approvalService, auditService, _, _ := newPhaseTwoServices(t)

	result, err := service.Submit(ctx, ActionRequestDTO{
		PrincipalID: "u_requester",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if result.Status != order.StatusWaitingApproval {
		t.Fatalf("expected waiting approval order, got %s", result.Status)
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approval_required=true")
	}

	approvalState, err := approvalService.Get(ctx, result.OrderID)
	if err != nil {
		t.Fatalf("Get approval state returned error: %v", err)
	}
	if approvalState.ApprovalStatus != order.ApprovalStatusWaitingApproval {
		t.Fatalf("expected waiting approval state, got %s", approvalState.ApprovalStatus)
	}
	if approvalState.ExpiresAt.IsZero() {
		t.Fatalf("expected approval expiry to be set from policy TTL")
	}

	ledger, err := auditService.GetViewByRequestID(ctx, result.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventRequestAccepted)
	assertAuditEvent(t, ledger, appaudit.EventAuthorizationDecided)
	assertAuditEvent(t, ledger, appaudit.EventOrderCreated)
	assertAuditEvent(t, ledger, appaudit.EventPlanFrozen)
	assertAuditEvent(t, ledger, appaudit.EventApprovalCreated)
	if ledger.LatestOrderStatus != string(order.StatusWaitingApproval) {
		t.Fatalf("expected latest order status WAITING_APPROVAL, got %q", ledger.LatestOrderStatus)
	}
	if ledger.LatestApprovalStatus != string(order.ApprovalStatusWaitingApproval) {
		t.Fatalf("expected latest approval status WAITING_APPROVAL, got %q", ledger.LatestApprovalStatus)
	}
}

func TestServiceExecuteApprovedOrderStartsTaskAndWritesEvidence(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, _ := newPhaseTwoServices(t)

	submitResult, err := service.Submit(ctx, ActionRequestDTO{
		PrincipalID: "u_requester",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	executeResult, err := service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "manual execute",
	})
	if err != nil {
		t.Fatalf("ExecuteApprovedOrder returned error: %v", err)
	}

	if executeResult.Status != order.StatusExecuting {
		t.Fatalf("expected status EXECUTING, got %s", executeResult.Status)
	}
	if executeResult.TaskID == "" {
		t.Fatalf("expected task id to be created")
	}

	orderView, err := service.GetOrder(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if orderView.Status != order.StatusExecuting {
		t.Fatalf("expected persisted order status EXECUTING, got %s", orderView.Status)
	}
	if orderView.LastExecuteTriggeredBy != "u_executor" {
		t.Fatalf("expected last_execute_triggered_by to be captured, got %q", orderView.LastExecuteTriggeredBy)
	}

	taskView, err := service.GetTask(ctx, executeResult.TaskID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if taskView.Status != "RUNNING" {
		t.Fatalf("expected running task skeleton, got %s", taskView.Status)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, submitResult.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventExecuteTriggered)
	assertAuditEvent(t, ledger, appaudit.EventPlanRevalidated)
	assertAuditEvent(t, ledger, appaudit.EventExecutionStarted)
	assertAuditEvent(t, ledger, appaudit.EventEvidenceWritten)
	if ledger.LatestTaskID != executeResult.TaskID {
		t.Fatalf("expected latest task id %q, got %q", executeResult.TaskID, ledger.LatestTaskID)
	}
	if ledger.LatestOrderStatus != string(order.StatusExecuting) {
		t.Fatalf("expected latest order status EXECUTING, got %q", ledger.LatestOrderStatus)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if pack.TaskID != executeResult.TaskID {
		t.Fatalf("expected evidence task id %q, got %q", executeResult.TaskID, pack.TaskID)
	}
	if !pack.ExecutionSuccess {
		t.Fatalf("expected phase-2 execute evidence to mark control flow success")
	}
	if !strings.Contains(pack.ResultSummary, "task skeleton started") {
		t.Fatalf("expected phase-2 evidence summary to mention task skeleton, got %q", pack.ResultSummary)
	}
}

func TestServiceExecuteApprovedOrderMarksPlanStaleWithoutCreatingTask(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, _ := newPhaseTwoServices(t)

	submitResult, err := service.Submit(ctx, ActionRequestDTO{
		PrincipalID: "u_requester",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
		RequestContext: map[string]any{
			"simulate_plan_stale": true,
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	executeResult, err := service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "manual execute",
	})
	if err != nil {
		t.Fatalf("ExecuteApprovedOrder returned error: %v", err)
	}

	if executeResult.Status != order.StatusPlanStale {
		t.Fatalf("expected PLAN_STALE result, got %s", executeResult.Status)
	}
	if executeResult.TaskID != "" {
		t.Fatalf("expected no task to be created on stale plan, got %q", executeResult.TaskID)
	}

	orderView, err := service.GetOrder(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if orderView.Status != order.StatusPlanStale {
		t.Fatalf("expected persisted order status PLAN_STALE, got %s", orderView.Status)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, submitResult.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventExecuteTriggered)
	assertAuditEvent(t, ledger, appaudit.EventPlanStale)
	assertAuditEvent(t, ledger, appaudit.EventEvidenceWritten)
	if ledger.LatestTaskID != "" {
		t.Fatalf("expected latest task id to stay empty on stale path, got %q", ledger.LatestTaskID)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if pack.TaskID != "" {
		t.Fatalf("expected stale evidence to keep task_id empty, got %q", pack.TaskID)
	}
	if pack.ExecutionSuccess {
		t.Fatalf("expected stale evidence to be marked unsuccessful")
	}
	if pack.FailureDetail["reason"] == "" {
		t.Fatalf("expected stale evidence to include failure reason")
	}
}

func TestServiceExecuteApprovedOrderReturnsExistingTaskWhenAlreadyExecuting(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, _, _ := newPhaseTwoServices(t)

	submitResult, err := service.Submit(ctx, ActionRequestDTO{
		PrincipalID: "u_requester",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	first, err := service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "first execute",
	})
	if err != nil {
		t.Fatalf("first ExecuteApprovedOrder returned error: %v", err)
	}

	second, err := service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "second execute",
	})
	if err != nil {
		t.Fatalf("second ExecuteApprovedOrder returned error: %v", err)
	}

	if second.TaskID != first.TaskID {
		t.Fatalf("expected second execute to return existing task %q, got %q", first.TaskID, second.TaskID)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, submitResult.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	if countAuditEvents(ledger, appaudit.EventExecutionStarted) != 1 {
		t.Fatalf("expected EXECUTION_STARTED to be written once, got %d", countAuditEvents(ledger, appaudit.EventExecutionStarted))
	}
}

func TestServiceExecuteApprovedOrderRejectsWaitingApprovalOrder(t *testing.T) {
	ctx := context.Background()
	service, _, _, _, _ := newPhaseTwoServices(t)

	submitResult, err := service.Submit(ctx, ActionRequestDTO{
		PrincipalID: "u_requester",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	_, err = service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "too early",
	})
	if err == nil {
		t.Fatalf("expected approval gate to block execute")
	}
	if code := common.ErrorCode(err); code != common.CodeApprovalRequired {
		t.Fatalf("expected %s, got %s", common.CodeApprovalRequired, code)
	}
}

func TestServiceExecuteApprovedOrderRejectsCallerWithoutExecutePolicy(t *testing.T) {
	ctx := context.Background()
	service, _, _, _, _ := newPhaseTwoServices(t)

	submitResult, err := service.Submit(ctx, ActionRequestDTO{
		PrincipalID: "u_requester",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	_, err = service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_assistant",
		Roles:                    []string{principal.RoleAssistantUser},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "unauthorized execute attempt",
	})
	if err == nil {
		t.Fatalf("expected execute policy to deny caller")
	}
	if code := common.ErrorCode(err); code != common.CodeExecutorNotAllowed {
		t.Fatalf("expected %s, got %s", common.CodeExecutorNotAllowed, code)
	}
}

func newPhaseTwoServices(t *testing.T) (Service, appapproval.Service, *appaudit.MemoryService, *appevidence.MemoryService, *persistence.MemoryStore) {
	t.Helper()

	ctx := context.Background()
	store := persistence.NewMemoryStore()
	if err := store.SaveApprovalPolicy(ctx, policy.ApprovalPolicy{
		PolicyID:           "approval_policy_prod_r2",
		ActionName:         "mysql.database.create",
		RiskLevels:         []string{string(risk.LevelR2)},
		ApproverRoles:      []string{principal.RoleProdApprover, principal.RolePlatformAdmin},
		MinApproverCount:   1,
		ForbidSelfApproval: true,
		TTL:                30 * time.Minute,
		Enabled:            true,
	}); err != nil {
		t.Fatalf("failed to seed approval policy: %v", err)
	}

	auditService := appaudit.NewMemoryService(store)
	evidenceService := appevidence.NewMemoryService(store)
	approvalService := appapproval.NewService(appapproval.Dependencies{
		Orders:    store,
		Plans:     store,
		Approvals: store,
		Policies:  store,
		Audit:     auditService,
	})
	actionService := NewService(Dependencies{
		PrincipalResolver: appauth.NewStaticPrincipalResolver(),
		AssetResolver:     appauth.NewInMemoryExactAssetResolver(appauth.StaticManagedAssets()),
		Authorization: appauth.NewAuthorizationService(
			appauth.NewStaticPolicyEngine(),
			appauth.NewStaticRiskEngine(),
		),
		ExecuteAuth: appauth.NewStaticExecuteAuthorizationService(),
		Planner:     appexec.NewStaticExecutionPlanner(),
		Router:      appexec.NewStaticExecutionRouter(),
		Runtime:     appexec.NewNoopTaskRuntime(store),
		Approval:    approvalService,
		Audit:       auditService,
		Evidence:    evidenceService,
		Requests:    store,
		Orders:      store,
		Plans:       store,
		Tasks:       store,
	})
	return actionService, approvalService, auditService, evidenceService, store
}

func assertAuditEvent(t *testing.T, ledger appaudit.LedgerView, eventType appaudit.EventType) {
	t.Helper()
	for _, event := range ledger.Events {
		if event.EventType == eventType {
			return
		}
	}
	t.Fatalf("expected audit event %s to be present", eventType)
}

func countAuditEvents(ledger appaudit.LedgerView, eventType appaudit.EventType) int {
	count := 0
	for _, event := range ledger.Events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}
