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
	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/task"
	"dba_ai_assistant/internal/persistence"
)

func TestServiceSubmitCreatesWaitingApprovalOrderAndApprovalState(t *testing.T) {
	ctx := context.Background()
	service, approvalService, auditService, _, _, _ := newPhaseTwoServices(t)

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
	if approvalState.TraceID != result.TraceID {
		t.Fatalf("expected approval state trace id %q, got %q", result.TraceID, approvalState.TraceID)
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
	if ledger.TraceID != result.TraceID {
		t.Fatalf("expected ledger trace id %q, got %q", result.TraceID, ledger.TraceID)
	}
	if ledger.LatestOrderStatus != string(order.StatusWaitingApproval) {
		t.Fatalf("expected latest order status WAITING_APPROVAL, got %q", ledger.LatestOrderStatus)
	}
	if ledger.LatestApprovalStatus != string(order.ApprovalStatusWaitingApproval) {
		t.Fatalf("expected latest approval status WAITING_APPROVAL, got %q", ledger.LatestApprovalStatus)
	}
}

func TestServiceGetOrderReturnsTraceIDFromSubmission(t *testing.T) {
	ctx := context.Background()
	service, _, _, _, _, _ := newPhaseTwoServices(t)

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

	view, err := service.GetOrder(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if view.TraceID != submitResult.TraceID {
		t.Fatalf("expected trace id %q, got %q", submitResult.TraceID, view.TraceID)
	}
}

func TestServiceExecuteApprovedOrderStartsTaskAndWritesEvidence(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, _, _ := newPhaseTwoServices(t)

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

	if executeResult.Status != order.StatusSucceeded {
		t.Fatalf("expected status SUCCEEDED, got %s", executeResult.Status)
	}
	if executeResult.TaskID == "" {
		t.Fatalf("expected task id to be created")
	}

	orderView, err := service.GetOrder(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if orderView.Status != order.StatusSucceeded {
		t.Fatalf("expected persisted order status SUCCEEDED, got %s", orderView.Status)
	}
	if orderView.LastExecuteTriggeredBy != "u_executor" {
		t.Fatalf("expected last_execute_triggered_by to be captured, got %q", orderView.LastExecuteTriggeredBy)
	}

	taskView, err := service.GetTask(ctx, executeResult.TaskID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if taskView.TraceID != submitResult.TraceID {
		t.Fatalf("expected task trace id %q, got %q", submitResult.TraceID, taskView.TraceID)
	}
	if taskView.Status != task.StatusSucceeded {
		t.Fatalf("expected succeeded task, got %s", taskView.Status)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, submitResult.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventExecuteTriggered)
	assertAuditEvent(t, ledger, appaudit.EventPlanRevalidated)
	assertAuditEvent(t, ledger, appaudit.EventExecutionStarted)
	assertAuditEvent(t, ledger, appaudit.EventExecutionSucceeded)
	assertAuditEvent(t, ledger, appaudit.EventEvidenceWritten)
	if ledger.TraceID != submitResult.TraceID {
		t.Fatalf("expected ledger trace id %q, got %q", submitResult.TraceID, ledger.TraceID)
	}
	if ledger.LatestTaskID != executeResult.TaskID {
		t.Fatalf("expected latest task id %q, got %q", executeResult.TaskID, ledger.LatestTaskID)
	}
	if ledger.LatestOrderStatus != string(order.StatusSucceeded) {
		t.Fatalf("expected latest order status SUCCEEDED, got %q", ledger.LatestOrderStatus)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if pack.TraceID != submitResult.TraceID {
		t.Fatalf("expected evidence trace id %q, got %q", submitResult.TraceID, pack.TraceID)
	}
	if pack.TaskID != executeResult.TaskID {
		t.Fatalf("expected evidence task id %q, got %q", executeResult.TaskID, pack.TaskID)
	}
	if !pack.ExecutionSuccess {
		t.Fatalf("expected execute evidence to mark real execution success")
	}
	if !strings.Contains(pack.ResultSummary, "database created") {
		t.Fatalf("expected evidence summary to mention database creation, got %q", pack.ResultSummary)
	}
}

func TestServiceExecuteApprovedOrderMarksPlanStaleWithoutCreatingTask(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, _, _ := newPhaseTwoServicesWithState(t, phaseThreeMySQLState{
		existingDatabases: map[string]bool{
			"order_center": true,
		},
	})

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
	if ledger.TraceID != submitResult.TraceID {
		t.Fatalf("expected ledger trace id %q, got %q", submitResult.TraceID, ledger.TraceID)
	}
	if ledger.LatestTaskID != "" {
		t.Fatalf("expected latest task id to stay empty on stale path, got %q", ledger.LatestTaskID)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if pack.TraceID != submitResult.TraceID {
		t.Fatalf("expected stale evidence trace id %q, got %q", submitResult.TraceID, pack.TraceID)
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
	service, _, auditService, _, store, _ := newPhaseTwoServices(t)

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

	orderView, err := service.GetOrder(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	orderView.Status = order.StatusExecuting
	if err := store.SaveOrder(ctx, orderView); err != nil {
		t.Fatalf("SaveOrder returned error: %v", err)
	}
	existingTask := task.ExecutionTask{
		TaskID:      "task_existing",
		OrderID:     submitResult.OrderID,
		TraceID:     submitResult.TraceID,
		ActionName:  "mysql.database.create",
		Status:      task.StatusRunning,
		StartedAt:   time.Now().UTC(),
		HeartbeatAt: time.Now().UTC(),
	}
	if err := store.SaveTask(ctx, existingTask); err != nil {
		t.Fatalf("SaveTask returned error: %v", err)
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

	if second.TaskID != existingTask.TaskID {
		t.Fatalf("expected execute to return existing task %q, got %q", existingTask.TaskID, second.TaskID)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, submitResult.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	if countAuditEvents(ledger, appaudit.EventExecutionStarted) != 0 {
		t.Fatalf("expected no new EXECUTION_STARTED event, got %d", countAuditEvents(ledger, appaudit.EventExecutionStarted))
	}
}

func TestServiceExecuteApprovedOrderRejectsWaitingApprovalOrder(t *testing.T) {
	ctx := context.Background()
	service, _, _, _, _, _ := newPhaseTwoServices(t)

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
	service, _, _, _, _, _ := newPhaseTwoServices(t)

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

func newPhaseTwoServices(t *testing.T) (Service, appapproval.Service, *appaudit.MemoryService, *appevidence.MemoryService, *persistence.MemoryStore, *phaseThreeMySQLAdmin) {
	t.Helper()
	return newPhaseTwoServicesWithState(t, phaseThreeMySQLState{})
}

func newPhaseTwoServicesWithState(t *testing.T, state phaseThreeMySQLState) (Service, appapproval.Service, *appaudit.MemoryService, *appevidence.MemoryService, *persistence.MemoryStore, *phaseThreeMySQLAdmin) {
	t.Helper()
	return newPhaseThreeServices(t, state)
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
