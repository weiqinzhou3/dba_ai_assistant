package actionrequest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"dba_ai_assistant/internal/adapters/dbnative"
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
	"dba_ai_assistant/internal/domain/task"
	"dba_ai_assistant/internal/persistence"
)

func TestServiceSubmitRejectsInvalidDatabaseNameInPhaseThree(t *testing.T) {
	ctx := context.Background()
	service, _, _, _, _, _ := newPhaseThreeServices(t, phaseThreeMySQLState{})

	_, err := service.Submit(ctx, ActionRequestDTO{
		PrincipalID: "u_requester",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "9bad-name",
		},
	})
	if err == nil {
		t.Fatalf("expected invalid database name to be rejected")
	}
	if code := common.ErrorCode(err); code != common.CodeRequestInvalid {
		t.Fatalf("expected %s, got %s", common.CodeRequestInvalid, code)
	}
}

func TestServiceExecuteApprovedOrderSucceedsWithRealTerminalAuditAndEvidence(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, _, admin := newPhaseThreeServices(t, phaseThreeMySQLState{})

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

	taskView, err := service.GetTask(ctx, executeResult.TaskID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if taskView.Status != task.StatusSucceeded {
		t.Fatalf("expected terminal task status SUCCEEDED, got %s", taskView.Status)
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
	if ledger.LatestOrderStatus != string(order.StatusSucceeded) {
		t.Fatalf("expected latest order status SUCCEEDED, got %q", ledger.LatestOrderStatus)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if !pack.ExecutionSuccess {
		t.Fatalf("expected evidence to mark real execution success")
	}
	if before, _ := pack.BeforeStateSnapshot["database_exists"].(bool); before {
		t.Fatalf("expected before snapshot database_exists=false, got %#v", pack.BeforeStateSnapshot["database_exists"])
	}
	if after, _ := pack.AfterStateSnapshot["database_exists"].(bool); !after {
		t.Fatalf("expected after snapshot database_exists=true, got %#v", pack.AfterStateSnapshot["database_exists"])
	}
	if !strings.Contains(pack.ResultSummary, "database created") {
		t.Fatalf("expected success summary to mention database created, got %q", pack.ResultSummary)
	}
	if len(pack.ArtifactRefs) == 0 {
		t.Fatalf("expected success evidence to include artifact refs")
	}
	if got := admin.createCount("order_center"); got != 1 {
		t.Fatalf("expected exactly one CREATE DATABASE call, got %d", got)
	}
}

func TestServiceExecuteApprovedOrderMarksPlanStaleWhenDatabaseAlreadyExists(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, _, admin := newPhaseThreeServices(t, phaseThreeMySQLState{
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
		t.Fatalf("expected no task to be created on stale path, got %q", executeResult.TaskID)
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
	assertAuditEvent(t, ledger, appaudit.EventPlanStale)
	if ledger.LatestTaskID != "" {
		t.Fatalf("expected no task on stale path, got %q", ledger.LatestTaskID)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if pack.ExecutionSuccess {
		t.Fatalf("expected stale evidence to be unsuccessful")
	}
	if code, _ := pack.FailureDetail["code"].(string); code != string(common.CodePlanStale) {
		t.Fatalf("expected stale evidence code %s, got %#v", common.CodePlanStale, pack.FailureDetail["code"])
	}
	if got := admin.createCount("order_center"); got != 0 {
		t.Fatalf("expected no CREATE DATABASE call on stale path, got %d", got)
	}
}

func TestServiceExecuteApprovedOrderReturnsIdempotencyConflictWhenKeyAlreadyRunning(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, store, admin := newPhaseThreeServices(t, phaseThreeMySQLState{})

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
	if err := store.SaveIdempotencyRecord(ctx, appexec.IdempotencyRecord{
		IdempotencyKey: "mysql.database.create:dbt_1001:order_center",
		OrderID:        "ord_running_other",
		TaskID:         "task_running_other",
		TaskStatus:     task.StatusRunning,
		UpdatedAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveIdempotencyRecord returned error: %v", err)
	}

	_, err = service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "manual execute",
	})
	if err == nil {
		t.Fatalf("expected idempotency conflict to reject execution")
	}
	if code := common.ErrorCode(err); code != common.CodeIdempotencyConflict {
		t.Fatalf("expected %s, got %s", common.CodeIdempotencyConflict, code)
	}

	orderView, err := service.GetOrder(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if orderView.Status != order.StatusApproved {
		t.Fatalf("expected order to remain APPROVED after conflict, got %s", orderView.Status)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, submitResult.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventExecutionFailed)
	if ledger.LatestTaskID != "" {
		t.Fatalf("expected no task to be created on idempotency conflict, got %q", ledger.LatestTaskID)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if pack.ExecutionSuccess {
		t.Fatalf("expected conflict evidence to be unsuccessful")
	}
	if code, _ := pack.FailureDetail["code"].(string); code != string(common.CodeIdempotencyConflict) {
		t.Fatalf("expected failure_detail.code=%s, got %#v", common.CodeIdempotencyConflict, pack.FailureDetail["code"])
	}
	if got := admin.createCount("order_center"); got != 0 {
		t.Fatalf("expected no CREATE DATABASE call on conflict path, got %d", got)
	}
}

func TestServiceExecuteApprovedOrderReplaysSuccessfulIdempotentKeyWithoutRecreatingDatabase(t *testing.T) {
	ctx := context.Background()
	service, _, auditService, evidenceService, store, admin := newPhaseThreeServices(t, phaseThreeMySQLState{
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
	if err := store.SaveIdempotencyRecord(ctx, appexec.IdempotencyRecord{
		IdempotencyKey: "mysql.database.create:dbt_1001:order_center",
		OrderID:        "ord_success_other",
		TaskID:         "task_success_other",
		TaskStatus:     task.StatusSucceeded,
		Summary:        "database created by previous order",
		UpdatedAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveIdempotencyRecord returned error: %v", err)
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
		t.Fatalf("expected SUCCEEDED result, got %s", executeResult.Status)
	}
	if executeResult.TaskID == "" {
		t.Fatalf("expected replay path to materialize a terminal task")
	}

	taskView, err := service.GetTask(ctx, executeResult.TaskID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if taskView.Status != task.StatusSucceeded {
		t.Fatalf("expected replay task to be terminal SUCCEEDED, got %s", taskView.Status)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, submitResult.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventExecutionSucceeded)

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if !pack.ExecutionSuccess {
		t.Fatalf("expected replay evidence to be successful")
	}
	if !strings.Contains(strings.ToLower(pack.ResultSummary), "idempotent") {
		t.Fatalf("expected replay summary to mention idempotent reuse, got %q", pack.ResultSummary)
	}
	if got := admin.createCount("order_center"); got != 0 {
		t.Fatalf("expected no fresh CREATE DATABASE call on replay path, got %d", got)
	}
}

func TestServiceExecuteApprovedOrderAllowsRetryAfterFailedIdempotentRecord(t *testing.T) {
	ctx := context.Background()
	service, _, _, evidenceService, store, admin := newPhaseThreeServices(t, phaseThreeMySQLState{})

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
	if err := store.SaveIdempotencyRecord(ctx, appexec.IdempotencyRecord{
		IdempotencyKey: "mysql.database.create:dbt_1001:order_center",
		OrderID:        "ord_failed_other",
		TaskID:         "task_failed_other",
		TaskStatus:     task.StatusFailed,
		Summary:        "previous execution failed",
		UpdatedAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveIdempotencyRecord returned error: %v", err)
	}

	executeResult, err := service.ExecuteApprovedOrder(ctx, appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "http",
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "manual retry after previous failure",
	})
	if err != nil {
		t.Fatalf("ExecuteApprovedOrder returned error: %v", err)
	}
	if executeResult.Status != order.StatusSucceeded {
		t.Fatalf("expected retry path to succeed, got %s", executeResult.Status)
	}
	if got := admin.createCount("order_center"); got != 1 {
		t.Fatalf("expected retry to invoke CREATE DATABASE exactly once, got %d", got)
	}

	pack, err := evidenceService.GetByOrderID(ctx, submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if !pack.ExecutionSuccess {
		t.Fatalf("expected retry evidence to be successful")
	}
}

type phaseThreeMySQLState struct {
	existingDatabases map[string]bool
	pingErr           error
	connectionErrs    map[string]error
	createErrs        map[string]error
	createNoEffect    bool
}

func newPhaseThreeServices(t *testing.T, state phaseThreeMySQLState) (Service, appapproval.Service, *appaudit.MemoryService, *appevidence.MemoryService, *persistence.MemoryStore, *phaseThreeMySQLAdmin) {
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
	admin := newPhaseThreeMySQLAdmin(state)
	adapter := dbnative.New(dbnative.Dependencies{
		ConnectionResolver: &phaseThreeConnectionResolver{
			configs: map[string]dbnative.ConnectionConfig{
				"secret://db-targets/mysql-order-main-test": {
					Engine: "mysql",
					DSN:    "mysql-test-dsn",
				},
				"secret://db-targets/mysql-order-main-prod": {
					Engine: "mysql",
					DSN:    "mysql-prod-dsn",
				},
			},
			errs: state.connectionErrs,
		},
		MySQL: admin,
		Now:   fixedPhaseThreeNow,
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
		Router:      appexec.NewStaticExecutionRouter(adapter),
		Runtime:     appexec.NewSynchronousTaskRuntime(store),
		Approval:    approvalService,
		Audit:       auditService,
		Evidence:    evidenceService,
		Requests:    store,
		Orders:      store,
		Plans:       store,
		Tasks:       store,
		Idempotency: store,
	})
	return actionService, approvalService, auditService, evidenceService, store, admin
}

func fixedPhaseThreeNow() time.Time {
	return time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
}

type phaseThreeConnectionResolver struct {
	configs map[string]dbnative.ConnectionConfig
	errs    map[string]error
}

func (r *phaseThreeConnectionResolver) Resolve(_ context.Context, connectionRef string) (dbnative.ConnectionConfig, error) {
	if err := r.errs[connectionRef]; err != nil {
		return dbnative.ConnectionConfig{}, err
	}
	cfg, ok := r.configs[connectionRef]
	if !ok {
		return dbnative.ConnectionConfig{}, errors.New("connection ref not found")
	}
	return cfg, nil
}

type phaseThreeMySQLAdmin struct {
	existing       map[string]bool
	pingErr        error
	createErrs     map[string]error
	createNoEffect bool
	createCalls    []string
}

func newPhaseThreeMySQLAdmin(state phaseThreeMySQLState) *phaseThreeMySQLAdmin {
	existing := map[string]bool{}
	for name, value := range state.existingDatabases {
		existing[name] = value
	}
	return &phaseThreeMySQLAdmin{
		existing:       existing,
		pingErr:        state.pingErr,
		createErrs:     state.createErrs,
		createNoEffect: state.createNoEffect,
	}
}

func (a *phaseThreeMySQLAdmin) Ping(_ context.Context, _ dbnative.ConnectionConfig) error {
	return a.pingErr
}

func (a *phaseThreeMySQLAdmin) DatabaseExists(_ context.Context, _ dbnative.ConnectionConfig, databaseName string) (bool, error) {
	return a.existing[databaseName], nil
}

func (a *phaseThreeMySQLAdmin) CreateDatabase(_ context.Context, _ dbnative.ConnectionConfig, databaseName string) error {
	a.createCalls = append(a.createCalls, databaseName)
	if err := a.createErrs[databaseName]; err != nil {
		return err
	}
	if !a.createNoEffect {
		a.existing[databaseName] = true
	}
	return nil
}

func (a *phaseThreeMySQLAdmin) createCount(databaseName string) int {
	count := 0
	for _, value := range a.createCalls {
		if value == databaseName {
			count++
		}
	}
	return count
}
