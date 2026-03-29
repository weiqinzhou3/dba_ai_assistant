package skill

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dba_ai_assistant/internal/adapters/dbnative"
	"dba_ai_assistant/internal/api"
	appaction "dba_ai_assistant/internal/application/actionrequest"
	appapproval "dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
	"dba_ai_assistant/internal/persistence"
)

func TestServiceRequestMySQLDatabaseCreateUsesRealControlAPIForNonProdAutoChain(t *testing.T) {
	httpServer := newSkillIntegrationServer(t)
	server := httptest.NewServer(httpServer)
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
		AutoExecute:     true,
	})
	if err != nil {
		t.Fatalf("RequestMySQLDatabaseCreate returned error: %v", err)
	}

	if !output.AutoExecuteTriggered {
		t.Fatalf("expected auto execute to be triggered")
	}
	if output.AutoExecuteResult == nil {
		t.Fatalf("expected auto execute result")
	}
	if output.OrderStatus != order.StatusSucceeded {
		t.Fatalf("expected final order status SUCCEEDED, got %s", output.OrderStatus)
	}
	if output.TaskID == "" {
		t.Fatalf("expected task id after real auto execute")
	}
}

func TestServiceRequestMySQLDatabaseCreateUsesRealControlAPIForProdApprovalBoundary(t *testing.T) {
	httpServer := newSkillIntegrationServer(t)
	server := httptest.NewServer(httpServer)
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

	if output.AutoExecuteTriggered {
		t.Fatalf("expected no auto execute for approval-required order")
	}
	if !output.ApprovalRequired {
		t.Fatalf("expected approval_required=true")
	}
	if output.OrderStatus != order.StatusWaitingApproval {
		t.Fatalf("expected WAITING_APPROVAL, got %s", output.OrderStatus)
	}
	if output.TaskID != "" {
		t.Fatalf("expected no task id before explicit execute")
	}
}

func newSkillIntegrationServer(t *testing.T) http.Handler {
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
		Router: appexec.NewStaticExecutionRouter(dbnative.New(dbnative.Dependencies{
			ConnectionResolver: skillStaticConnectionResolver{
				config: dbnative.ConnectionConfig{Engine: "mysql", DSN: "mysql-test"},
			},
			MySQL: &skillFakeMySQLAdmin{
				existing: map[string]bool{},
			},
			Now: skillFixedNow,
		})),
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

	return api.NewServer(api.Dependencies{
		ActionRequests: actionService,
		Approvals:      approvalService,
		Audit:          auditService,
		Evidence:       evidenceService,
	})
}

func skillFixedNow() time.Time {
	return time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
}

type skillStaticConnectionResolver struct {
	config dbnative.ConnectionConfig
	err    error
}

func (r skillStaticConnectionResolver) Resolve(_ context.Context, _ string) (dbnative.ConnectionConfig, error) {
	if r.err != nil {
		return dbnative.ConnectionConfig{}, r.err
	}
	return r.config, nil
}

type skillFakeMySQLAdmin struct {
	existing map[string]bool
}

func (a *skillFakeMySQLAdmin) Ping(_ context.Context, _ dbnative.ConnectionConfig) error {
	return nil
}

func (a *skillFakeMySQLAdmin) DatabaseExists(_ context.Context, _ dbnative.ConnectionConfig, databaseName string) (bool, error) {
	return a.existing[databaseName], nil
}

func (a *skillFakeMySQLAdmin) CreateDatabase(_ context.Context, _ dbnative.ConnectionConfig, databaseName string) error {
	a.existing[databaseName] = true
	return nil
}
