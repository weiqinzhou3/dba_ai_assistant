package skill

import (
	"testing"

	appaction "dba_ai_assistant/internal/application/actionrequest"
	"dba_ai_assistant/internal/domain/order"
)

func TestMapRequestMySQLDatabaseCreateOutputFromActionSubmissionResult(t *testing.T) {
	got := MapRequestMySQLDatabaseCreateOutput(appaction.ActionSubmissionResult{
		OrderID:          "ord_01",
		Status:           order.StatusApproved,
		ApprovalRequired: false,
		TraceID:          "trace_01",
	})

	if got.OrderID != "ord_01" {
		t.Fatalf("expected order id to be mapped")
	}
	if got.OrderStatus != order.StatusApproved {
		t.Fatalf("expected order status to be mapped")
	}
	if got.RequiresApproval {
		t.Fatalf("expected requires_approval=false")
	}
	if got.TraceID != "trace_01" {
		t.Fatalf("expected trace id to be mapped")
	}
}

func TestMapExecuteAssistantOrderOutputFromExecuteOrderResult(t *testing.T) {
	got := MapExecuteAssistantOrderOutput(appaction.ExecuteOrderResult{
		TaskID:  "task_01",
		Status:  order.StatusExecuting,
		TraceID: "trace_01",
	})

	if got.TaskID != "task_01" {
		t.Fatalf("expected task id to be mapped")
	}
	if got.TaskStatus != string(order.StatusExecuting) {
		t.Fatalf("expected task status to be mapped")
	}
	if got.TraceID != "trace_01" {
		t.Fatalf("expected trace id to be mapped")
	}
}
