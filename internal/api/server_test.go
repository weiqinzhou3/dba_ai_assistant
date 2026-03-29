package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appaction "dba_ai_assistant/internal/application/actionrequest"
	appapproval "dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
	"dba_ai_assistant/internal/persistence"
)

func TestServerRoutesActionRequestsThroughUnifiedEntry(t *testing.T) {
	actionService := &stubActionRequestService{
		submitResult: appaction.ActionSubmissionResult{
			RequestID:        "req_01",
			OrderID:          "ord_01",
			ActionName:       "mysql.database.create",
			Status:           order.StatusApproved,
			ApprovalRequired: false,
			TraceID:          "trace_01",
		},
	}

	server := NewServer(Dependencies{
		ActionRequests: actionService,
	})

	body := []byte(`{"principal_id":"u_1001","action_hint":"mysql.database.create","resource_selector":{"project":"order-platform","environment":"test","service_instance":"mysql-order-main"},"parameters":{"database_name":"order_center"}}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/action-requests", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", recorder.Code)
	}
	if !actionService.submitCalled {
		t.Fatalf("expected action request service to be used as unified entry")
	}

	var response appaction.ActionSubmissionResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.TraceID != "trace_01" {
		t.Fatalf("expected trace id in response, got %q", response.TraceID)
	}
}

func TestServerMapsExecuteApprovalGateErrors(t *testing.T) {
	actionService := &stubActionRequestService{
		executeErr: common.NewError(common.CodeApprovalRequired, "order still requires approval", nil),
	}

	server := NewServer(Dependencies{
		ActionRequests: actionService,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/ord_01/execute", bytes.NewReader([]byte(`{"reason":"please run"}`)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Principal-ID", "u_2001")
	request.Header.Set("X-Roles", "mysql_operator")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}

	var response errorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if response.Error.Code != string(common.CodeApprovalRequired) {
		t.Fatalf("expected error code %s, got %s", common.CodeApprovalRequired, response.Error.Code)
	}
}

func TestServerReturnsTraceIDOnNorthboundResponses(t *testing.T) {
	actionService := &stubActionRequestService{
		getOrderResult: appaction.AssistantOrderView{
			OrderID: "ord_01",
			TraceID: "trace_01",
		},
		getTaskResult: appaction.ExecutionTaskView{
			TaskID:     "task_01",
			OrderID:    "ord_01",
			TraceID:    "trace_01",
			Status:     "RUNNING",
			ActionName: "mysql.database.create",
		},
	}
	approvalService := &stubApprovalService{
		decideResult: appapproval.State{
			OrderID: "ord_01",
			TraceID: "trace_01",
			Status:  order.StatusApproved,
		},
	}
	auditService := &stubAuditService{
		view: appaudit.LedgerView{
			RequestID: "req_01",
			TraceID:   "trace_01",
		},
	}
	evidenceService := &stubEvidenceService{
		view: appevidence.PackView{
			OrderID: "ord_01",
			TraceID: "trace_01",
		},
	}

	server := NewServer(Dependencies{
		ActionRequests: actionService,
		Approvals:      approvalService,
		Audit:          auditService,
		Evidence:       evidenceService,
	})

	assertTrace := func(method string, path string, body []byte) {
		t.Helper()

		request := httptest.NewRequest(method, path, bytes.NewReader(body))
		if len(body) > 0 {
			request.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)

		if recorder.Code < 200 || recorder.Code >= 300 {
			t.Fatalf("expected success for %s %s, got %d", method, path, recorder.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode response for %s %s: %v", method, path, err)
		}
		if payload["trace_id"] != "trace_01" {
			t.Fatalf("expected trace_id on %s %s response, got %#v", method, path, payload["trace_id"])
		}
	}

	assertTrace(http.MethodGet, "/api/v1/orders/ord_01", nil)
	assertTrace(http.MethodGet, "/api/v1/tasks/task_01", nil)
	assertTrace(http.MethodPost, "/api/v1/orders/ord_01/approvals", []byte(`{"approver_id":"u_2001","decision":"APPROVE"}`))
	assertTrace(http.MethodGet, "/api/v1/audit-ledger/req_01", nil)
	assertTrace(http.MethodGet, "/api/v1/evidence-packs/ord_01", nil)
}

func TestServerRoutesApprovalsWithoutCallingExecute(t *testing.T) {
	actionService := &stubActionRequestService{}
	approvalService := &stubApprovalService{
		decideResult: appapproval.State{
			OrderID: "ord_01",
			Status:  order.StatusApproved,
			TraceID: "trace_01",
		},
	}

	server := NewServer(Dependencies{
		ActionRequests: actionService,
		Approvals:      approvalService,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/ord_01/approvals", bytes.NewReader([]byte(`{"approver_id":"u_2001","decision":"APPROVE"}`)))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", recorder.Code)
	}
	if !approvalService.decideCalled {
		t.Fatalf("expected approval service to handle approval route")
	}
	if actionService.executeCalled {
		t.Fatalf("approval route must not trigger execute path")
	}
}

func TestServerSupportsManualControlFlowOverHTTP(t *testing.T) {
	server := newPhaseTwoHTTPServer(t)

	submitBody := []byte(`{"principal_id":"u_requester","action_hint":"mysql.database.create","resource_selector":{"project":"order-platform","environment":"prod","service_instance":"mysql-order-main"},"parameters":{"database_name":"order_center"}}`)
	submitRequest := httptest.NewRequest(http.MethodPost, "/api/v1/action-requests", bytes.NewReader(submitBody))
	submitRequest.Header.Set("Content-Type", "application/json")
	submitRecorder := httptest.NewRecorder()
	server.ServeHTTP(submitRecorder, submitRequest)
	if submitRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected submit to return 202, got %d", submitRecorder.Code)
	}

	var submitResult appaction.ActionSubmissionResult
	if err := json.Unmarshal(submitRecorder.Body.Bytes(), &submitResult); err != nil {
		t.Fatalf("failed to decode submit response: %v", err)
	}
	if submitResult.Status != order.StatusWaitingApproval {
		t.Fatalf("expected waiting approval status, got %s", submitResult.Status)
	}

	approvalBody := []byte(`{"approver_id":"u_approver","decision":"APPROVE","comment":"approved"}`)
	approvalRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+submitResult.OrderID+"/approvals", bytes.NewReader(approvalBody))
	approvalRequest.Header.Set("Content-Type", "application/json")
	approvalRecorder := httptest.NewRecorder()
	server.ServeHTTP(approvalRecorder, approvalRequest)
	if approvalRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected approval to return 202, got %d", approvalRecorder.Code)
	}

	executeBody := []byte(`{"reason":"manual execute"}`)
	executeRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+submitResult.OrderID+"/execute", bytes.NewReader(executeBody))
	executeRequest.Header.Set("Content-Type", "application/json")
	executeRequest.Header.Set("X-Principal-ID", "u_executor")
	executeRequest.Header.Set("X-Roles", principal.RoleMySQLOperator)
	executeRecorder := httptest.NewRecorder()
	server.ServeHTTP(executeRecorder, executeRequest)
	if executeRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected execute to return 202, got %d", executeRecorder.Code)
	}

	var executeResult appaction.ExecuteOrderResult
	if err := json.Unmarshal(executeRecorder.Body.Bytes(), &executeResult); err != nil {
		t.Fatalf("failed to decode execute response: %v", err)
	}
	if executeResult.Status != order.StatusExecuting {
		t.Fatalf("expected execute to move order to EXECUTING, got %s", executeResult.Status)
	}
	if executeResult.TaskID == "" {
		t.Fatalf("expected execute to create a task id")
	}

	auditRequest := httptest.NewRequest(http.MethodGet, "/api/v1/audit-ledger/"+submitResult.RequestID, nil)
	auditRecorder := httptest.NewRecorder()
	server.ServeHTTP(auditRecorder, auditRequest)
	if auditRecorder.Code != http.StatusOK {
		t.Fatalf("expected audit query to return 200, got %d", auditRecorder.Code)
	}

	var ledger appaudit.LedgerView
	if err := json.Unmarshal(auditRecorder.Body.Bytes(), &ledger); err != nil {
		t.Fatalf("failed to decode audit ledger response: %v", err)
	}
	if ledger.TraceID != submitResult.TraceID {
		t.Fatalf("expected ledger trace id %q, got %q", submitResult.TraceID, ledger.TraceID)
	}
	if ledger.LatestOrderStatus != string(order.StatusExecuting) {
		t.Fatalf("expected latest order status EXECUTING, got %q", ledger.LatestOrderStatus)
	}
	if ledger.LatestTaskID != executeResult.TaskID {
		t.Fatalf("expected latest task id %q, got %q", executeResult.TaskID, ledger.LatestTaskID)
	}

	evidenceRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evidence-packs/"+submitResult.OrderID, nil)
	evidenceRecorder := httptest.NewRecorder()
	server.ServeHTTP(evidenceRecorder, evidenceRequest)
	if evidenceRecorder.Code != http.StatusOK {
		t.Fatalf("expected evidence query to return 200, got %d", evidenceRecorder.Code)
	}

	var pack appevidence.PackView
	if err := json.Unmarshal(evidenceRecorder.Body.Bytes(), &pack); err != nil {
		t.Fatalf("failed to decode evidence response: %v", err)
	}
	if pack.TraceID != submitResult.TraceID {
		t.Fatalf("expected evidence trace id %q, got %q", submitResult.TraceID, pack.TraceID)
	}
	if pack.TaskID != executeResult.TaskID {
		t.Fatalf("expected evidence task id %q, got %q", executeResult.TaskID, pack.TaskID)
	}
}

type stubActionRequestService struct {
	submitResult   appaction.ActionSubmissionResult
	getOrderResult appaction.AssistantOrderView
	getTaskResult  appaction.ExecutionTaskView
	executeResult  appaction.ExecuteOrderResult
	submitErr      error
	executeErr     error
	submitCalled   bool
	executeCalled  bool
}

func (s *stubActionRequestService) Submit(_ context.Context, _ appaction.ActionRequestDTO) (appaction.ActionSubmissionResult, error) {
	s.submitCalled = true
	return s.submitResult, s.submitErr
}

func (s *stubActionRequestService) GetOrder(_ context.Context, _ string) (appaction.AssistantOrderView, error) {
	return s.getOrderResult, nil
}

func (s *stubActionRequestService) GetTask(_ context.Context, _ string) (appaction.ExecutionTaskView, error) {
	return s.getTaskResult, nil
}

func (s *stubActionRequestService) ExecuteApprovedOrder(_ context.Context, _ appauth.AuthContext, _ appaction.ExecuteOrderInput) (appaction.ExecuteOrderResult, error) {
	s.executeCalled = true
	return s.executeResult, s.executeErr
}

type noopApprovalService struct{}

func (n *noopApprovalService) Create(_ context.Context, _ order.AssistantOrder, _ appapproval.CreateInput) (appapproval.State, error) {
	return appapproval.State{}, nil
}

func (n *noopApprovalService) Decide(_ context.Context, _ string, _ appapproval.DecisionInput) (appapproval.State, error) {
	return appapproval.State{}, nil
}

func (n *noopApprovalService) Get(_ context.Context, _ string) (appapproval.State, error) {
	return appapproval.State{}, nil
}

func (n *noopApprovalService) ExpireStaleApprovals(_ context.Context, _ time.Time, _ int) ([]appapproval.ExpiryResult, error) {
	return nil, nil
}

type stubApprovalService struct {
	decideResult appapproval.State
	decideErr    error
	decideCalled bool
}

func (s *stubApprovalService) Create(_ context.Context, _ order.AssistantOrder, _ appapproval.CreateInput) (appapproval.State, error) {
	return appapproval.State{}, nil
}

func (s *stubApprovalService) Decide(_ context.Context, _ string, _ appapproval.DecisionInput) (appapproval.State, error) {
	s.decideCalled = true
	return s.decideResult, s.decideErr
}

func (s *stubApprovalService) Get(_ context.Context, _ string) (appapproval.State, error) {
	return appapproval.State{}, nil
}

func (s *stubApprovalService) ExpireStaleApprovals(_ context.Context, _ time.Time, _ int) ([]appapproval.ExpiryResult, error) {
	return nil, nil
}

type noopAuditService struct{}

func (n *noopAuditService) AppendEvent(_ context.Context, _ appaudit.Event) error { return nil }
func (n *noopAuditService) ListEventsByRequestID(_ context.Context, _ string) ([]appaudit.Event, error) {
	return nil, nil
}
func (n *noopAuditService) GetViewByRequestID(_ context.Context, _ string) (appaudit.LedgerView, error) {
	return appaudit.LedgerView{}, nil
}

type stubAuditService struct {
	view appaudit.LedgerView
}

func (s *stubAuditService) AppendEvent(_ context.Context, _ appaudit.Event) error { return nil }
func (s *stubAuditService) ListEventsByRequestID(_ context.Context, _ string) ([]appaudit.Event, error) {
	return s.view.Events, nil
}
func (s *stubAuditService) GetViewByRequestID(_ context.Context, _ string) (appaudit.LedgerView, error) {
	return s.view, nil
}

type noopEvidenceService struct{}

func (n *noopEvidenceService) Build(_ context.Context, _ appevidence.BuildInput) (appevidence.Pack, error) {
	return appevidence.Pack{}, nil
}
func (n *noopEvidenceService) GetByOrderID(_ context.Context, _ string) (appevidence.PackView, error) {
	return appevidence.PackView{}, nil
}

type stubEvidenceService struct {
	view appevidence.PackView
}

func (s *stubEvidenceService) Build(_ context.Context, _ appevidence.BuildInput) (appevidence.Pack, error) {
	return appevidence.Pack{}, nil
}
func (s *stubEvidenceService) GetByOrderID(_ context.Context, _ string) (appevidence.PackView, error) {
	return s.view, nil
}

func newPhaseTwoHTTPServer(t *testing.T) http.Handler {
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
	actionService := appaction.NewService(appaction.Dependencies{
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

	return NewServer(Dependencies{
		ActionRequests: actionService,
		Approvals:      approvalService,
		Audit:          auditService,
		Evidence:       evidenceService,
	})
}
