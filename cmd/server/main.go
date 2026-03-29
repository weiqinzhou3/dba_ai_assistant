package main

import (
	"log"
	"net/http"

	"dba_ai_assistant/internal/api"
	appaction "dba_ai_assistant/internal/application/actionrequest"
	"dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	appexec "dba_ai_assistant/internal/application/execution"
)

func main() {
	auditService := appaudit.NewMemoryService()
	evidenceService := appevidence.NewMemoryService()
	actionService := appaction.NewService(
		appauth.NewStaticPrincipalResolver(),
		appauth.NewInMemoryExactAssetResolver(appauth.StaticManagedAssets()),
		appauth.NewAuthorizationService(
			appauth.NewStaticPolicyEngine(),
			appauth.NewStaticRiskEngine(),
		),
		appauth.NewStaticExecuteAuthorizationService(),
		appexec.NewStaticExecutionPlanner(),
		auditService,
	)

	server := api.NewServer(api.Dependencies{
		ActionRequests: actionService,
		Approvals:      approval.NewNoopService(),
		Audit:          auditService,
		Evidence:       evidenceService,
	})

	log.Printf("listening on :8080")
	if err := http.ListenAndServe(":8080", server); err != nil {
		log.Fatal(err)
	}
}
