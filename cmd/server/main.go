package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"dba_ai_assistant/internal/api"
	appaction "dba_ai_assistant/internal/application/actionrequest"
	"dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
	"dba_ai_assistant/internal/persistence"
)

func main() {
	store := persistence.NewMemoryStore()
	if err := seedPolicies(store); err != nil {
		log.Fatal(err)
	}

	auditService := appaudit.NewMemoryService(store)
	evidenceService := appevidence.NewMemoryService(store)
	approvalService := approval.NewService(approval.Dependencies{
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

	server := api.NewServer(api.Dependencies{
		ActionRequests: actionService,
		Approvals:      approvalService,
		Audit:          auditService,
		Evidence:       evidenceService,
	})

	log.Printf("listening on :8080")
	if err := http.ListenAndServe(":8080", server); err != nil {
		log.Fatal(err)
	}
}

func seedPolicies(store *persistence.MemoryStore) error {
	return store.SaveApprovalPolicy(context.Background(), policy.ApprovalPolicy{
		PolicyID:           "approval_policy_prod_r2",
		ActionName:         "mysql.database.create",
		RiskLevels:         []string{string(risk.LevelR2)},
		ApproverRoles:      []string{principal.RoleProdApprover, principal.RolePlatformAdmin},
		MinApproverCount:   1,
		ForbidSelfApproval: true,
		TTL:                30 * time.Minute,
		Enabled:            true,
	})
}
