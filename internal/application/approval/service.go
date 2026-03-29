package approval

import (
	"context"
	"time"

	appaudit "dba_ai_assistant/internal/application/audit"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/policy"
)

type orderRepository interface {
	SaveOrder(ctx context.Context, ord order.AssistantOrder) error
	GetOrder(ctx context.Context, orderID string) (order.AssistantOrder, error)
}

type approvalRepository interface {
	SaveApprovalState(ctx context.Context, state State) error
	GetApprovalStateByOrderID(ctx context.Context, orderID string) (State, error)
	ListWaitingApprovalStates(ctx context.Context, limit int) ([]State, error)
}

type approvalPolicyRepository interface {
	ListApprovalPoliciesByAction(ctx context.Context, actionName string) ([]policy.ApprovalPolicy, error)
}

type Dependencies struct {
	Orders    orderRepository
	Plans     any
	Approvals approvalRepository
	Policies  approvalPolicyRepository
	Audit     appaudit.Service
}

type service struct {
	orders    orderRepository
	approvals approvalRepository
	policies  approvalPolicyRepository
	audit     appaudit.Service
}

func NewService(deps Dependencies) Service {
	return &service{
		orders:    deps.Orders,
		approvals: deps.Approvals,
		policies:  deps.Policies,
		audit:     deps.Audit,
	}
}

func (s *service) Create(ctx context.Context, ord order.AssistantOrder, input CreateInput) (State, error) {
	if !ord.ApprovalRequired {
		state := State{
			OrderID:        ord.OrderID,
			ApprovalStatus: ord.ApprovalStatus,
			Status:         ord.Status,
		}
		if err := s.approvals.SaveApprovalState(ctx, state); err != nil {
			return State{}, err
		}
		return state, nil
	}

	policyValue, err := s.resolveApprovalPolicy(ctx, ord, input.ApprovalPolicyRef)
	if err != nil {
		return State{}, err
	}

	now := time.Now().UTC()
	state := State{
		ApprovalPolicyRef: policyValue.PolicyID,
		OrderID:           ord.OrderID,
		ApprovalStatus:    order.ApprovalStatusWaitingApproval,
		Status:            order.StatusWaitingApproval,
		CreatedAt:         now,
		ExpiresAt:         now.Add(policyValue.TTL),
	}
	if err := s.approvals.SaveApprovalState(ctx, state); err != nil {
		return State{}, err
	}

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType: appaudit.EventApprovalCreated,
		RequestID: ord.RequestID,
		OrderID:   ord.OrderID,
		TraceID:   "",
		Metadata: map[string]any{
			"approval_status": string(order.ApprovalStatusWaitingApproval),
			"order_status":    string(order.StatusWaitingApproval),
			"approval_policy": policyValue.PolicyID,
		},
	})

	return state, nil
}

func (s *service) Decide(ctx context.Context, orderID string, decision DecisionInput) (State, error) {
	ord, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return State{}, err
	}
	state, err := s.approvals.GetApprovalStateByOrderID(ctx, orderID)
	if err != nil {
		return State{}, err
	}

	if state.ApprovalStatus != order.ApprovalStatusWaitingApproval || ord.Status != order.StatusWaitingApproval {
		return State{}, common.NewError(common.CodeOrderNotExecutable, "approval state does not allow decision", map[string]any{"order_id": orderID})
	}
	if decision.ApproverID == ord.CreatedBy {
		return State{}, common.NewError(common.CodeSelfApprovalForbidden, "self approval is forbidden", map[string]any{"order_id": orderID})
	}

	record := Record{
		ApprovalID:  "apr_" + orderID,
		OrderID:     orderID,
		ApproverID:  decision.ApproverID,
		Decision:    decision.Decision,
		Comment:     decision.Comment,
		RiskLevel:   ord.RiskLevel,
		PlanID:      ord.PlanID,
		PlanVersion: ord.PlanVersion,
		ApprovedAt:  time.Now().UTC(),
	}
	state.Records = append(state.Records, record)

	eventType := appaudit.EventApprovalRejected
	if decision.Decision == DecisionApprove {
		state.ApprovalStatus = order.ApprovalStatusApproved
		state.Status = order.StatusApproved
		ord.ApprovalStatus = order.ApprovalStatusApproved
		ord.Status = order.StatusApproved
		eventType = appaudit.EventApprovalApproved
	} else {
		state.ApprovalStatus = order.ApprovalStatusRejected
		state.Status = order.StatusRejected
		ord.ApprovalStatus = order.ApprovalStatusRejected
		ord.Status = order.StatusRejected
	}
	ord.UpdatedAt = time.Now().UTC()

	if err := s.orders.SaveOrder(ctx, ord); err != nil {
		return State{}, err
	}
	if err := s.approvals.SaveApprovalState(ctx, state); err != nil {
		return State{}, err
	}

	_ = s.audit.AppendEvent(ctx, appaudit.Event{
		EventType:       eventType,
		RequestID:       ord.RequestID,
		OrderID:         ord.OrderID,
		ApprovalActorID: decision.ApproverID,
		Metadata: map[string]any{
			"approval_status": string(state.ApprovalStatus),
			"order_status":    string(state.Status),
		},
	})

	return state, nil
}

func (s *service) Get(ctx context.Context, orderID string) (State, error) {
	return s.approvals.GetApprovalStateByOrderID(ctx, orderID)
}

func (s *service) ExpireStaleApprovals(ctx context.Context, now time.Time, limit int) ([]ExpiryResult, error) {
	waitingStates, err := s.approvals.ListWaitingApprovalStates(ctx, limit)
	if err != nil {
		return nil, err
	}

	results := make([]ExpiryResult, 0, len(waitingStates))
	for _, state := range waitingStates {
		if state.ExpiresAt.IsZero() || now.Before(state.ExpiresAt) {
			continue
		}

		ord, err := s.orders.GetOrder(ctx, state.OrderID)
		if err != nil {
			return nil, err
		}

		state.ApprovalStatus = order.ApprovalStatusExpired
		state.Status = order.StatusExpired
		ord.ApprovalStatus = order.ApprovalStatusExpired
		ord.Status = order.StatusExpired
		ord.UpdatedAt = now

		if err := s.orders.SaveOrder(ctx, ord); err != nil {
			return nil, err
		}
		if err := s.approvals.SaveApprovalState(ctx, state); err != nil {
			return nil, err
		}

		_ = s.audit.AppendEvent(ctx, appaudit.Event{
			EventType: appaudit.EventApprovalExpired,
			RequestID: ord.RequestID,
			OrderID:   ord.OrderID,
			Metadata: map[string]any{
				"approval_status": string(state.ApprovalStatus),
				"order_status":    string(state.Status),
			},
		})

		results = append(results, ExpiryResult{
			OrderID: state.OrderID,
			Expired: true,
		})
	}

	return results, nil
}

func (s *service) resolveApprovalPolicy(ctx context.Context, ord order.AssistantOrder, requestedPolicyRef string) (policy.ApprovalPolicy, error) {
	policies, err := s.policies.ListApprovalPoliciesByAction(ctx, ord.ActionName)
	if err != nil {
		return policy.ApprovalPolicy{}, err
	}
	for _, candidate := range policies {
		if !candidate.Enabled {
			continue
		}
		if requestedPolicyRef != "" && candidate.PolicyID != requestedPolicyRef {
			continue
		}
		if len(candidate.RiskLevels) == 0 || contains(candidate.RiskLevels, string(ord.RiskLevel)) {
			return candidate, nil
		}
	}
	return policy.ApprovalPolicy{}, common.NewError(common.CodeSystemInternalError, "approval policy not found", map[string]any{
		"order_id":    ord.OrderID,
		"action_name": ord.ActionName,
		"risk_level":  ord.RiskLevel,
	})
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
