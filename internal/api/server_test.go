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
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
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

type stubActionRequestService struct {
	submitResult appaction.ActionSubmissionResult
	submitErr    error
	executeErr   error
	submitCalled bool
}

func (s *stubActionRequestService) Submit(_ context.Context, _ appaction.ActionRequestDTO) (appaction.ActionSubmissionResult, error) {
	s.submitCalled = true
	return s.submitResult, s.submitErr
}

func (s *stubActionRequestService) GetOrder(_ context.Context, _ string) (appaction.AssistantOrderView, error) {
	return appaction.AssistantOrderView{}, nil
}

func (s *stubActionRequestService) GetTask(_ context.Context, _ string) (appaction.ExecutionTaskView, error) {
	return appaction.ExecutionTaskView{}, nil
}

func (s *stubActionRequestService) ExecuteApprovedOrder(_ context.Context, _ appauth.AuthContext, _ appaction.ExecuteOrderInput) (appaction.ExecuteOrderResult, error) {
	return appaction.ExecuteOrderResult{}, s.executeErr
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

type noopAuditService struct{}

func (n *noopAuditService) AppendEvent(_ context.Context, _ appaudit.Event) error { return nil }
func (n *noopAuditService) ListEventsByRequestID(_ context.Context, _ string) ([]appaudit.Event, error) {
	return nil, nil
}
func (n *noopAuditService) GetViewByRequestID(_ context.Context, _ string) (appaudit.LedgerView, error) {
	return appaudit.LedgerView{}, nil
}

type noopEvidenceService struct{}

func (n *noopEvidenceService) Build(_ context.Context, _ appevidence.BuildInput) (appevidence.Pack, error) {
	return appevidence.Pack{}, nil
}
func (n *noopEvidenceService) GetByOrderID(_ context.Context, _ string) (appevidence.PackView, error) {
	return appevidence.PackView{}, nil
}
