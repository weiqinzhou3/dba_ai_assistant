package actionrequest

import (
	"context"
	"testing"

	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appexec "dba_ai_assistant/internal/application/execution"
	"dba_ai_assistant/internal/domain/asset"
	dauth "dba_ai_assistant/internal/domain/authorization"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/domain/risk"
)

func TestServiceSubmitCreatesApprovedOrderWithoutExecuting(t *testing.T) {
	auditRecorder := &recordingAuditService{}
	planner := &stubExecutionPlanner{
		plan: plan.ExecutionPlan{
			PlanID:      "plan_01",
			PlanVersion: 1,
			PlanStatus:  plan.StatusFrozen,
		},
	}

	service := NewService(
		&stubPrincipalResolver{},
		&stubAssetResolver{},
		&stubAuthorizationService{
			decision: dauth.Decision{
				Authorized:       true,
				FinalDecision:    dauth.FinalDecisionAllowNoApproval,
				ApprovalRequired: false,
				RiskLevel:        risk.LevelR1,
			},
		},
		&stubExecuteAuthorizationService{
			decision: appauth.ExecuteAuthorizationDecision{Allowed: true},
		},
		planner,
		auditRecorder,
	)

	result, err := service.Submit(context.Background(), ActionRequestDTO{
		PrincipalID: "u_1001",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
		RequestContext: map[string]any{
			"source": "deep_agent",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if result.Status != order.StatusApproved {
		t.Fatalf("expected approved order, got %s", result.Status)
	}
	if result.ApprovalRequired {
		t.Fatalf("expected approval_required=false")
	}
	if result.TaskID != "" {
		t.Fatalf("expected task_id to stay empty before execute, got %q", result.TaskID)
	}
	if planner.buildCalls != 1 {
		t.Fatalf("expected planner Build to be called once, got %d", planner.buildCalls)
	}
	if !auditRecorder.HasEvent(appaudit.EventRequestAccepted) {
		t.Fatalf("expected REQUEST_ACCEPTED audit event")
	}
	if !auditRecorder.HasEvent(appaudit.EventPlanFrozen) {
		t.Fatalf("expected PLAN_FROZEN audit event")
	}
	if first := auditRecorder.FirstEventType(); first != appaudit.EventRequestAccepted {
		t.Fatalf("expected first audit event to be %s, got %s", appaudit.EventRequestAccepted, first)
	}
}

func TestServiceExecuteApprovedOrderRejectsWaitingApprovalOrder(t *testing.T) {
	service := NewService(
		&stubPrincipalResolver{},
		&stubAssetResolver{},
		&stubAuthorizationService{
			decision: dauth.Decision{
				Authorized:       true,
				FinalDecision:    dauth.FinalDecisionAllowWithApproval,
				ApprovalRequired: true,
				RiskLevel:        risk.LevelR2,
			},
		},
		&stubExecuteAuthorizationService{
			decision: appauth.ExecuteAuthorizationDecision{Allowed: true},
		},
		&stubExecutionPlanner{
			plan: plan.ExecutionPlan{
				PlanID:      "plan_02",
				PlanVersion: 1,
				PlanStatus:  plan.StatusFrozen,
			},
		},
		&recordingAuditService{},
	)

	submitResult, err := service.Submit(context.Background(), ActionRequestDTO{
		PrincipalID: "u_1001",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "prod",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	_, err = service.ExecuteApprovedOrder(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_2001",
		Roles:                    []string{"mysql_operator"},
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "try execute too early",
	})
	if err == nil {
		t.Fatalf("expected approval gate to block execute")
	}
	if code := common.ErrorCode(err); code != common.CodeApprovalRequired {
		t.Fatalf("expected %s, got %s", common.CodeApprovalRequired, code)
	}
}

func TestServiceExecuteApprovedOrderRejectsCallerWithoutExecutePolicy(t *testing.T) {
	service := NewService(
		&stubPrincipalResolver{},
		&stubAssetResolver{},
		&stubAuthorizationService{
			decision: dauth.Decision{
				Authorized:       true,
				FinalDecision:    dauth.FinalDecisionAllowNoApproval,
				ApprovalRequired: false,
				RiskLevel:        risk.LevelR1,
			},
		},
		&stubExecuteAuthorizationService{
			decision: appauth.ExecuteAuthorizationDecision{
				Allowed: false,
				Reasons: []string{"assistant_user cannot execute approved orders"},
			},
		},
		&stubExecutionPlanner{
			plan: plan.ExecutionPlan{
				PlanID:      "plan_03",
				PlanVersion: 1,
				PlanStatus:  plan.StatusFrozen,
			},
		},
		&recordingAuditService{},
	)

	submitResult, err := service.Submit(context.Background(), ActionRequestDTO{
		PrincipalID: "u_1001",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	_, err = service.ExecuteApprovedOrder(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_2002",
		Roles:                    []string{"assistant_user"},
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "unauthorized execute attempt",
	})
	if err == nil {
		t.Fatalf("expected execute policy to deny caller")
	}
	if code := common.ErrorCode(err); code != common.CodeExecutorNotAllowed {
		t.Fatalf("expected %s, got %s", common.CodeExecutorNotAllowed, code)
	}
}

func TestServiceExecuteApprovedOrderFailsClosedWhenExecuteAuthorizationIsMissing(t *testing.T) {
	service := NewService(
		&stubPrincipalResolver{},
		&stubAssetResolver{},
		&stubAuthorizationService{
			decision: dauth.Decision{
				Authorized:       true,
				FinalDecision:    dauth.FinalDecisionAllowNoApproval,
				ApprovalRequired: false,
				RiskLevel:        risk.LevelR1,
			},
		},
		nil,
		&stubExecutionPlanner{
			plan: plan.ExecutionPlan{
				PlanID:      "plan_04",
				PlanVersion: 1,
				PlanStatus:  plan.StatusFrozen,
			},
		},
		&recordingAuditService{},
	)

	submitResult, err := service.Submit(context.Background(), ActionRequestDTO{
		PrincipalID: "u_1001",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	_, err = service.ExecuteApprovedOrder(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_2002",
		Roles:                    []string{"mysql_operator"},
	}, ExecuteOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  "missing execute auth wiring",
	})
	if err == nil {
		t.Fatalf("expected execute auth missing to fail closed")
	}
	if code := common.ErrorCode(err); code != common.CodeSystemInternalError {
		t.Fatalf("expected %s, got %s", common.CodeSystemInternalError, code)
	}
}

func TestServiceGetOrderReturnsTraceIDFromSubmission(t *testing.T) {
	service := NewService(
		&stubPrincipalResolver{},
		&stubAssetResolver{},
		&stubAuthorizationService{
			decision: dauth.Decision{
				Authorized:       true,
				FinalDecision:    dauth.FinalDecisionAllowNoApproval,
				ApprovalRequired: false,
				RiskLevel:        risk.LevelR1,
			},
		},
		&stubExecuteAuthorizationService{
			decision: appauth.ExecuteAuthorizationDecision{Allowed: true},
		},
		&stubExecutionPlanner{
			plan: plan.ExecutionPlan{
				PlanID:      "plan_05",
				PlanVersion: 1,
				PlanStatus:  plan.StatusFrozen,
			},
		},
		&recordingAuditService{},
	)

	submitResult, err := service.Submit(context.Background(), ActionRequestDTO{
		PrincipalID: "u_1001",
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         "order-platform",
			Environment:     "test",
			ServiceInstance: "mysql-order-main",
		},
		Parameters: map[string]any{
			"database_name": "order_center",
		},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	view, err := service.GetOrder(context.Background(), submitResult.OrderID)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if view.TraceID != submitResult.TraceID {
		t.Fatalf("expected trace id %q, got %q", submitResult.TraceID, view.TraceID)
	}
}

type stubPrincipalResolver struct{}

func (s *stubPrincipalResolver) Resolve(_ context.Context, principalID string, authCtx appauth.AuthContext) (principal.Principal, error) {
	roles := []string{"mysql_operator"}
	if len(authCtx.Roles) > 0 {
		roles = append([]string(nil), authCtx.Roles...)
	}
	return principal.Principal{
		PrincipalID: principalID,
		Roles:       roles,
		IsActive:    true,
	}, nil
}

type stubAssetResolver struct{}

func (s *stubAssetResolver) ResolveExact(_ context.Context, _ string, selector asset.Selector) (asset.ResolvedAssetSet, error) {
	return asset.ResolvedAssetSet{
		AssetIDs: []string{"dbt_1001"},
		Assets: []asset.Asset{
			{
				AssetID:         "dbt_1001",
				AssetType:       asset.TypeDatabaseTarget,
				Project:         selector.Project,
				Environment:     selector.Environment,
				ServiceInstance: selector.ServiceInstance,
				CanonicalName:   selector.ServiceInstance,
			},
		},
		MatchedExactly: true,
		AssetType:      asset.TypeDatabaseTarget,
	}, nil
}

type stubAuthorizationService struct {
	decision dauth.Decision
}

func (s *stubAuthorizationService) Evaluate(_ context.Context, _ dauth.Input) (dauth.Decision, error) {
	return s.decision, nil
}

type stubExecuteAuthorizationService struct {
	decision appauth.ExecuteAuthorizationDecision
}

func (s *stubExecuteAuthorizationService) Authorize(_ context.Context, _ appauth.ExecuteAuthorizationInput) (appauth.ExecuteAuthorizationDecision, error) {
	return s.decision, nil
}

type stubExecutionPlanner struct {
	plan       plan.ExecutionPlan
	buildCalls int
}

func (s *stubExecutionPlanner) Build(_ context.Context, _ order.AssistantOrder) (plan.ExecutionPlan, error) {
	s.buildCalls++
	return s.plan, nil
}

func (s *stubExecutionPlanner) Revalidate(_ context.Context, _ order.AssistantOrder, _ plan.ExecutionPlan) (appexec.PlanValidationResult, error) {
	return appexec.PlanValidationResult{
		Valid:  true,
		Status: plan.StatusRevalidated,
	}, nil
}

type recordingAuditService struct {
	events []appaudit.Event
}

func (r *recordingAuditService) AppendEvent(_ context.Context, event appaudit.Event) error {
	r.events = append(r.events, event)
	return nil
}

func (r *recordingAuditService) ListEventsByRequestID(_ context.Context, _ string) ([]appaudit.Event, error) {
	return append([]appaudit.Event(nil), r.events...), nil
}

func (r *recordingAuditService) GetViewByRequestID(_ context.Context, requestID string) (appaudit.LedgerView, error) {
	return appaudit.LedgerView{
		RequestID: requestID,
		Events:    append([]appaudit.Event(nil), r.events...),
	}, nil
}

func (r *recordingAuditService) HasEvent(eventType appaudit.EventType) bool {
	for _, event := range r.events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}

func (r *recordingAuditService) FirstEventType() appaudit.EventType {
	if len(r.events) == 0 {
		return ""
	}
	return r.events[0].EventType
}
