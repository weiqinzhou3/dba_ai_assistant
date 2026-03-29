package approval_test

import (
	"context"
	"testing"
	"time"

	appapproval "dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
	"dba_ai_assistant/internal/persistence"
)

func TestServiceApproveTransitionUpdatesOrderAndAudit(t *testing.T) {
	ctx := context.Background()
	store := persistence.NewMemoryStore()
	auditService := appaudit.NewMemoryService(store)
	service := appapproval.NewService(appapproval.Dependencies{
		Orders:    store,
		Plans:     store,
		Approvals: store,
		Policies:  store,
		Audit:     auditService,
	})

	mustSaveApprovalPolicy(t, ctx, store)
	ord := mustSaveWaitingApprovalOrder(t, ctx, store)
	if _, err := service.Create(ctx, ord, appapproval.CreateInput{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	state, err := service.Decide(ctx, ord.OrderID, appapproval.DecisionInput{
		ApproverID: "u_approver",
		Decision:   appapproval.DecisionApprove,
		Comment:    "approved",
	})
	if err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}

	if state.ApprovalStatus != order.ApprovalStatusApproved {
		t.Fatalf("expected approval status APPROVED, got %s", state.ApprovalStatus)
	}
	if state.Status != order.StatusApproved {
		t.Fatalf("expected order status APPROVED, got %s", state.Status)
	}
	if len(state.Records) != 1 {
		t.Fatalf("expected one approval record, got %d", len(state.Records))
	}

	persistedOrder, err := store.GetOrder(ctx, ord.OrderID)
	if err != nil {
		t.Fatalf("Get order returned error: %v", err)
	}
	if persistedOrder.Status != order.StatusApproved {
		t.Fatalf("expected persisted order status APPROVED, got %s", persistedOrder.Status)
	}
	if persistedOrder.ApprovalStatus != order.ApprovalStatusApproved {
		t.Fatalf("expected persisted approval status APPROVED, got %s", persistedOrder.ApprovalStatus)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, ord.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventApprovalCreated)
	assertAuditEvent(t, ledger, appaudit.EventApprovalApproved)
}

func TestServiceRejectsSelfApproval(t *testing.T) {
	ctx := context.Background()
	store := persistence.NewMemoryStore()
	auditService := appaudit.NewMemoryService(store)
	service := appapproval.NewService(appapproval.Dependencies{
		Orders:    store,
		Plans:     store,
		Approvals: store,
		Policies:  store,
		Audit:     auditService,
	})

	mustSaveApprovalPolicy(t, ctx, store)
	ord := mustSaveWaitingApprovalOrder(t, ctx, store)
	if _, err := service.Create(ctx, ord, appapproval.CreateInput{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err := service.Decide(ctx, ord.OrderID, appapproval.DecisionInput{
		ApproverID: ord.CreatedBy,
		Decision:   appapproval.DecisionApprove,
	})
	if err == nil {
		t.Fatalf("expected self approval to be rejected")
	}
}

func TestServiceExpireStaleApprovalsTransitionsOrderToExpired(t *testing.T) {
	ctx := context.Background()
	store := persistence.NewMemoryStore()
	auditService := appaudit.NewMemoryService(store)
	service := appapproval.NewService(appapproval.Dependencies{
		Orders:    store,
		Plans:     store,
		Approvals: store,
		Policies:  store,
		Audit:     auditService,
	})

	mustSaveApprovalPolicy(t, ctx, store)
	ord := mustSaveWaitingApprovalOrder(t, ctx, store)
	state, err := service.Create(ctx, ord, appapproval.CreateInput{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	state.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	if err := store.SaveApprovalState(ctx, state); err != nil {
		t.Fatalf("Save approval state returned error: %v", err)
	}

	results, err := service.ExpireStaleApprovals(ctx, time.Now().UTC(), 10)
	if err != nil {
		t.Fatalf("ExpireStaleApprovals returned error: %v", err)
	}
	if len(results) != 1 || !results[0].Expired {
		t.Fatalf("expected one expired result, got %+v", results)
	}

	persistedOrder, err := store.GetOrder(ctx, ord.OrderID)
	if err != nil {
		t.Fatalf("Get order returned error: %v", err)
	}
	if persistedOrder.Status != order.StatusExpired {
		t.Fatalf("expected order status EXPIRED, got %s", persistedOrder.Status)
	}
	if persistedOrder.ApprovalStatus != order.ApprovalStatusExpired {
		t.Fatalf("expected approval status EXPIRED, got %s", persistedOrder.ApprovalStatus)
	}

	ledger, err := auditService.GetViewByRequestID(ctx, ord.RequestID)
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	assertAuditEvent(t, ledger, appaudit.EventApprovalExpired)
}

func mustSaveApprovalPolicy(t *testing.T, ctx context.Context, store *persistence.MemoryStore) {
	t.Helper()
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
}

func mustSaveWaitingApprovalOrder(t *testing.T, ctx context.Context, store *persistence.MemoryStore) order.AssistantOrder {
	t.Helper()
	ord := order.AssistantOrder{
		OrderID:          "ord_approval_01",
		RequestID:        "req_approval_01",
		ActionName:       "mysql.database.create",
		RiskLevel:        risk.LevelR2,
		ApprovalRequired: true,
		ApprovalStatus:   order.ApprovalStatusWaitingApproval,
		Status:           order.StatusWaitingApproval,
		PlanID:           "plan_approval_01",
		PlanVersion:      1,
		CreatedBy:        "u_requester",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := store.SaveOrder(ctx, ord); err != nil {
		t.Fatalf("failed to save order: %v", err)
	}
	return ord
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
