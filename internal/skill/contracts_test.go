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
}
