package skill

import (
	"testing"

	appaction "dba_ai_assistant/internal/application/actionrequest"
	"dba_ai_assistant/internal/domain/order"
)

func TestMapRequestMySQLDatabaseCreateOutputFromActionSubmissionResult(t *testing.T) {
	got := MapRequestMySQLDatabaseCreateOutput(appaction.ActionSubmissionResult{
		RequestID:        "req_01",
		OrderID:          "ord_01",
		ActionName:       "mysql.database.create",
		Status:           order.StatusApproved,
		ApprovalRequired: false,
		UserMessage:      "请求已创建，无需审批；请通过 execute 显式触发执行。",
		NextPollURI:      "/api/v1/orders/ord_01",
		TraceID:          "trace_01",
	})

	if got.RequestID != "req_01" {
		t.Fatalf("expected request id to be mapped")
	}
	if got.OrderID != "ord_01" {
		t.Fatalf("expected order id to be mapped")
	}
	if got.ActionName != "mysql.database.create" {
		t.Fatalf("expected action name to be mapped")
	}
	if got.OrderStatus != order.StatusApproved {
		t.Fatalf("expected order status to be mapped")
	}
	if got.ApprovalRequired {
		t.Fatalf("expected approval_required=false")
	}
	if got.UserMessage == "" {
		t.Fatalf("expected user message to be mapped")
	}
	if got.NextPollURI != "/api/v1/orders/ord_01" {
		t.Fatalf("expected next poll uri to be mapped")
	}
	if got.TraceID != "trace_01" {
		t.Fatalf("expected trace id to be mapped")
	}
}

func TestMapExecuteAssistantOrderOutputFromExecuteOrderResult(t *testing.T) {
	got := MapExecuteAssistantOrderOutput(appaction.ExecuteOrderResult{
		OrderID:    "ord_01",
		TaskID:     "task_01",
		Status:     order.StatusSucceeded,
		ExecutorID: "u_executor",
		TraceID:    "trace_01",
	})

	if got.OrderID != "ord_01" {
		t.Fatalf("expected order id to be mapped")
	}
	if got.TaskID != "task_01" {
		t.Fatalf("expected task id to be mapped")
	}
	if got.OrderStatus != order.StatusSucceeded {
		t.Fatalf("expected order status to be mapped")
	}
	if got.ExecutorID != "u_executor" {
		t.Fatalf("expected executor id to be mapped")
	}
	if got.TraceID != "trace_01" {
		t.Fatalf("expected trace id to be mapped")
	}
	if got.UserMessage == "" {
		t.Fatalf("expected execute user message to be populated")
	}
}
