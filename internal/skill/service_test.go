package skill

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appaction "dba_ai_assistant/internal/application/actionrequest"
	appauth "dba_ai_assistant/internal/application/authorization"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/principal"
)

func TestServiceRequestMySQLDatabaseCreateAutoChainsExecuteForApprovedOrders(t *testing.T) {
	t.Helper()

	var submitRequest appaction.ActionRequestDTO
	var executeRequest appaction.ExecuteOrderInput
	var executeHeaders http.Header
	submitCalls := 0
	executeCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/action-requests":
			submitCalls++
			if err := json.NewDecoder(r.Body).Decode(&submitRequest); err != nil {
				t.Fatalf("failed to decode submit request: %v", err)
			}
			writeTestJSON(w, http.StatusAccepted, appaction.ActionSubmissionResult{
				RequestID:        "req_01",
				OrderID:          "ord_01",
				ActionName:       "mysql.database.create",
				Status:           order.StatusApproved,
				ApprovalRequired: false,
				UserMessage:      "请求已创建，无需审批；请通过 execute 显式触发执行。",
				NextPollURI:      "/api/v1/orders/ord_01",
				TraceID:          "trace_01",
			})
		case "/api/v1/orders/ord_01/execute":
			executeCalls++
			executeHeaders = r.Header.Clone()
			if err := json.NewDecoder(r.Body).Decode(&executeRequest); err != nil {
				t.Fatalf("failed to decode execute request: %v", err)
			}
			writeTestJSON(w, http.StatusAccepted, appaction.ExecuteOrderResult{
				OrderID:    "ord_01",
				Status:     order.StatusSucceeded,
				TaskID:     "task_01",
				ExecutorID: "u_requester",
				TraceID:    "trace_01",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service, err := NewService(Dependencies{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	output, err := service.RequestMySQLDatabaseCreate(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_requester",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "deep_agent",
	}, RequestMySQLDatabaseCreateInput{
		Project:         "order-platform",
		Environment:     "test",
		ServiceInstance: "mysql-order-main",
		DatabaseName:    "order_center",
		ConversationID:  "conv_01",
		MessageID:       "msg_01",
		Reason:          "user requested database creation",
		AutoExecute:     true,
	})
	if err != nil {
		t.Fatalf("RequestMySQLDatabaseCreate returned error: %v", err)
	}

	if submitCalls != 1 {
		t.Fatalf("expected 1 submit call, got %d", submitCalls)
	}
	if executeCalls != 1 {
		t.Fatalf("expected 1 execute call, got %d", executeCalls)
	}
	if submitRequest.PrincipalID != "u_requester" {
		t.Fatalf("expected principal_id from auth context, got %q", submitRequest.PrincipalID)
	}
	if submitRequest.ResourceSelector.Project != "order-platform" || submitRequest.ResourceSelector.Environment != "test" || submitRequest.ResourceSelector.ServiceInstance != "mysql-order-main" {
		t.Fatalf("expected business selector to be forwarded, got %#v", submitRequest.ResourceSelector)
	}
	if submitRequest.RequestContext["source"] != "deep_agent" {
		t.Fatalf("expected request_context.source=deep_agent, got %#v", submitRequest.RequestContext["source"])
	}
	if submitRequest.RequestContext["conversation_id"] != "conv_01" {
		t.Fatalf("expected conversation id to be forwarded")
	}
	if executeRequest.OrderID != "ord_01" {
		t.Fatalf("expected execute order id, got %q", executeRequest.OrderID)
	}
	if got := executeHeaders.Get("X-Principal-ID"); got != "u_requester" {
		t.Fatalf("expected execute header principal, got %q", got)
	}
	if got := executeHeaders.Get("X-Roles"); !strings.Contains(got, principal.RoleMySQLOperator) {
		t.Fatalf("expected execute roles header to include mysql_operator, got %q", got)
	}
	if !output.AutoExecuteTriggered {
		t.Fatalf("expected auto execute to be triggered")
	}
	if output.AutoExecuteResult == nil {
		t.Fatalf("expected auto execute result")
	}
	if output.AutoExecuteResult.TaskID != "task_01" {
		t.Fatalf("expected auto execute task id")
	}
	if output.OrderStatus != order.StatusSucceeded {
		t.Fatalf("expected final order status SUCCEEDED, got %s", output.OrderStatus)
	}
	if !strings.Contains(output.UserMessage, "自动") {
		t.Fatalf("expected final user message to mention auto execute, got %q", output.UserMessage)
	}
}

func TestServiceRequestMySQLDatabaseCreateStopsAtApprovalBoundary(t *testing.T) {
	t.Helper()

	executeCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/action-requests":
			writeTestJSON(w, http.StatusAccepted, appaction.ActionSubmissionResult{
				RequestID:        "req_02",
				OrderID:          "ord_02",
				ActionName:       "mysql.database.create",
				Status:           order.StatusWaitingApproval,
				ApprovalRequired: true,
				UserMessage:      "请求已创建，等待审批。",
				NextPollURI:      "/api/v1/orders/ord_02",
				TraceID:          "trace_02",
			})
		case "/api/v1/orders/ord_02/execute":
			executeCalls++
			t.Fatalf("execute must not be called for approval-required order")
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service, err := NewService(Dependencies{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	output, err := service.RequestMySQLDatabaseCreate(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_requester",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "deep_agent",
	}, RequestMySQLDatabaseCreateInput{
		Project:         "order-platform",
		Environment:     "prod",
		ServiceInstance: "mysql-order-main",
		DatabaseName:    "order_center",
		AutoExecute:     true,
	})
	if err != nil {
		t.Fatalf("RequestMySQLDatabaseCreate returned error: %v", err)
	}

	if executeCalls != 0 {
		t.Fatalf("expected no execute calls, got %d", executeCalls)
	}
	if output.AutoExecuteTriggered {
		t.Fatalf("expected auto execute to stay disabled for approval-required order")
	}
	if !output.ApprovalRequired {
		t.Fatalf("expected approval_required=true")
	}
	if output.OrderStatus != order.StatusWaitingApproval {
		t.Fatalf("expected order status WAITING_APPROVAL, got %s", output.OrderStatus)
	}
}

func TestServiceRequestMySQLDatabaseCreatePreservesRequestResultWhenAutoExecuteFails(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/action-requests":
			writeTestJSON(w, http.StatusAccepted, appaction.ActionSubmissionResult{
				RequestID:        "req_03",
				OrderID:          "ord_03",
				ActionName:       "mysql.database.create",
				Status:           order.StatusApproved,
				ApprovalRequired: false,
				UserMessage:      "请求已创建，无需审批；请通过 execute 显式触发执行。",
				NextPollURI:      "/api/v1/orders/ord_03",
				TraceID:          "trace_03",
			})
		case "/api/v1/orders/ord_03/execute":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"EXECUTOR_NOT_ALLOWED","message":"executor is not allowed to trigger this order","details":{"order_id":"ord_03"}},"trace_id":"trace_exec_03"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service, err := NewService(Dependencies{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	output, err := service.RequestMySQLDatabaseCreate(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_requester",
		Roles:                    []string{principal.RoleAssistantUser},
		Source:                   "deep_agent",
	}, RequestMySQLDatabaseCreateInput{
		Project:         "order-platform",
		Environment:     "test",
		ServiceInstance: "mysql-order-main",
		DatabaseName:    "order_center",
		AutoExecute:     true,
	})
	if err != nil {
		t.Fatalf("request skill must preserve successful request even if auto execute fails: %v", err)
	}

	if !output.AutoExecuteTriggered {
		t.Fatalf("expected auto execute attempt to be recorded")
	}
	if output.AutoExecuteError == nil {
		t.Fatalf("expected auto execute error details")
	}
	if output.AutoExecuteError.Code != string(common.CodeExecutorNotAllowed) {
		t.Fatalf("expected EXECUTOR_NOT_ALLOWED, got %s", output.AutoExecuteError.Code)
	}
	if output.AutoExecuteError.TraceID != "trace_exec_03" {
		t.Fatalf("expected execute error trace id to be preserved")
	}
	if output.OrderStatus != order.StatusApproved {
		t.Fatalf("expected request result to remain APPROVED, got %s", output.OrderStatus)
	}
	if !strings.Contains(output.UserMessage, "显式") {
		t.Fatalf("expected fallback guidance in user message, got %q", output.UserMessage)
	}
}

func TestServiceExecuteAssistantOrderCallsControlAPI(t *testing.T) {
	t.Helper()

	var executeRequest appaction.ExecuteOrderInput
	var executeHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orders/ord_04/execute" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		executeHeaders = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&executeRequest); err != nil {
			t.Fatalf("failed to decode execute request: %v", err)
		}
		writeTestJSON(w, http.StatusAccepted, appaction.ExecuteOrderResult{
			OrderID:    "ord_04",
			Status:     order.StatusSucceeded,
			TaskID:     "task_04",
			ExecutorID: "u_executor",
			TraceID:    "trace_04",
		})
	}))
	defer server.Close()

	service, err := NewService(Dependencies{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	output, err := service.ExecuteAssistantOrder(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_executor",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "deep_agent",
	}, ExecuteAssistantOrderInput{
		OrderID: "ord_04",
		Reason:  "auto-chain from request_mysql_database_create",
	})
	if err != nil {
		t.Fatalf("ExecuteAssistantOrder returned error: %v", err)
	}

	if executeRequest.OrderID != "ord_04" {
		t.Fatalf("expected execute request order id to be injected from path")
	}
	if executeRequest.Reason == "" {
		t.Fatalf("expected execute reason to be forwarded")
	}
	if executeHeaders.Get("X-Principal-ID") != "u_executor" {
		t.Fatalf("expected execute principal header")
	}
	if output.OrderStatus != order.StatusSucceeded {
		t.Fatalf("expected order status SUCCEEDED")
	}
	if output.UserMessage == "" {
		t.Fatalf("expected execute user message")
	}
}

func writeTestJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
