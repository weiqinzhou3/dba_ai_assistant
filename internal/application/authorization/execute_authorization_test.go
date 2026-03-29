package authorization

import (
	"context"
	"testing"

	"dba_ai_assistant/internal/domain/principal"
)

func TestStaticExecuteAuthorizationServiceAllowsControlExecutor(t *testing.T) {
	service := NewStaticExecuteAuthorizationService()

	decision, err := service.Authorize(context.Background(), ExecuteAuthorizationInput{
		ActionName: "mysql.database.create",
		OrderID:    "ord_01",
		Executor: principal.Principal{
			PrincipalID: "svc_executor",
			Roles:       []string{principal.RoleControlExecutor},
		},
	})
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected control_executor to be allowed")
	}
}

func TestStaticExecuteAuthorizationServiceRejectsAssistantUser(t *testing.T) {
	service := NewStaticExecuteAuthorizationService()

	decision, err := service.Authorize(context.Background(), ExecuteAuthorizationInput{
		ActionName: "mysql.database.create",
		OrderID:    "ord_01",
		Executor: principal.Principal{
			PrincipalID: "u_1002",
			Roles:       []string{principal.RoleAssistantUser},
		},
	})
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected assistant_user to be denied")
	}
}
